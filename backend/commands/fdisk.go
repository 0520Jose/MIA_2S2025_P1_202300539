package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "os"
    "strconv"
    "strings"
)

func Fdisk(params map[string]string) {
    path, existPath := params["-path"]
    name, existName := params["-name"]
    sizeStr, existSize := params["-size"]

    if !existPath || !existName || !existSize {
        fmt.Println("Error: parámetros obligatorios faltantes")
        return
    }

    size, err := strconv.ParseInt(sizeStr, 10, 32)
    if err != nil || size <= 0 {
        fmt.Println("Error: -size debe ser un entero positivo")
        return
    }

    unit := "K"
    if u, existe := params["-unit"]; existe {
        unit = strings.ToUpper(u)
    }

    tamanioBytes := size
    switch unit {
    case "B":
    case "K":
        tamanioBytes *= 1024
    case "M":
        tamanioBytes *= 1024 * 1024
    default:
        fmt.Println("Error: unit inválida (B, K, M)")
        return
    }

    partType := byte('P')
    if t, existe := params["-type"]; existe {
        switch strings.ToUpper(t) {
        case "P":
            partType = 'P'
        case "E":
            partType = 'E'
        case "L":
            partType = 'L'
        default:
            fmt.Println("Error: type inválido (P, E, L)")
            return
        }
    }

    fit := byte('W')
    if a, existe := params["-fit"]; existe {
        switch strings.ToUpper(a) {
        case "BF":
            fit = 'B'
        case "FF":
            fit = 'F'
        case "WF":
            fit = 'W'
        default:
            fmt.Println("Error: fit inválido (BF, FF, WF)")
            return
        }
    }

    file, err := os.OpenFile(path, os.O_RDWR, 0644)
    if err != nil {
        fmt.Println("Error abriendo disco:", err)
        return
    }
    defer file.Close()

    mbr, err := structs.LeerMBR(file)
    if err != nil {
        fmt.Println("Error leyendo MBR:", err)
        return
    }

    inicio := int32(binary.Size(mbr))
    var mejorInicio int32 = -1
    var mejorTamanio int32 = 1<<31 - 1
    var peorInicio int32 = -1
    var peorTamanio int32 = -1
    var espacioFinal int32

    for i := 0; i < 4; i++ {
        part := mbr.Mbr_partitions[i]
        if part.Part_status == 1 {
            espacioLibre := part.Part_start - inicio

            if espacioLibre >= int32(tamanioBytes) {
                switch fit {
                case 'F':
                    goto ENCONTRADO
                case 'B':
                    if espacioLibre < mejorTamanio {
                        mejorTamanio = espacioLibre
                        mejorInicio = inicio
                    }
                case 'W':
                    if espacioLibre > peorTamanio {
                        peorTamanio = espacioLibre
                        peorInicio = inicio
                    }
                }
            }
            inicio = part.Part_start + part.Part_s
        }
    }

    espacioFinal = mbr.Mbr_tamano - inicio
    if espacioFinal >= int32(tamanioBytes) {
        switch fit {
        case 'F':
        case 'B':
            if espacioFinal < mejorTamanio {
                mejorInicio = inicio
            }
        case 'W':
            if espacioFinal > peorTamanio {
                peorInicio = inicio
            }
        }
    }

    switch fit {
    case 'F':
    case 'B':
        if mejorInicio != -1 {
            inicio = mejorInicio
        } else {
            fmt.Println("Error: no hay espacio (Best Fit)")
            return
        }
    case 'W':
        if peorInicio != -1 {
            inicio = peorInicio
        } else {
            fmt.Println("Error: no hay espacio (Worst Fit)")
            return
        }
    }

ENCONTRADO:
    nuevaParticion := structs.Partition{
        Part_status: 1,
        Part_type: partType,
        Part_fit: fit,
        Part_start: inicio,
        Part_s: int32(tamanioBytes),
        Part_correlative: -1,
    }

    copy(nuevaParticion.Part_name[:], []byte(name))
    if len(name) > 16 {
        copy(nuevaParticion.Part_name[:], []byte(name[:16]))
    }

    agregado := false
    for i := 0; i < 4; i++ {
        if mbr.Mbr_partitions[i].Part_status == 0 {
            mbr.Mbr_partitions[i] = nuevaParticion
            agregado = true
            break
        }
    }

    if !agregado {
        fmt.Println("Error: máximo 4 particiones alcanzado")
        return
    }

    file.Seek(0, 0)
    if err := binary.Write(file, binary.LittleEndian, &mbr); err != nil {
        fmt.Println("Error escribiendo MBR:", err)
        return
    }

    fmt.Printf("Partición creada: %s @ %d-%d (%d bytes)\n",
        name,
        inicio,
        inicio+ int32(tamanioBytes),
        tamanioBytes,
    )
}