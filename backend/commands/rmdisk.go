package commands

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

func Rmdisk(params map[string]string) string {
    path, existPath := params["-path"]
    if !existPath {
        return fmt.Sprintf("Error: parámetro -path es obligatorio")
    }

    var err error
    path, err = limpiarRuta(path)
    if err != nil {
        return fmt.Sprintf("Error en la ruta: %v", err)
    }

    if !strings.HasSuffix(path, ".mia") {
        return fmt.Sprintf("Error: el archivo debe tener extensión .mia")
    }

    if _, err := os.Stat(path); os.IsNotExist(err) {
        return fmt.Sprintf("Error: el archivo no existe en la ruta especificada")
    }

    fmt.Printf("¿Desea eliminar el disco %s? (S/N): ", path)
    reader := bufio.NewReader(os.Stdin)
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(strings.ToUpper(input))

    if input != "S" {
        return "Operación cancelada por el usuario"
    }

    if err := os.Remove(path); err != nil {
        return fmt.Sprintf("Error eliminando archivo: %v", err)
    }

    return fmt.Sprintf("Disco eliminado exitosamente: %s\n", path)
}
