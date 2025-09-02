package commands

import (
    "fmt"
    "strings"
)

func Rmusr(args map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    if usuarioActual.Username != "root" {
        return "Error: Solo el usuario root puede ejecutar rmusr."
    }

    u, ok := args["-user"]
    if !ok || strings.TrimSpace(u) == "" {
        return "Error: Falta el parámetro obligatorio -user."
    }
    target := strings.TrimSpace(u)

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    contenido, err := LeerArchivoUsersTXT(disk, sb)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    lineas := strings.Split(contenido, "\n")
    found := false
    for i, l := range lineas {
        l = strings.TrimSpace(l)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) >= 5 {
            id := strings.TrimSpace(p[0])
            tipo := strings.TrimSpace(p[1])
            user := strings.TrimSpace(p[3])
            if tipo == "U" && user == target && id != "0" {
                p[0] = "0"
                lineas[i] = strings.Join([]string{
                    strings.TrimSpace(p[0]),
                    strings.TrimSpace(p[1]),
                    strings.TrimSpace(p[2]),
                    strings.TrimSpace(p[3]),
                    strings.TrimSpace(p[4]),
                }, ",")
                found = true
                break
            }
        }
    }

    if !found {
        return fmt.Sprintf("Error: El usuario '%s' no existe.", target)
    }

    nuevoContenido := strings.Join(lineas, "\n")
    if !strings.HasSuffix(nuevoContenido, "\n") {
        nuevoContenido += "\n"
    }

    if err := EscribirUsersTxt(usuarioActual.PartitionID, nuevoContenido); err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Usuario '%s' eliminado exitosamente", target)
}