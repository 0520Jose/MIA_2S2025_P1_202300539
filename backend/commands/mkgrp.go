package commands

import (
    "fmt"
    "strconv"
    "strings"
    "backend/structs"
    "unsafe"
)

func Mkgrp(args map[string]string) string {
    if currentUser == nil {
        return "Error: No hay una sesión activa."
    }

    if currentUser.Username != "root" {
        return "Error: Solo el usuario root puede ejecutar mkgrp."
    }

    name, ok := args["-name"]
    if !ok || strings.TrimSpace(name) == "" {
        return "Error: Falta el parámetro obligatorio -name."
    }

    // Leer contenido actual
    contenidoActual := LeerArchivoUsersTXT(currentUser.PartitionID)
    if strings.HasPrefix(contenidoActual, "Error:") {
        return contenidoActual
    }

    // Verificar si el grupo ya existe
    if GrupoExiste(contenidoActual, name) {
        return "Error: El grupo '" + name + "' ya existe."
    }

    // Obtener siguiente ID
    nuevoID := ObtenerSiguienteIDGrupo(contenidoActual)
    
    // Crear nueva línea
    nuevaLinea := fmt.Sprintf("%d,G,%s\n", nuevoID, name)
    nuevoContenido := contenidoActual + nuevaLinea

    // Escribir usando el método existente
    err := EscribirUsersTxt(currentUser.PartitionID, nuevoContenido)
    if err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Grupo '%s' creado exitosamente con ID %d", name, nuevoID)
}

// Función para escribir users.txt usando tu sistema existente
func EscribirUsersTxt(partitionID string, contenido string) error {
    for _, pm := range structs.Particiones_Montadas {
        if pm.Id == partitionID {
            path := pm.Path
            partition := pm.Partition

            fs := CargarSistemaEXT2(path, partition)
            if fs == nil {
                return fmt.Errorf("no se pudo cargar el sistema de archivos")
            }
            defer fs.File.Close()

            // Leer el inodo 0 (users.txt)
            inodo, err := fs.LeerInodo(0)
            if err != nil {
                return fmt.Errorf("error al leer inodo de users.txt: %v", err)
            }

            // Verificar que es un archivo
            if inodo.I_type[0] != byte(1) {
                return fmt.Errorf("users.txt no es un archivo válido")
            }

            // Leer el bloque actual
            bloque, err := fs.LeerBloqueArchivo(inodo.I_block[0])
            if err != nil {
                return fmt.Errorf("error al leer bloque de users.txt: %v", err)
            }

            // Limpiar el bloque y copiar nuevo contenido
            for i := range bloque.B_content {
                bloque.B_content[i] = 0
            }

            contenidoBytes := []byte(contenido)
            if len(contenidoBytes) > 64 {
                contenidoBytes = contenidoBytes[:64] // Truncar si es muy largo
            }
            
            copy(bloque.B_content[:], contenidoBytes)

            // Actualizar tamaño en el inodo
            inodo.I_s = int32(len(contenidoBytes))

            // Escribir el bloque actualizado
            posicionBloque := int64(fs.SuperBlock.S_block_start) + int64(inodo.I_block[0])*int64(fs.SuperBlock.S_block_s)
            bloqueBytes := (*[unsafe.Sizeof(bloque)]byte)(unsafe.Pointer(&bloque))[:]
            
            _, err = fs.File.WriteAt(bloqueBytes, posicionBloque)
            if err != nil {
                return fmt.Errorf("error al escribir bloque: %v", err)
            }

            // Escribir el inodo actualizado
            posicionInodo := int64(fs.SuperBlock.S_inode_start)
            inodoBytes := (*[unsafe.Sizeof(inodo)]byte)(unsafe.Pointer(&inodo))[:]
            
            _, err = fs.File.WriteAt(inodoBytes, posicionInodo)
            if err != nil {
                return fmt.Errorf("error al escribir inodo: %v", err)
            }

            return nil
        }
    }
    return fmt.Errorf("partición con ID %s no encontrada", partitionID)
}

// Funciones auxiliares
func GrupoExiste(contenido string, grupo string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, linea := range lineas {
        linea = strings.TrimSpace(linea)
        if linea == "" {
            continue
        }
        campos := strings.Split(linea, ",")
        if len(campos) >= 3 && strings.TrimSpace(campos[1]) == "G" && strings.TrimSpace(campos[2]) == grupo {
            return true
        }
    }
    return false
}

func ObtenerSiguienteIDGrupo(contenido string) int {
    maxID := 0
    lineas := strings.Split(contenido, "\n")
    for _, linea := range lineas {
        linea = strings.TrimSpace(linea)
        if linea == "" {
            continue
        }
        campos := strings.Split(linea, ",")
        if len(campos) >= 1 {
            id, err := strconv.Atoi(strings.TrimSpace(campos[0]))
            if err == nil && id > maxID {
                maxID = id
            }
        }
    }
    return maxID + 1
}