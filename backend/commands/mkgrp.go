package commands

import (
    "backend/structs"
    "bytes"
    "encoding/binary"
    "fmt"
    "os"
    "strconv"
    "strings"
)

func Mkgrp(args map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    if usuarioActual.Username != "root" {
        return "Error: Solo el usuario root puede ejecutar mkgrp."
    }

    name, ok := args["-name"]
    if !ok || strings.TrimSpace(name) == "" {
        return "Error: Falta el parámetro obligatorio -name."
    }

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    contenidoActual, err := LeerArchivoUsersTXT(disk, sb)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    if GrupoExiste(contenidoActual, name) {
        return "Error: El grupo '" + name + "' ya existe."
    }

    nuevoID := ObtenerSiguienteIDGrupo(contenidoActual)

    contenidoActual = nuevoContenidoSeguro(contenidoActual)
    nuevaLinea := fmt.Sprintf("%d,G,%s\n", nuevoID, name)

    if err := EscribirUsersTxt(usuarioActual.PartitionID, contenidoActual+nuevaLinea); err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Grupo '%s' creado exitosamente con ID %d", name, nuevoID)
}

func EscribirUsersTxt(partitionID string, contenido string) error {
   disk, sb, err := CargarSistemaEXT2(partitionID)
    if err != nil {
        return err
    }
    defer disk.Close()

    inodeOffset := int64(sb.S_inode_start) + int64(usersInodeIndex)*int64(sb.S_inode_s)

    var ino structs.Inodo
    if _, err := disk.Seek(inodeOffset, 0); err != nil {
        return fmt.Errorf("Posicionar inodo users.txt: %v", err)
    }
    if err := binary.Read(disk, binary.LittleEndian, &ino); err != nil {
        return fmt.Errorf("Leer inodo users.txt: %v", err)
    }

    data := []byte(contenido)
    blkDataSize := len(structs.BArchivo{}.B_content)
    if blkDataSize <= 0 {
        return fmt.Errorf("Tamaño de BArchivo.B_content inválido")
    }
    need := (len(data) + blkDataSize - 1) / blkDataSize

    assignedIdxs := make([]int, 0, len(ino.I_block))
    assignedBlks := make([]int32, 0, len(ino.I_block))
    for i := 0; i < len(ino.I_block); i++ {
        if ino.I_block[i] != -1 {
            assignedIdxs = append(assignedIdxs, i)
            assignedBlks = append(assignedBlks, ino.I_block[i])
        }
    }

    bmCount := int(sb.S_blocks_count)
    bm := make([]byte, bmCount)
    if _, err := disk.ReadAt(bm, int64(sb.S_bm_block_start)); err != nil {
        return fmt.Errorf("Leer bitmap de bloques: %v", err)
    }

    allocated := 0
    if need > len(assignedBlks) {
        falta := need - len(assignedBlks)
        for bi := 0; bi < bmCount && allocated < falta; bi++ {
            if bm[bi] == 0 {
                bm[bi] = 1
                puesto := false
                for ii := 0; ii < len(ino.I_block); ii++ {
                    if ino.I_block[ii] == -1 {
                        ino.I_block[ii] = int32(bi)
                        assignedIdxs = append(assignedIdxs, ii)
                        assignedBlks = append(assignedBlks, int32(bi))
                        puesto = true
                        break
                    }
                }
                if !puesto {
                    return fmt.Errorf("users.txt necesita mas punteros de bloque de los disponibles en el inodo")
                }
                allocated++
            }
        }
        if allocated != falta {
            return fmt.Errorf("No hay bloques libres suficientes para users.txt (necesita %d, asigno %d)", falta, allocated)
        }
    }

    freed := 0
    if need < len(assignedBlks) {
        exceso := len(assignedBlks) - need
        for k := 0; k < exceso; k++ {
            idxInInode := assignedIdxs[len(assignedIdxs)-1]
            blkNum := assignedBlks[len(assignedBlks)-1]
            if blkNum >= 0 && int(blkNum) < bmCount {
                bm[blkNum] = 0
                freed++
            }
            ino.I_block[idxInInode] = -1
            assignedIdxs = assignedIdxs[:len(assignedIdxs)-1]
            assignedBlks = assignedBlks[:len(assignedBlks)-1]
        }
    }

    if allocated > 0 || freed > 0 {
        if _, err := disk.WriteAt(bm, int64(sb.S_bm_block_start)); err != nil {
            return fmt.Errorf("Escribir bitmap de bloques: %v", err)
        }
        sb.S_free_blocks_count = sb.S_free_blocks_count - int32(allocated) + int32(freed)
        if err := escribirSuperBloque(disk, sb); err != nil {
            return err
        }
    }

    written := 0
    for i := 0; i < len(ino.I_block) && written < len(data); i++ {
        blkNum := ino.I_block[i]
        if blkNum == -1 {
            continue
        }
        var b structs.BArchivo
        for j := range b.B_content {
            b.B_content[j] = 0
        }
        end := written + blkDataSize
        if end > len(data) {
            end = len(data)
        }
        copy(b.B_content[:], data[written:end])
        written = end

        blockOffset := int64(sb.S_block_start) + int64(blkNum)*int64(sb.S_block_s)
        var buf bytes.Buffer
        if err := binary.Write(&buf, binary.LittleEndian, &b); err != nil {
            return fmt.Errorf("Serializar bloque: %v", err)
        }
        if _, err := disk.WriteAt(buf.Bytes(), blockOffset); err != nil {
            return fmt.Errorf("Escribir bloque users.txt: %v", err)
        }
    }

    ino.I_s = int32(len(data))
    var ibuf bytes.Buffer
    if err := binary.Write(&ibuf, binary.LittleEndian, &ino); err != nil {
        return fmt.Errorf("Serializar inodo: %v", err)
    }
    if _, err := disk.WriteAt(ibuf.Bytes(), inodeOffset); err != nil {
        return fmt.Errorf("Escribir inodo users.txt: %v", err)
    }

    return nil
}

