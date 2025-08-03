package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "os"
    "strconv"
    "strings"
)

var particionesMontadas []PartitionMount

type PartitionMount struct {
    Id string
    Path string
    partition structs.Partition
}

func Mount(params map[string]string) {
    path := params["-path"]
    name := params["-name"]

    archivo, err := os.OpenFile(path, os.O_RDONLY, 0644)
    if err != nil {
        fmt.Println("Error abriendo el disco:", err)
        return
    }
    defer archivo.Close()

    var mbr structs.MBR
    if err := binary.Read(archivo, binary.BigEndian, &mbr); err != nil {
        fmt.Println("Error leyendo el MBR del disco:", err)
        return
    }

    var particionEncontrada structs.Partition
    var encontrada bool
    for _, p := range mbr.Mbr_partitions {
        namePartition := strings.TrimRight(string(p.Part_name[:]), "\x00")
        if p.Part_status == 1 && namePartition == name {
            particionEncontrada = p
            encontrada = true
            break
        }
    }

    if !encontrada {
        fmt.Println("Error: no se encontró la partición con el name especificado.")
        return
    }

    id := "01" + strconv.Itoa(len(particionesMontadas)+1) + "A"

    particionesMontadas = append(particionesMontadas, PartitionMount{
        Id: id,
        Path: path,
        partition: particionEncontrada,
    })

    fmt.Println("Partición montada con ID:", id)
}