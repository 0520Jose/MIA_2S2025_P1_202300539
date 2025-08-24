package commands

import (
    "backend/structs"
    "fmt"
    "os"
    "strings"
)

func Mkdir(params map[string]string) string {
    if currentUser == nil {
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

    disk, sb, err := CargarSistemaEXT2(currentUser.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()
    pm := getMountByID(currentUser.PartitionID)
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
    if !hasPerm(&dirPadre, permWrite) {
        return "Error: permiso denegado en carpeta padre."
    }

    if childIdx, _ := findEntryInDir(disk, sb, padreIno, nombre); childIdx >= 0 {
        child, err := readInode(disk, sb, int32(childIdx))
        if err != nil {
            return "Error: " + err.Error()
        }
        if child.I_type[0] == '0' {
            return "Carpeta ya existe"
        }
        return "Error: ya existe un archivo con ese nombre."
    }

    if _, err := createDirectoryWithPerm(disk, sb, padreIno, nombre, [3]byte{6, 6, 4}); err != nil {
        return "Error: " + err.Error()
    }

    if err := writeSuperBlock(disk, pm.Partition.Part_start, sb); err != nil {
        return "Error al actualizar superbloque: " + err.Error()
    }

    return "Carpeta creada exitosamente"
}

func createDirectoryWithPerm(f *os.File, sb *structs.SuperBloque, parentIno int32, name string, perm [3]byte) (int32, error) {
    idxIno, err := allocInode(f, sb)
    if err != nil {
        return -1, err
    }
    idxBlk, err := allocBlock(f, sb)
    if err != nil {
        return -1, err
    }

    var ino structs.Inodo
    ino.I_uid = int32(currentUser.UID)
    ino.I_gid = int32(currentUser.GID)
    ino.I_s = 0
    t := fecha17()
    copy(ino.I_atime[:], t)
    copy(ino.I_ctime[:], t)
    copy(ino.I_mtime[:], t)
    for i := range ino.I_block {
        ino.I_block[i] = -1
    }
    ino.I_block[0] = idxBlk
    ino.I_type[0] = 0
    ino.I_perm = perm
    if err := writeInode(f, sb, idxIno, &ino); err != nil {
        return -1, err
    }

    var dir structs.BCarpeta
    for i := range dir.B_content {
        dir.B_content[i].B_inodo = -1
        for j := range dir.B_content[i].B_name {
            dir.B_content[i].B_name[j] = 0
        }
    }
    dir.B_content[0].B_inodo = idxIno
    copy(dir.B_content[0].B_name[:], ".")
    dir.B_content[1].B_inodo = parentIno
    copy(dir.B_content[1].B_name[:], "..")
    if err := writeDirBlock(f, sb, idxBlk, &dir); err != nil {
        return -1, err
    }

    if err := addDirEntry(f, sb, parentIno, name, idxIno); err != nil {
        return -1, err
    }
    return idxIno, nil
}