package commands

import (
    "fmt"
    "os"
    "strings"
)

func Rmdisk(params map[string]string) string {
    // Validar parámetros permitidos y normalizar llaves a minúsculas
    allowed := map[string]struct{}{
        "-path": {},
    }
    normalized := make(map[string]string, len(params))
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if _, ok := allowed[lk]; !ok {
            return fmt.Sprintf("Error: parámetro no reconocido: %s", k)
        }
        normalized[lk] = v
    }

    // -path obligatorio
    path, ok := normalized["-path"]
    if !ok {
        return "Error: parámetro -path es obligatorio"
    }

    // Soporte de comillas dobles y validaciones de ruta
    var err error
    path, err = limpiarRuta(path)
    if err != nil {
        return fmt.Sprintf("Error en la ruta: %v", err)
    }

    // Solo archivos .mia
    if !strings.HasSuffix(path, ".mia") {
        return "Error: el archivo debe tener extensión .mia"
    }

    // Verificar existencia
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return "Error: el archivo no existe en la ruta especificada"
    } else if err != nil {
        return fmt.Sprintf("Error verificando archivo: %v", err)
    }

    // Eliminar archivo
    if err := os.Remove(path); err != nil {
        return fmt.Sprintf("Error eliminando archivo: %v", err)
    }

    return fmt.Sprintf("Disco eliminado exitosamente: %s\n", path)
}