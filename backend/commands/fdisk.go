package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "os"
    "sort"
    "strconv"
    "strings"
)

func Fdisk(params map[string]string) string {
    path, okPath := params["-path"]
    name, okName := params["-name"]
    sizeStr, okSize := params["-size"]
    if !okPath || !okName || !okSize {
        return "Error: parámetros obligatorios faltantes (-path, -name, -size)"
    }

    var err error
    path, err = LimpiarRuta(path)
    if err != nil {
        return fmt.Sprintf("Error en la ruta: %v", err)
    }

    unit := "K"
    if v, ok := params["-unit"]; ok {
        unit = strings.ToUpper(v)
        if unit != "B" && unit != "K" && unit != "M" {
            return "Error: unidad inválida (use B, K o M)"
        }
    }

    ptype := "P"
    if v, ok := params["-type"]; ok {
        ptype = strings.ToUpper(v)
        if ptype != "P" && ptype != "E" && ptype != "L" {
            return "Error: type inválido (use P, E o L)"
        }
    }

    fit := "WF"
    if v, ok := params["-fit"]; ok {
        fit = strings.ToUpper(v)
        if fit != "FF" && fit != "BF" && fit != "WF" {
            return "Error: fit inválido (use FF, BF o WF)"
        }
    }

    sizeVal, err := strconv.ParseInt(sizeStr, 10, 64)
    if err != nil || sizeVal <= 0 {
        return "Error: -size debe ser un entero positivo"
    }
    var reqBytes int64
    switch unit {
    case "B":
        reqBytes = sizeVal
    case "K":
        reqBytes = sizeVal * 1024
    case "M":
        reqBytes = sizeVal * 1024 * 1024
    }

    file, err := os.OpenFile(path, os.O_RDWR, 0666)
    if err != nil {
        return fmt.Sprintf("Error abriendo disco: %v", err)
    }
    defer file.Close()

    mbr, err := structs.LeerMBR(path)
    if err != nil {
        return fmt.Sprintf("Error leyendo MBR: %v", err)
    }

    if nombreDuplicado(file, &mbr, name) {
        return "Error: el nombre de partición ya existe en el disco"
    }

    if ptype == "L" {
        extIdx := indiceExtendida(&mbr)
        if extIdx == -1 {
            return "Error: no existe partición extendida para crear una lógica"
        }
        return crearLogica(file, &mbr.Mbr_partitions[extIdx], name, reqBytes, fit)
    }

    if ptype == "E" && existeExtendida(&mbr) {
        return "Error: ya existe una partición extendida en el disco"
    }

    slot := primerSlotLibre(&mbr)
    if slot == -1 {
        return "Error: límite de 4 particiones primarias/extendidas alcanzado"
    }

    start, err := ubicarPrimariaExtendida(&mbr, reqBytes, fit)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    var fitCh byte
    switch fit {
    case "FF":
        fitCh = 'F'
    case "BF":
        fitCh = 'B'
    case "WF":
        fitCh = 'W'
    }

    mbr.Mbr_partitions[slot].Part_status = 1
    mbr.Mbr_partitions[slot].Part_type = ptype[0]
    mbr.Mbr_partitions[slot].Part_fit = fitCh
    mbr.Mbr_partitions[slot].Part_start = int32(start)
    mbr.Mbr_partitions[slot].Part_s = int32(reqBytes)
    mbr.Mbr_partitions[slot].Part_correlative = -1

    for i := range mbr.Mbr_partitions[slot].Part_name {
        mbr.Mbr_partitions[slot].Part_name[i] = 0
    }
    copy(mbr.Mbr_partitions[slot].Part_name[:], name)

    if err := escribirMBR(file, &mbr); err != nil {
        return fmt.Sprintf("Error escribiendo MBR: %v", err)
    }

    if ptype == "E" {
        var ebr structs.EBR
        ebr.Part_mount = 0
        ebr.Part_fit = 0
        ebr.Part_start = 0
        ebr.Part_s = 0
        ebr.Part_next = -1
        for i := range ebr.Part_name {
            ebr.Part_name[i] = 0
        }
        if _, err := file.Seek(int64(mbr.Mbr_partitions[slot].Part_start), 0); err != nil {
            return fmt.Sprintf("Error posicionando EBR: %v", err)
        }
        if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
            return fmt.Sprintf("Error inicializando EBR: %v", err)
        }
    }

    return fmt.Sprintf("Partición creada: %s", name)
}

