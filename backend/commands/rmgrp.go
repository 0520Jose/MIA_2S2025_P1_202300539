package commands

import (
    "backend/structs"
    "bytes"
    "encoding/binary"
    "fmt"
    "strings"
)

func Rmgrp(args map[string]string) string {
    if currentUser == nil {
        return "Error: No hay una sesión activa."
    }
    if currentUser.Username != "root" {
        return "Error: Solo el usuario root puede ejecutar rmgrp."
    }

    name, ok := args["-name"]
    if !ok || strings.TrimSpace(name) == "" {
        return "Error: Falta el parámetro obligatorio -name."
    }
    grupo := strings.TrimSpace(name)

    // Cargar FS
    disk, sb, err := CargarSistemaEXT2(currentUser.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    // Leer users.txt
    contenidoActual, err := LeerArchivoUsersTXT(disk, sb)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    // Marcar el grupo como eliminado (id=0) si existe con id>0
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

    // Escribir users.txt actualizado
    if err := escribirUsersTxtRaw(currentUser.PartitionID, nuevoContenido); err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Grupo '%s' eliminado exitosamente", grupo)
}

// escribirUsersTxtRaw escribe contenido completo en el bloque de users.txt (inodo 1).
func escribirUsersTxtRaw(partitionID string, contenido string) error {
    disk, sb, err := CargarSistemaEXT2(partitionID)
    if err != nil {
        return err
    }
    defer disk.Close()

    // Inodo 1 = users.txt
    inodoPos := int64(sb.S_inode_start + int32(binary.Size(structs.Inodo{})))
    if _, err := disk.Seek(inodoPos, 0); err != nil {
        return fmt.Errorf("error al ubicar inodo de users.txt: %v", err)
    }
    var inodo structs.Inodo
    if err := binary.Read(disk, binary.LittleEndian, &inodo); err != nil {
        return fmt.Errorf("error al leer inodo de users.txt: %v", err)
    }
    if inodo.I_block[0] == -1 {
        return fmt.Errorf("users.txt no tiene bloques asignados")
    }

    // Preparar bloque
    var bloque structs.BArchivo
    data := []byte(contenido)
    if len(data) > len(bloque.B_content) {
        data = data[:len(bloque.B_content)]
    }
    for i := range bloque.B_content {
        bloque.B_content[i] = 0
    }
    copy(bloque.B_content[:], data)

    // Escribir bloque
    bloquePos := int64(sb.S_block_start) + int64(inodo.I_block[0])*int64(sb.S_block_s)
    var bbuf bytes.Buffer
    if err := binary.Write(&bbuf, binary.LittleEndian, &bloque); err != nil {
        return fmt.Errorf("error serializando bloque: %v", err)
    }
    if _, err := disk.WriteAt(bbuf.Bytes(), bloquePos); err != nil {
        return fmt.Errorf("error al escribir bloque de users.txt: %v", err)
    }

    // Actualizar tamaño y reescribir inodo
    inodo.I_s = int32(len(data))
    var ibuf bytes.Buffer
    if err := binary.Write(&ibuf, binary.LittleEndian, &inodo); err != nil {
        return fmt.Errorf("error serializando inodo: %v", err)
    }
    if _, err := disk.WriteAt(ibuf.Bytes(), inodoPos); err != nil {
        return fmt.Errorf("error al escribir inodo actualizado: %v", err)
    }

    return nil
}