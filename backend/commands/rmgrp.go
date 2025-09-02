package commands

import (
    "fmt"
    "strings"
)

func Rmgrp(args map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    if usuarioActual.Username != "root" {
        return "Error: Solo el usuario root puede ejecutar rmgrp."
    }

    name, ok := args["-name"]
    if !ok || strings.TrimSpace(name) == "" {
        return "Error: Falta el parámetro obligatorio -name."
    }
    grupo := strings.TrimSpace(name)

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    contenidoActual, err := LeerArchivoUsersTXT(disk, sb)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    lineas := strings.Split(contenidoActual, "\n")
    encontrado := false
    for i, linea := range lineas {
        l := strings.TrimSpace(linea)
        if l == "" {
            continue
        }
        campos := strings.Split(l, ",")
        if len(campos) < 3 {
            continue
        }
        id := strings.TrimSpace(campos[0])
        tipo := strings.TrimSpace(campos[1])
        nombre := strings.TrimSpace(campos[2])
        if tipo == "G" && nombre == grupo {
            if id == "0" {
                return "Error: El grupo ya está eliminado."
            }
            campos[0] = "0"
            lineas[i] = strings.Join(campos, ",")
            encontrado = true
            break
        }
    }
    if !encontrado {
        return "Error: El grupo no existe."
    }

    nuevoContenido := strings.Join(lineas, "\n")
    if !strings.HasSuffix(nuevoContenido, "\n") {
        nuevoContenido += "\n"
    }

    if err := EscribirUsersTxt(usuarioActual.PartitionID, nuevoContenido); err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Grupo '%s' eliminado exitosamente", grupo)
}