func nuevoContenidoSeguro(s string) string {
    s = strings.ReplaceAll(s, "\r", "")
    if s == "" {
        return s
    }
    if !strings.HasSuffix(s, "\n") {
        s += "\n"
    }
    return s
}

func GrupoExiste(contenido string, grupo string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, linea := range lineas {
        linea = strings.TrimSpace(linea)
        if linea == "" || strings.HasPrefix(linea, "#") {
            continue
        }
        campos := strings.Split(linea, ",")
        if len(campos) >= 3 {
            id := strings.TrimSpace(campos[0])
            tipo := strings.TrimSpace(campos[1])
            nombre := strings.TrimSpace(campos[2])
            if tipo == "G" && nombre == grupo && id != "0" {
                return true
            }
        }
    }
    return false
}

func ObtenerSiguienteIDGrupo(contenido string) int {
    maxID := 0
    for _, l := range strings.Split(contenido, "\n") {
        l = strings.TrimSpace(l)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 2 {
            continue
        }
        if strings.TrimSpace(p[1]) != "G" {
            continue
        }
        idStr := strings.TrimSpace(p[0])
        if idStr == "0" {
            continue
        }
        if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
            maxID = id
        }
    }
    return maxID + 1
}

func escribirSuperBloque(disk *os.File, sb *structs.SuperBloque) error {
    sbSize := int64(binary.Size(structs.SuperBloque{}))
    sbOffset := int64(sb.S_bm_inode_start) - sbSize

    var sbBuf bytes.Buffer
    if err := binary.Write(&sbBuf, binary.LittleEndian, *sb); err != nil {
        return fmt.Errorf("Serializar superbloque: %v", err)
    }
    if _, err := disk.WriteAt(sbBuf.Bytes(), sbOffset); err != nil {
        return fmt.Errorf("Escribir superbloque: %v", err)
    }
    return nil
}
