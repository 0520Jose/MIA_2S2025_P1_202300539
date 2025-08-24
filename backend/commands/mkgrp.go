package commands

import (
    "backend/structs"
    "bytes"
    "encoding/binary"
    "fmt"
    "strconv"
    "strings"
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
    disk, sb, err := CargarSistemaEXT2(currentUser.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    contenidoActual, err := LeerArchivoUsersTXT(disk, sb)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    if GrupoExiste(contenidoActual, name) {
        return "Error: El grupo '" + name + "' ya existe."
    }

    nuevoID := ObtenerSiguienteIDGrupo(contenidoActual)
    nuevaLinea := fmt.Sprintf("%d,G,%s\n", nuevoID, name)
    nuevoContenido := contenidoActual + nuevaLinea

    if err := EscribirUsersTxt(currentUser.PartitionID, nuevoContenido); err != nil {
        return "Error al escribir users.txt: " + err.Error()
    }

    return fmt.Sprintf("Grupo '%s' creado exitosamente con ID %d", name, nuevoID)
}

func EscribirUsersTxt(partitionID string, contenido string) error {
    disk, sb, err := CargarSistemaEXT2(partitionID)
    if err != nil {
        return err
    }
    defer disk.Close()

    inodeOffset := int64(sb.S_inode_start) + 1*int64(sb.S_inode_s)
    if _, err := disk.Seek(inodeOffset, 0); err != nil {
        return fmt.Errorf("posicionar inodo users.txt: %v", err)
    }
    var ino structs.Inodo
    if err := binary.Read(disk, binary.LittleEndian, &ino); err != nil {
        return fmt.Errorf("leer inodo users.txt: %v", err)
    }
    if ino.I_block[0] < 0 {
        return fmt.Errorf("users.txt sin bloque asignado")
    }

    var b structs.BArchivo
    data := []byte(contenido)
    if len(data) > len(b.B_content) {
        data = data[:len(b.B_content)]
    }
    for i := range b.B_content {
        b.B_content[i] = 0
    }
    copy(b.B_content[:], data)

    blockOffset := int64(sb.S_block_start) + int64(ino.I_block[0])*int64(sb.S_block_s)
    var buf bytes.Buffer
    if err := binary.Write(&buf, binary.LittleEndian, &b); err != nil {
        return fmt.Errorf("serializar bloque: %v", err)
    }
    if _, err := disk.WriteAt(buf.Bytes(), blockOffset); err != nil {
        return fmt.Errorf("escribir bloque users.txt: %v", err)
    }

    ino.I_s = int32(len(data))
    var ibuf bytes.Buffer
    if err := binary.Write(&ibuf, binary.LittleEndian, &ino); err != nil {
        return fmt.Errorf("serializar inodo: %v", err)
    }
    if _, err := disk.WriteAt(ibuf.Bytes(), inodeOffset); err != nil {
        return fmt.Errorf("escribir inodo users.txt: %v", err)
    }
    return nil
}

func nuevoContenidoSeguro(s string) string {
    s = strings.ReplaceAll(s, "\r", "")
    if s == "" {
        return s
    }
    if !strings.HasSuffix(s, "\n") {
        s += "\n"
    }
    return s
}

func GrupoExiste(contenido string, grupo string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, linea := range lineas {
        linea = strings.TrimSpace(linea)
        if linea == "" {
            continue
        }
        campos := strings.Split(linea, ",")
        if len(campos) >= 3 {
            id := strings.TrimSpace(campos[0])
            tipo := strings.TrimSpace(campos[1])
            nombre := strings.TrimSpace(campos[2])
            // Ignorar grupos eliminados (id=0)
            if tipo == "G" && nombre == grupo && id != "0" {
                return true
            }
        }
    }
    return false
}

func ObtenerSiguienteIDGrupo(contenido string) int {
    maxID := 0
    for _, l := range strings.Split(contenido, "\n") {
        l = strings.TrimSpace(l)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 2 {
            continue
        }
        if strings.TrimSpace(p[1]) != "G" {
            continue
        }
        idStr := strings.TrimSpace(p[0])
        if idStr == "0" {
            continue
        }
        if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
            maxID = id
        }
    }
    return maxID + 1
}