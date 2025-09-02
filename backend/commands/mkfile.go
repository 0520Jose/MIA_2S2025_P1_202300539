package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

func Mkfile(params map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    rawPath, ok := params["-path"]
    if !ok || strings.TrimSpace(rawPath) == "" {
        return "Error: parámetro -path es obligatorio."
    }
    rflag := false
    if _, ok := params["-r"]; ok {
        if strings.TrimSpace(params["-r"]) != "" {
            return "Error: -r no recibe valor."
        }
        rflag = true
    }

    var data []byte
    if cont, ok := params["-cont"]; ok && strings.TrimSpace(cont) != "" {
        hostPath := unquoteValue(strings.TrimSpace(cont))
        b, err := os.ReadFile(hostPath)
        if err != nil {
            return "Error: no se pudo leer -cont: " + err.Error()
        }
        data = b
    } else {
        size := 0
        if s, ok := params["-size"]; ok && strings.TrimSpace(s) != "" {
            if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &size); err != nil {
                return "Error: -size inválido."
            }
            if size < 0 {
                return "Error: -size no puede ser negativo."
            }
        }
        if size > 0 {
            data = make([]byte, size)
            for i := 0; i < size; i++ {
                data[i] = byte('0' + (i % 10))
            }
        } else {
            data = []byte{}
        }
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
        return "Error: nombre de archivo vacío."
    }
    if len(nombre) > len(structs.BContent{}.B_name) {
        return "Error: nombre de archivo excede 12 caracteres."
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

    padreIno, err := ensureParentDir(disk, sb, partes[:len(partes)-1], rflag)
    if err != nil {
        return "Error: " + err.Error()
    }

    dirIno, err := readInode(disk, sb, padreIno)
    if err != nil {
        return "Error: " + err.Error()
    }
    if !Permisos(&dirIno, permWrite|permExec) {
        return "Error: permiso denegado en carpeta padre."
    }

    if idx, _ := findEntryInDir(disk, sb, padreIno, nombre); idx >= 0 {
        return "Error: el archivo ya existe."
    }

    inodeIdx, err := allocInode(disk, sb)
    if err != nil {
        return "Error al reservar inodo: " + err.Error()
    }

    blocksNeeded := 0
    if len(data) > 0 {
        blocksNeeded = (len(data) + len(structs.BArchivo{}.B_content) - 1) / len(structs.BArchivo{}.B_content)
    }

    usadoBlocks := make([]int32, 0, blocksNeeded)
    offset := 0
    for i := 0; i < blocksNeeded; i++ {
        blk, err := allocBlock(disk, sb)
        if err != nil {
            return "Error al reservar bloque: " + err.Error()
        }
        usadoBlocks = append(usadoBlocks, blk)
        chunk := len(structs.BArchivo{}.B_content)
        if offset+chunk > len(data) {
            chunk = len(data) - offset
        }
        var b structs.BArchivo
        for j := range b.B_content {
            b.B_content[j] = 0
        }
        copy(b.B_content[:], data[offset:offset+chunk])
        if err := writeFileBlock(disk, sb, blk, &b); err != nil {
            return "Error al escribir bloque de datos: " + err.Error()
        }
        offset += chunk
    }

    var ino structs.Inodo
    ino.I_uid = int32(usuarioActual.UID)
    ino.I_gid = int32(usuarioActual.GID)
    ino.I_s = int32(len(data))
    t := fecha17()
    copy(ino.I_atime[:], t)
    copy(ino.I_ctime[:], t)
    copy(ino.I_mtime[:], t)
    for i := range ino.I_block {
        ino.I_block[i] = -1
    }
    for i := 0; i < len(usadoBlocks) && i < len(ino.I_block); i++ {
        ino.I_block[i] = usadoBlocks[i]
    }
    ino.I_type[0] = 1
    ino.I_perm = [3]byte{6, 6, 4}

    if err := writeInode(disk, sb, inodeIdx, &ino); err != nil {
        return "Error al escribir inodo de archivo: " + err.Error()
    }

    if err := addDirEntry(disk, sb, padreIno, nombre, inodeIdx); err != nil {
        return "Error al agregar entrada en directorio: " + err.Error()
    }

    if err := writeSuperBlock(disk, pm.Partition.Part_start, sb); err != nil {
        return "Error al actualizar superbloque: " + err.Error()
    }

    return "Archivo creado exitosamente"
}

func splitPathComponents(p string) []string {
    p = filepath.Clean(p)
    parts := strings.Split(p, "/")
    res := make([]string, 0, len(parts))
    for _, s := range parts {
        if s == "" {
            continue
        }
        res = append(res, s)
    }
    return res
}

func ensureParentDir(f *os.File, sb *structs.SuperBloque, parts []string, recursive bool) (int32, error) {
    curr := int32(0)
    for _, name := range parts {
        if strings.TrimSpace(name) == "" {
            continue
        }
        idx, err := findEntryInDir(f, sb, curr, name)
        if err != nil {
            return -1, err
        }
        if idx >= 0 {
            ino, err := readInode(f, sb, int32(idx))
            if err != nil {
                return -1, err
            }
            if ino.I_type[0] != 0 {
                return -1, fmt.Errorf("ya existe un archivo con el nombre '%s'", name)
            }
            curr = int32(idx)
            continue
        }
        if !recursive {
            return -1, fmt.Errorf("carpeta padre '%s' no existe", name)
        }
        parentIno, err := readInode(f, sb, curr)
        if err != nil {
            return -1, err
        }
        if !Permisos(&parentIno, permWrite) && !EsRoot() {
            return -1, fmt.Errorf("permiso denegado para crear carpeta '%s'", name)
        }
        newIno, err := createDirectory(f, sb, curr, name)
        if err != nil {
            return -1, err
        }
        curr = newIno
    }
    return curr, nil
}

