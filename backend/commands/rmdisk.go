package commands

import (
    "fmt"
    "os"
    "strings"
)

func Rmdisk(params map[string]string) {
    path, existPath := params["-path"]
    if !existPath {
        fmt.Println("Error: parámetro -path es obligatorio")
        return
    }

    var err error
	path, err = limpiarRuta(path)
	if err != nil {
		fmt.Println("Error en la ruta:", err)
		return
	}

    if !strings.HasSuffix(path, ".mia") {
        fmt.Println("Error: el archivo debe tener extensión .mia")
        return
    }

    if _, err := os.Stat(path); os.IsNotExist(err) {
        fmt.Println("Error: el archivo no existe en la ruta especificada")
        return
    }

    if err := os.Remove(path); err != nil {
        fmt.Println("Error eliminando archivo:", err)
        return
    }

    fmt.Printf("Disco eliminado exitosamente: %s\n", path)
}