func escribirMBR(file *os.File, mbr *structs.MBR) error {
    if _, err := file.Seek(0, 0); err != nil {
        return err
    }
    return binary.Write(file, binary.LittleEndian, mbr)
}

func primerSlotLibre(mbr *structs.MBR) int {
    for i := 0; i < 4; i++ {
        if mbr.Mbr_partitions[i].Part_status == 0 {
            return i
        }
    }
    return -1
}

func existeExtendida(mbr *structs.MBR) bool {
    for i := 0; i < 4; i++ {
        if mbr.Mbr_partitions[i].Part_status == 1 && mbr.Mbr_partitions[i].Part_type == 'E' {
            return true
        }
    }
    return false
}

func indiceExtendida(mbr *structs.MBR) int {
    for i := 0; i < 4; i++ {
        if mbr.Mbr_partitions[i].Part_status == 1 && mbr.Mbr_partitions[i].Part_type == 'E' {
            return i
        }
    }
    return -1
}

func nombreDuplicado(file *os.File, mbr *structs.MBR, name string) bool {
    for i := 0; i < 4; i++ {
        p := mbr.Mbr_partitions[i]
        if p.Part_status == 1 && bytesToString(p.Part_name[:]) == name {
            return true
        }
        if p.Part_status == 1 && p.Part_type == 'E' {
            if nombreDuplicadoLogicas(file, int64(p.Part_start), name) {
                return true
            }
        }
    }
    return false
}

func nombreDuplicadoLogicas(file *os.File, extStart int64, name string) bool {
    var ebr structs.EBR
    pos := extStart
    if _, err := file.Seek(pos, 0); err != nil {
        return false
    }
    if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
        return false
    }

    if ebr.Part_mount == 0 && (ebr.Part_next == -1 || ebr.Part_next == 0) && ebr.Part_s == 0 {
        return false
    }

    for {
        if ebr.Part_mount == 1 {
            if bytesToString(ebr.Part_name[:]) == name {
                return true
            }
        }
        if ebr.Part_next == -1 || ebr.Part_next == 0 {
            break
        }
        pos = int64(ebr.Part_next)
        if _, err := file.Seek(pos, 0); err != nil {
            break
        }
        if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
            break
        }
    }
    return false
}

func ubicarPrimariaExtendida(mbr *structs.MBR, req int64, fit string) (int64, error) {
    type seg struct{ start, size int64 }
    var ocupadas []seg
    for i := 0; i < 4; i++ {
        p := mbr.Mbr_partitions[i]
        if p.Part_status == 1 {
            ocupadas = append(ocupadas, seg{int64(p.Part_start), int64(p.Part_s)})
        }
    }
    sort.Slice(ocupadas, func(i, j int) bool { return ocupadas[i].start < ocupadas[j].start })
    header := int64(binary.Size(*mbr))

    var gaps []seg
    cur := header
    for _, s := range ocupadas {
        if s.start > cur {
            gap := seg{cur, s.start - cur}
            gaps = append(gaps, gap)
        }
        cur = s.start + s.size
    }
    if int64(mbr.Mbr_tamano) > cur {
        gap := seg{cur, int64(mbr.Mbr_tamano) - cur}
        gaps = append(gaps, gap)
    }

    if len(gaps) == 0 {
        return 0, fmt.Errorf("No hay espacio disponible")
    }

    switch fit {
    case "FF":
        for _, g := range gaps {
            if g.size >= req {
                return g.start, nil
            }
        }
    case "BF":
        bestIdx := -1
        for i, g := range gaps {
            if g.size >= req {
                if bestIdx == -1 || g.size < gaps[bestIdx].size {
                    bestIdx = i
                }
            }
        }
        if bestIdx != -1 {
            return gaps[bestIdx].start, nil
        }
    case "WF":
        worstIdx := -1
        for i, g := range gaps {
            if g.size >= req {
                if worstIdx == -1 || g.size > gaps[worstIdx].size {
                    worstIdx = i
                }
            }
        }
        if worstIdx != -1 {
            return gaps[worstIdx].start, nil
        }
    }
    return 0, fmt.Errorf("No hay un segmento libre con tamaño suficiente")
}

