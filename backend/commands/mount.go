package commands

import (
    "backend/structs"
    "fmt"
    "os"
    "strings"
)

var (
    particionesMontadas []PartitionMount
    contadorDiscos      = make(map[string]int)
    letrasDiscos        = make(map[string]string)
    letraActual         = 'A'
)

type PartitionMount struct {
    Id        string
    Path      string
    Partition structs.Partition
}

func Mount(params map[string]string) {
    path := params["-path"]
    name := params["-name"]

    if path == "" || name == "" {
        fmt.Println("Error: parámetros -path y -name son obligatorios")
        return
    }

    archivo, err := os.OpenFile(path, os.O_RDONLY, 0644)
    if err != nil {
        fmt.Println("Error abriendo el disco:", err)
        return
    }
    defer archivo.Close()

    mbr, err := structs.LeerMBR(archivo)
    if err != nil {
        fmt.Println("Error leyendo MBR:", err)
        return
    }

    var particion structs.Partition
    encontrada := false

    for _, p := range mbr.Mbr_partitions {
        partitionName := strings.TrimRight(string(p.Part_name[:]), "\x00")
        
        if partitionName == name && p.Part_type == 'P' && p.Part_status == 1 {
            particion = p
            encontrada = true
            break
        }
    }

    if !encontrada {
        fmt.Println("Error: no se encontró partición primaria activa con ese nombre")
        return
    }

    for _, pm := range particionesMontadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        if pm.Path == path && pmName == name {
            fmt.Println("Error: partición ya montada")
            return
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

    particionesMontadas = append(particionesMontadas, PartitionMount{
        Id:        id,
        Path:      path,
        Partition: particion,
    })

    fmt.Printf("Partición montada con ID: %s\n", id)
    fmt.Println("Particiones actualmente montadas:")
    for _, pm := range particionesMontadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        fmt.Printf(" - %s: %s (%s)\n", pm.Id, pmName, pm.Path)
    }
}