func findEntryInDir(f *os.File, sb *structs.SuperBloque, dirIno int32, name string) (int, error) {
    ino, err := readInode(f, sb, dirIno)
    if err != nil {
        return -1, err
    }
    for _, b := range ino.I_block {
        if b < 0 {
            continue
        }
        dir, err := readDirBlock(f, sb, b)
        if err != nil {
            return -1, err
        }
        for _, e := range dir.B_content {
            if e.B_inodo < 0 {
                continue
            }
            en := strings.TrimRight(string(e.B_name[:]), "\x00")
            if en == name {
                return int(e.B_inodo), nil
            }
        }
    }
    return -1, nil
}

func createDirectory(f *os.File, sb *structs.SuperBloque, parentIno int32, name string) (int32, error) {
    idxIno, err := allocInode(f, sb)
    if err != nil {
        return -1, err
    }
    idxBlk, err := allocBlock(f, sb)
    if err != nil {
        return -1, err
    }

    var ino structs.Inodo
    ino.I_uid = int32(usuarioActual.UID)
    ino.I_gid = int32(usuarioActual.GID)
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
    ino.I_perm = [3]byte{7, 5, 5}
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

func addDirEntry(f *os.File, sb *structs.SuperBloque, dirIno int32, name string, childIno int32) error {
    ino, err := readInode(f, sb, dirIno)
    if err != nil {
        return err
    }
    for bi := 0; bi < len(ino.I_block); bi++ {
        b := ino.I_block[bi]
        if b < 0 {
            nb, err := allocBlock(f, sb)
            if err != nil {
                return err
            }
            var newDir structs.BCarpeta
            for i := range newDir.B_content {
                newDir.B_content[i].B_inodo = -1
                for j := range newDir.B_content[i].B_name {
                    newDir.B_content[i].B_name[j] = 0
                }
            }
            ino.I_block[bi] = nb
            if err := writeInode(f, sb, dirIno, &ino); err != nil {
                return err
            }
            if err := writeDirBlock(f, sb, nb, &newDir); err != nil {
                return err
            }
            b = nb
        }
        dir, err := readDirBlock(f, sb, b)
        if err != nil {
            return err
        }
        for i := range dir.B_content {
            if dir.B_content[i].B_inodo < 0 {
                dir.B_content[i].B_inodo = childIno
                for j := range dir.B_content[i].B_name {
                    dir.B_content[i].B_name[j] = 0
                }
                copy(dir.B_content[i].B_name[:], name)
                if err := writeDirBlock(f, sb, b, &dir); err != nil {
                    return err
                }
                return nil
            }
        }
    }
    return fmt.Errorf("directorio sin espacio para nuevas entradas")
}

func allocInode(f *os.File, sb *structs.SuperBloque) (int32, error) {
    bm := make([]byte, sb.S_inodes_count)
    if _, err := f.ReadAt(bm, int64(sb.S_bm_inode_start)); err != nil {
        return -1, err
    }
    for i := 0; i < int(sb.S_inodes_count); i++ {
        if bm[i] == 0 {
            bm[i] = 1
            if _, err := f.WriteAt(bm, int64(sb.S_bm_inode_start)); err != nil {
                return -1, err
            }
            sb.S_free_inodes_count--
            sb.S_first_ino = nextFreeIndex(bm)
            return int32(i), nil
        }
    }
    return -1, fmt.Errorf("No hay inodos disponibles")
}

func allocBlock(f *os.File, sb *structs.SuperBloque) (int32, error) {
    bm := make([]byte, sb.S_blocks_count)
    if _, err := f.ReadAt(bm, int64(sb.S_bm_block_start)); err != nil {
        return -1, err
    }
    for i := 0; i < int(sb.S_blocks_count); i++ {
        if bm[i] == 0 {
            bm[i] = 1
            if _, err := f.WriteAt(bm, int64(sb.S_bm_block_start)); err != nil {
                return -1, err
            }
            sb.S_free_blocks_count--
            sb.S_first_blo = nextFreeIndex(bm)
            return int32(i), nil
        }
    }
    return -1, fmt.Errorf("No hay bloques disponibles")
}

func nextFreeIndex(bm []byte) int32 {
    for i := 0; i < len(bm); i++ {
        if bm[i] == 0 {
            return int32(i)
        }
    }
    return -1
}

func writeInode(f *os.File, sb *structs.SuperBloque, idx int32, ino *structs.Inodo) error {
    off := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(off, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, ino)
}

func writeDirBlock(f *os.File, sb *structs.SuperBloque, blk int32, dir *structs.BCarpeta) error {
    off := int64(sb.S_block_start) + int64(blk)*int64(sb.S_block_s)
    if _, err := f.Seek(off, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, dir)
}

func writeFileBlock(f *os.File, sb *structs.SuperBloque, blk int32, b *structs.BArchivo) error {
    off := int64(sb.S_block_start) + int64(blk)*int64(sb.S_block_s)
    if _, err := f.Seek(off, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, b)
}

func writeSuperBlock(f *os.File, partStart int32, sb *structs.SuperBloque) error {
    if _, err := f.Seek(int64(partStart), io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, sb)
}

func getMountByID(id string) *structs.PartitionMount {
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == id {
            return &structs.Particiones_Montadas[i]
        }
    }
    return nil
}