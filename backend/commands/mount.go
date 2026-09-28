package commands

import (
    "backend/structs"
    "fmt"
    "os"
    "strings"
)

var (
    particionesMontadas []structs.PartitionMount
    contadorDiscos      = make(map[string]int)
    letrasDiscos        = make(map[string]string)
    letraActual         = 'A'
)

func Mount(params map[string]string) string {
    allowed := map[string]struct{}{
        "-path": {}, "-name": {},
    }
    normalized := make(map[string]string, len(params))
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if _, ok := allowed[lk]; !ok {
            return fmt.Sprintf("Error: parámetro no reconocido: %s\n", k)
        }
        normalized[lk] = v
    }

    path, ok := normalized["-path"]
    if !ok || strings.TrimSpace(path) == "" {
        return "Error: parámetros -path y -name son obligatorios\n"
    }
    name, ok := normalized["-name"]
    if !ok || strings.TrimSpace(name) == "" {
        return "Error: parámetros -path y -name son obligatorios\n"
    }

    var err error
    path, err = LimpiarRuta(path)
    if err != nil {
        return fmt.Sprintf("Error en la ruta: %v\n", err)
    }

    archivo, err := os.OpenFile(path, os.O_RDONLY, 0)
    if err != nil {
        return fmt.Sprintf("Error abriendo el disco: %v\n", err)
    }
    defer archivo.Close()

    mbr, err := structs.LeerMBR(path)
    if err != nil {
        return fmt.Sprintf("Error leyendo MBR: %v\n", err)
    }

    var particion structs.Partition
    encontrada := false
    for _, p := range mbr.Mbr_partitions {
        partitionName := strings.TrimRight(string(p.Part_name[:]), "\x00")
        if partitionName == name && p.Part_type == 'P' {
            particion = p
            encontrada = true
            break
        }
    }
    if !encontrada {
        return "Error: no se encontró una partición primaria con ese nombre\n"
    }

    for _, pm := range particionesMontadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        if pm.Path == path && pmName == name {
            return "Error: partición ya montada\n"
        }
    }
    carnetUltimos2 := "39"
    letra, exists := letrasDiscos[path]
    if !exists {
        letra = string(letraActual)
        letrasDiscos[path] = letra
        letraActual++
        contadorDiscos[path] = 0
    }
    contadorDiscos[path]++
    correlativo := contadorDiscos[path]

    id := fmt.Sprintf("%s%d%s", carnetUltimos2, correlativo, letra)

    particion.Part_status = 1
    particion.Part_correlative = int32(correlativo)
    copy(particion.Part_id[:], id)

    particionesMontadas = append(particionesMontadas, structs.PartitionMount{
        Id:        id,
        Path:      path,
        Partition: particion,
    })
    structs.Particiones_Montadas = particionesMontadas

    var output strings.Builder
    for _, pm := range particionesMontadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        output.WriteString(fmt.Sprintf(" - %s: %s (%s)\n", pm.Id, pmName, pm.Path))
    }
    return output.String()
}