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
    path := params["-path"]
    name := params["-name"]

    if path == "" || name == "" {
        return fmt.Sprintf("Error: parámetros -path y -name son obligatorios\n")
    }

    archivo, err := os.OpenFile(path, os.O_RDONLY, 0644)
    if err != nil {
        return fmt.Sprintf("Error abriendo el disco: %v\n", err)
    }
    defer archivo.Close()

    mbr, err := structs.LeerMBR(archivo)
    if err != nil {
        return fmt.Sprintf("Error leyendo MBR: %v\n", err)
    }

    var particion structs.Partition
    encontrada := false
    indiceParticion := -1

    for i, p := range mbr.Mbr_partitions {
        partitionName := strings.TrimRight(string(p.Part_name[:]), "\x00")
        if partitionName == name && p.Part_type == 'P' {
            particion = p
            indiceParticion = i
            encontrada = true
            break
        }
    }

    if !encontrada {
        return fmt.Sprintf("Error: no se encontró partición primaria con ese nombre\n")
    }

    for _, pm := range particionesMontadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        if pm.Path == path && pmName == name {
            return fmt.Sprintf("Error: partición ya montada\n")
        }
    }

    carnet := "39"

    letra, exists := letrasDiscos[path]
    if !exists {
        letra = string(letraActual)
        letrasDiscos[path] = letra
        letraActual++
        contadorDiscos[path] = 0
    }
    contadorDiscos[path]++

    id := fmt.Sprintf("%s%d%s", carnet, contadorDiscos[path], letra)

    particion.Part_status = 1
    particion.Part_correlative = int32(contadorDiscos[path])
    copy(particion.Part_id[:], id)

    mbr.Mbr_partitions[indiceParticion] = particion

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
