package commands

import (
    "backend/structs"
    "fmt"
    "strings"
)

func Mkdir(params map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    
    rawPath, ok := params["-path"]
    if !ok || strings.TrimSpace(rawPath) == "" {
        return "Error: parámetro -path es obligatorio."
    }

    pflag := false
    if v, ok := params["-p"]; ok {
        if strings.TrimSpace(v) != "" {
            return "Error: -p no recibe valor."
        }
        pflag = true
    }

    ruta := unquoteValue(strings.TrimSpace(rawPath))
    if !strings.HasPrefix(ruta, "/") {
        return "Error: -path debe ser ruta absoluta."
    }
    partes := splitPathComponents(ruta)
    if len(partes) == 0 {
        return "Error: ruta inválida."
    }
    nombre := partes[len(partes)-1]
    if nombre == "" {
        return "Error: nombre de carpeta vacío."
    }
    if len(nombre) > len(structs.BContent{}.B_name) {
        return "Error: nombre de carpeta excede 12 caracteres."
    }

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()
    
    pm := getMountByID(usuarioActual.PartitionID)
    if pm == nil {
        return "Error: partición no montada."
    }

    padreIno, err := ensureParentDir(disk, sb, partes[:len(partes)-1], pflag)
    if err != nil {
        return "Error: " + err.Error()
    }

    dirPadre, err := readInode(disk, sb, padreIno)
    if err != nil {
        return "Error: " + err.Error()
    }
    if !Permisos(&dirPadre, permWrite) {
        return "Error: permiso denegado en carpeta padre."
    }

    if childIdx, _ := findEntryInDir(disk, sb, padreIno, nombre); childIdx >= 0 {
        child, err := readInode(disk, sb, int32(childIdx))
        if err != nil {
            return "Error: " + err.Error()
        }
        if child.I_type[0] == 0 {
            return "Carpeta ya existe"
        }
        return "Error: ya existe un archivo con ese nombre."
    }

    if _, err := createDirectoryWithPerm(disk, sb, padreIno, nombre, [3]byte{6, 6, 4}); err != nil {
        return "Error: " + err.Error()
    }

    if err := writeSuperBlock(disk, sb, int64(pm.Partition.Part_start)); err != nil {
        return "Error al actualizar superbloque: " + err.Error()
    }

    return "Carpeta creada exitosamente"
}