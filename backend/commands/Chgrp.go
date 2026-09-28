package commands

import (
    "fmt"
    "strings"
)

func Chgrp(args map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    if usuarioActual.Username != "root" {
        return "Error: Solo el usuario root puede ejecutar chgrp."
    }

    user, okU := args["-user"]
    grp, okG := args["-grp"]
    user = strings.TrimSpace(user)
    grp = strings.TrimSpace(grp)

    if !okU || user == "" || !okG || grp == "" {
        return "Error: parámetros obligatorios -user y -grp."
    }
    if len(user) > 10 {
        return "Error: -user excede 10 caracteres."
    }
    if len(grp) > 10 {
        return "Error: -grp excede 10 caracteres."
    }

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Errorf("Error: %v", err).Error()
    }
    defer disk.Close()

    contenido, err := LeerArchivoUsersTXT(disk, sb)
    if err != nil {
        return fmt.Errorf("Error: %v", err).Error()
    }

    if !GrupoExisteActivo(contenido, grp) {
        return fmt.Sprintf("Error: El grupo '%s' no existe o está eliminado.", grp)
    }
    if !UsuarioExisteActivo(contenido, user) {
        return fmt.Sprintf("Error: El usuario '%s' no existe.", user)
    }

    lineas := strings.Split(contenido, "\n")
    actualizado := false
    for i, l := range lineas {
        t := strings.TrimSpace(l)
        if t == "" || strings.HasPrefix(t, "#") {
            continue
        }
        p := strings.Split(t, ",")
        if len(p) < 5 {
            continue
        }
        id := strings.TrimSpace(p[0])
        tipo := strings.TrimSpace(p[1])
        usr := strings.TrimSpace(p[3])
        pass := strings.TrimSpace(p[4])

        if tipo == "U" && usr == user {
            if id == "0" {
                return fmt.Sprintf("Error: El usuario '%s' está eliminado.", user)
            }
            lineas[i] = strings.Join([]string{ id, "U", grp, user, pass }, ",")
            actualizado = true
            break
        }
    }

    if !actualizado {
        return fmt.Sprintf("Error: No se pudo actualizar el usuario '%s'.", user)
    }

    nuevoContenido := asegurarNuevaLineaFinal(strings.Join(lineas, "\n"))
    if err := EscribirUsersTxt(usuarioActual.PartitionID, nuevoContenido); err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Usuario '%s' movido al grupo '%s' exitosamente", user, grp)
}