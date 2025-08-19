package commands

import (
    "backend/structs"
    "fmt"
    "strings"
)

func Cat(args map[string]string) string {
    if currentUser == nil {
        return "Error: No hay una sesión activa."
    }

    filePath, existe := args["-file"]
    if !existe {
        return "Error: parámetro -file es obligatorio. Uso: cat -file=/ruta/archivo"
    }

    if filePath == "/users.txt" || filePath == "users.txt" {
        contenido := LeerArchivoUsersTXT(currentUser.PartitionID)
        if strings.HasPrefix(contenido, "Error:") {
            return contenido
        }

        usersTXT := structs.UsersTXT{}
        usersTXT.FromString(contenido)

        if currentUser.Username != "root" {
            return "Error: No tienes permisos para ver users.txt"
        }

        return usersTXT.ToString()
    }

    return fmt.Sprintf("Error: archivo '%s' no encontrado o no soportado", filePath)
}

func CatUsersFile(partitionID string) string {
    if currentUser == nil {
        return "Error: No hay una sesión activa."
    }

    if currentUser.PartitionID != partitionID {
        return "Error: No tienes permisos para acceder a esta partición."
    }

    contenido := LeerArchivoUsersTXT(partitionID)
    if strings.HasPrefix(contenido, "Error:") {
        return contenido
    }

    usersTXT := structs.UsersTXT{}
    usersTXT.FromString(contenido)

    if currentUser.Username != "root" {
        return "Error: No tienes permisos para ver users.txt"
    }

    return fmt.Sprintf("=== Contenido de users.txt ===\n%s", usersTXT.ToString())
}

func CatMultiple(args map[string]string) string {
    if currentUser == nil {
        return "Error: No hay una sesión activa."
    }

    var contenidoTotal strings.Builder
    hayArchivos := false

    for i := 1; i <= 10; i++ {
        key := fmt.Sprintf("-file%d", i)
        path, existe := args[key]
        if !existe {
            continue
        }

        hayArchivos = true

        if path == "/users.txt" || path == "users.txt" {
            contenido := LeerArchivoUsersTXT(currentUser.PartitionID)
            if strings.HasPrefix(contenido, "Error:") {
                contenidoTotal.WriteString(fmt.Sprintf("Error al leer %s: %s\n", path, contenido))
            } else {
                usersTXT := structs.UsersTXT{}
                usersTXT.FromString(contenido)

                if currentUser.Username != "root" {
                    contenidoTotal.WriteString(fmt.Sprintf("Error: No tienes permisos para ver %s\n", path))
                } else {
                    contenidoTotal.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", path, usersTXT.ToString()))
                }
            }
        } else {
            contenidoTotal.WriteString(fmt.Sprintf("Error: archivo '%s' no encontrado\n", path))
        }
    }

    if !hayArchivos {
        return "Error: No se proporcionaron archivos para el comando cat. Uso: cat -file=/ruta/archivo"
    }

    return contenidoTotal.String()
}