func crearLogica(file *os.File, ext *structs.Partition, name string, reqBytes int64, fit string) string {

    var ebr structs.EBR
    ebrSize := int64(binary.Size(ebr))
    extStart := int64(ext.Part_start)
    extEnd := extStart + int64(ext.Part_s)

    if _, err := file.Seek(extStart, 0); err != nil {
        return fmt.Sprintf("Error posicionando EBR: %v", err)
    }
    if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
        return fmt.Sprintf("Error leyendo EBR: %v", err)
    }

    if ebr.Part_next == 0 {
        ebr.Part_next = -1
    }

    if ebr.Part_mount == 0 && ebr.Part_next == -1 && ebr.Part_s == 0 && ebr.Part_start == 0 {
        firstDataStart := extStart + ebrSize
        if firstDataStart+reqBytes > extEnd {
            return "Error: no hay espacio en la partición extendida"
        }

        ebr.Part_mount = 1
        ebr.Part_fit = fit[0]
        ebr.Part_start = int32(firstDataStart)
        ebr.Part_s = int32(reqBytes)
        ebr.Part_next = -1

        for i := range ebr.Part_name {
            ebr.Part_name[i] = 0
        }
        copy(ebr.Part_name[:], name)

        if _, err := file.Seek(extStart, 0); err != nil {
            return fmt.Sprintf("Error posicionando EBR: %v", err)
        }
        if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
            return fmt.Sprintf("Error escribiendo EBR: %v", err)
        }
        return fmt.Sprintf("Partición lógica creada: %s", name)
    }

    pos := extStart
    ebrCount := 0

    for {
        ebrCount++

        if ebr.Part_next == -1 || ebr.Part_next == 0 {

            currentDataEnd := int64(ebr.Part_start) + int64(ebr.Part_s)
            newEBRPos := currentDataEnd

            if newEBRPos < extStart {
                return "Error: cálculo de posición de EBR inválido (underflow)"
            }
            if newEBRPos+ebrSize+reqBytes > extEnd {
                return "Error: no hay espacio en la partición extendida"
            }

            ebr.Part_next = int32(newEBRPos)
            if _, err := file.Seek(pos, 0); err != nil {
                return fmt.Sprintf("Error posicionando EBR previo: %v", err)
            }
            if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
                return fmt.Sprintf("Error actualizando EBR previo: %v", err)
            }

            var newEBR structs.EBR
            newEBR.Part_mount = 1
            newEBR.Part_fit = fit[0]
            newEBR.Part_start = int32(newEBRPos + ebrSize)
            newEBR.Part_s = int32(reqBytes)
            newEBR.Part_next = -1
            for i := range newEBR.Part_name {
                newEBR.Part_name[i] = 0
            }
            copy(newEBR.Part_name[:], name)

            if _, err := file.Seek(newEBRPos, 0); err != nil {
                return fmt.Sprintf("Error posicionando nuevo EBR: %v", err)
            }
            if err := binary.Write(file, binary.LittleEndian, &newEBR); err != nil {
                return fmt.Sprintf("Error escribiendo nuevo EBR: %v", err)
            }
            return fmt.Sprintf("Partición lógica creada: %s", name)
        }

        pos = int64(ebr.Part_next)
        if _, err := file.Seek(pos, 0); err != nil {
            return fmt.Sprintf("Error recorriendo EBR: %v", err)
        }
        if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
            return fmt.Sprintf("Error leyendo EBR: %v", err)
        }
        if ebr.Part_next == 0 {
            ebr.Part_next = -1
        }
    }
}

func bytesToString(b []byte) string {
    n := 0
    for i := range b {
        if b[i] == 0 {
            break
        }
        n++
    }
    return string(b[:n])
}
