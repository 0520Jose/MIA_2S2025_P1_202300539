package commands

import (
    "fmt"
    "strconv"
    "strings"
)

func Mkusr(args map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    if usuarioActual.Username != "root" {
        return "Error: Solo el usuario root puede ejecutar mkusr."
    }

    user, okU := args["-user"]
    pass, okP := args["-pass"]
    grp, okG := args["-grp"]

    user = strings.TrimSpace(user)
    pass = strings.TrimSpace(pass)
    grp = strings.TrimSpace(grp)

    if !okU || user == "" || !okP || pass == "" || !okG || grp == "" {
        return "Error: parámetros obligatorios -user, -pass y -grp."
    }
    if len(user) > 10 {
        return "Error: -user excede 10 caracteres."
    }
    if len(pass) > 10 {
        return "Error: -pass excede 10 caracteres."
    }
    if len(grp) > 10 {
        return "Error: -grp excede 10 caracteres."
    }

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    contenido, err := LeerArchivoUsersTXT(disk, sb)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    if !GrupoExisteActivo(contenido, grp) {
        return fmt.Sprintf("Error: El grupo '%s' no existe en esta partición.", grp)
    }
    if UsuarioExisteActivo(contenido, user) {
        return fmt.Sprintf("Error: El usuario '%s' ya existe.", user)
    }

    nuevoID := SiguienteID(contenido)
    nuevaLinea := fmt.Sprintf("%d,U,%s,%s,%s\n", nuevoID, grp, user, pass)
    nuevoContenido := asegurarNuevaLineaFinal(contenido) + nuevaLinea

    if err := EscribirUsersTxt(usuarioActual.PartitionID, nuevoContenido); err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Usuario '%s' creado exitosamente en el grupo '%s' con ID %d", user, grp, nuevoID)
}

func GrupoExisteActivo(contenido, grupo string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, l := range lineas {
        l = strings.TrimSpace(l)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 3 {
            continue
        }
        id := strings.TrimSpace(p[0])
        tipo := strings.TrimSpace(p[1])
        nombre := strings.TrimSpace(p[2])
        if tipo == "G" && nombre == grupo && id != "0" {
            return true
        }
    }
    return false
}

func UsuarioExisteActivo(contenido, usuario string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, l := range lineas {
        l = strings.TrimSpace(l)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 5 {
            continue
        }
        id := strings.TrimSpace(p[0])
        tipo := strings.TrimSpace(p[1])
        nombreUsuario := strings.TrimSpace(p[3])
        if tipo == "U" && nombreUsuario == usuario && id != "0" {
            return true
        }
    }
    return false
}

func SiguienteID(contenido string) int {
    maxID := 0
    for _, l := range strings.Split(contenido, "\n") {
        l = strings.TrimSpace(l)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 5 {
            continue
        }
        idStr := strings.TrimSpace(p[0])
        tipo := strings.TrimSpace(p[1])
        if tipo != "U" || idStr == "0" {
            continue
        }
        if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
            maxID = id
        }
    }
    return maxID + 1
}

func asegurarNuevaLineaFinal(s string) string {
    if s == "" {
        return ""
    }
    if strings.HasSuffix(s, "\n") {
        return s
    }
    return s + "\n"
}