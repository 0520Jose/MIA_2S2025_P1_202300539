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

func Mkdisk(params map[string]string) string {
    allowed := map[string]struct{}{
        "-size": {}, "-unit": {}, "-fit": {}, "-path": {},
    }
    normalized := make(map[string]string, len(params))
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if _, ok := allowed[lk]; !ok {
            return fmt.Sprintf("Error: parámetro no reconocido: %s", k)
        }
        normalized[lk] = v
    }

    sizeStr, existe := normalized["-size"]
    if !existe {
        return fmt.Sprintf("Error: parámetro -size es obligatorio")
    }

    size, err := strconv.Atoi(sizeStr)
    if err != nil || size <= 0 {
        return fmt.Sprintf("Error: -size debe ser un entero positivo")
    }

    path, existe := normalized["-path"]
    if !existe {
        return fmt.Sprintf("Error: parámetro -path es obligatorio")
    }

    path, err = LimpiarRuta(path)
    if err != nil {
        return fmt.Sprintf("Error en la ruta: %v", err)
    }

    if !strings.HasSuffix(path, ".mia") {
        return fmt.Sprintf("Error: el archivo debe tener extensión .mia")
    }

    unit := "M"
    if u, existe := normalized["-unit"]; existe {
        unit = strings.ToUpper(u)
    }

    tamanioBytes := size
    switch unit {
    case "K":
        tamanioBytes *= 1024
    case "M":
        tamanioBytes *= 1024 * 1024
    default:
        return fmt.Sprintf("Error: unit inválida (use K o M)")
    }

    fit := "FF"
    if f, existe := normalized["-fit"]; existe {
        f = strings.ToUpper(f)
        if f != "FF" && f != "BF" && f != "WF" {
            return fmt.Sprintf("Error: fit inválido (use FF, BF o WF)")
        }
        fit = f
    }

    var fitChar byte
    switch fit {
    case "FF":
        fitChar = 'F'
    case "BF":
        fitChar = 'B'
    case "WF":
        fitChar = 'W'
    }

    if _, err := os.Stat(path); err == nil {
        return fmt.Sprintf("Error: ya existe un disco en %s", path)
    }

    carpetaPadre := filepath.Dir(path)
    if err := os.MkdirAll(carpetaPadre, 0755); err != nil {
        return fmt.Sprintf("Error creando directorios: %v", err)
    }

    archivo, err := os.Create(path)
    if err != nil {
        return fmt.Sprintf("Error creando disco: %v", err)
    }
    defer archivo.Close()

    buffer := make([]byte, 1024)
    bytesEscritos := 0
    for bytesEscritos < tamanioBytes {
        if tamanioBytes-bytesEscritos < 1024 {
            buffer = make([]byte, tamanioBytes-bytesEscritos)
        }
        if _, err := archivo.Write(buffer); err != nil {
            return fmt.Sprintf("Error escribiendo ceros: %v", err)
        }
        bytesEscritos += len(buffer)
    }

    mbr := structs.MBR{
        Mbr_tamano:        int32(tamanioBytes),
        Mbr_dsk_signature: rand.Int31(),
        Dsk_fit:           fitChar,
    }

    fechaActual := time.Now().Format("2006-01-02T15:04")
    copy(mbr.Mbr_fecha_creacion[:], fechaActual)

    archivo.Seek(0, 0)
    if err := binary.Write(archivo, binary.LittleEndian, &mbr); err != nil {
        return fmt.Sprintf("Error escribiendo MBR: %v", err)
    }

    return fmt.Sprintf("Disco creado exitosamente: %s (%d bytes) [fit=%s]\n", path, tamanioBytes, fit)
}
