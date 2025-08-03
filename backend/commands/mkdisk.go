package commands

import (
    "encoding/binary"
    "fmt"
    "math/rand"
    "backend/structs"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

func Mkdisk(params map[string]string) {
    sizeStr, existe := params["-size"]
    if !existe {
        fmt.Println("Error: parámetro -size es obligatorio")
        return
    }

    size, err := strconv.Atoi(sizeStr)
    if err != nil || size <= 0 {
        fmt.Println("Error: -size debe ser un entero positivo")
        return
    }

    path, existe := params["-path"]
    if !existe {
        fmt.Println("Error: parámetro -path es obligatorio")
        return
    }

    path, err = limpiarRuta(path)
    if err != nil {
        fmt.Println("Error en la ruta:", err)
        return
    }

    if !strings.HasSuffix(path, ".mia") {
        fmt.Println("Error: el archivo debe tener extensión .mia")
        return
    }

    unit := "M"
    if u, existe := params["-unit"]; existe {
        unit = strings.ToUpper(u)
    }

    tamanioBytes := size
    switch unit {
    case "K":
        tamanioBytes *= 1024
    case "M":
        tamanioBytes *= 1024 * 1024
    default:
        fmt.Println("Error: unit inválida (use K o M)")
        return
    }

    carpetaPadre := filepath.Dir(path)
    if err := os.MkdirAll(carpetaPadre, 0755); err != nil {
        fmt.Println("Error creando directorios:", err)
        return
    }

    archivo, err := os.Create(path)
    if err != nil {
        fmt.Println("Error creando disco:", err)
        return
    }
    defer archivo.Close()

    buffer := make([]byte, 1024)
    bytesEscritos := 0
    for bytesEscritos < tamanioBytes {
        if tamanioBytes-bytesEscritos < 1024 {
            buffer = make([]byte, tamanioBytes-bytesEscritos)
        }
        if _, err := archivo.Write(buffer); err != nil {
            fmt.Println("Error escribiendo ceros:", err)
            return
        }
        bytesEscritos += len(buffer)
    }

    mbr := structs.MBR{
        Mbr_tamano:        int32(tamanioBytes),
        Mbr_dsk_signature: rand.Int31(),
        Dsk_fit:           'F',
    }

    fechaActual := time.Now().Format("2006-01-02T15:04")
    copy(mbr.Mbr_fecha_creacion[:], fechaActual)

    archivo.Seek(0, 0)
    if err := binary.Write(archivo, binary.LittleEndian, &mbr); err != nil {
        fmt.Println("Error escribiendo MBR:", err)
        return
    }

    fmt.Printf("Disco creado exitosamente: %s (%d bytes)\n", path, tamanioBytes)
}