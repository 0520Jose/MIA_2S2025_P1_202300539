package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "os"
    "strings"
    "io"

)

func CargarSistemaEXT2(partitionID string) (*os.File, *structs.SuperBloque, error) {
    var pm *structs.PartitionMount
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == partitionID {
            pm = &structs.Particiones_Montadas[i]
            break
        }
    }
    if pm == nil {
        return nil, nil, fmt.Errorf("partición con ID %s no está montada", partitionID)
    }

    f, err := os.OpenFile(pm.Path, os.O_RDWR, 0666)
    if err != nil {
        return nil, nil, fmt.Errorf("error al abrir disco: %v", err)
    }

    sb, err := CargarSuperBloque(f, pm.Partition.Part_start)
    if err != nil {
        f.Close()
        return nil, nil, err
    }
    return f, sb, nil
}

func CargarSuperBloque(f *os.File, partStart int32) (*structs.SuperBloque, error) {
    if _, err := f.Seek(int64(partStart), 0); err != nil {
        return nil, fmt.Errorf("error al posicionar superbloque: %v", err)
    }
    
    var sb structs.SuperBloque
    if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
        return nil, fmt.Errorf("error al leer superbloque: %v", err)
    }
    
    if sb.S_magic != 0xEF53 {
        return nil, fmt.Errorf("superbloque inválido: magic number incorrecto (0x%X)", sb.S_magic)
    }
    
    return &sb, nil
}

func ValidarSistemaEXT2(partitionID string) error {
    f, sb, err := CargarSistemaEXT2(partitionID)
    if err != nil {
        return err
    }
    defer f.Close()
    
    if sb.S_filesystem_type != 2 {
        return fmt.Errorf("sistema de archivos no es EXT2 (tipo: %d)", sb.S_filesystem_type)
    }
    
    if sb.S_inodes_count <= 0 || sb.S_blocks_count <= 0 {
        return fmt.Errorf("superbloque corrupto: contadores inválidos")
    }
    
    return nil
}

const (
    permRead  = 4
    permWrite = 2
    permExec  = 1
    usersInodeIndex = 2
)

func splitPathComponents(p string) []string {
    if p == "/" {
        return []string{}
    }
    return strings.Split(strings.Trim(p, "/"), "/")
}

func ensureParentDir(f *os.File, sb *structs.SuperBloque, parts []string, recursive bool) (int32, error) {
    currentIno := int32(0)

    for _, part := range parts {
        childIno, err := findEntryInDir(f, sb, currentIno, part)
        if err != nil {
            if !recursive {
                return -1, fmt.Errorf("directorio %s no existe y recursive no está habilitado", part)
            }
            
            childInoCreated, err := createDirectoryWithPerm(f, sb, currentIno, part, [3]byte{7, 7, 5})
            if err != nil {
                return -1, err
            }
            currentIno = childInoCreated
        } else {
            currentIno = int32(childIno)
        }
    }

    return currentIno, nil
}

func findEntryInDir(f *os.File, sb *structs.SuperBloque, dirIno int32, name string) (int, error) {
    ino, ok := structs.ObtenerInodo(f, sb, int(dirIno))
    if !ok {
        return -1, fmt.Errorf("no se pudo obtener inodo %d", dirIno)
    }

    for i := 0; i < 12; i++ {
        blockIdx := ino.I_block[i]
        if blockIdx < 0 {
            break
        }

        bc, ok := structs.LeerBloqueCarpeta(f, sb, blockIdx)
        if !ok {
            continue
        }

        for _, content := range bc.B_content {
            entryName := strings.TrimRight(string(content.B_name[:]), "\x00")
            if entryName == name {
                return int(content.B_inodo), nil
            }
        }
    }

    return -1, fmt.Errorf("entrada %s no encontrada", name)
}

func addDirEntry(f *os.File, sb *structs.SuperBloque, dirIno int32, name string, childIno int32) error {
    ino, ok := structs.ObtenerInodo(f, sb, int(dirIno))
    if !ok {
        return fmt.Errorf("no se pudo obtener inodo %d", dirIno)
    }

    for i := 0; i < 12; i++ {
        blockIdx := ino.I_block[i]
        if blockIdx < 0 {
            newBlockIdx, err := allocBlock(f, sb)
            if err != nil {
                return fmt.Errorf("no se pudo asignar nuevo bloque: %v", err)
            }
            
            ino.I_block[i] = newBlockIdx
            
            var bc structs.BCarpeta
            for j := range bc.B_content {
                bc.B_content[j].B_inodo = -1
                for k := range bc.B_content[j].B_name {
                    bc.B_content[j].B_name[k] = 0
                }
            }
            
            copy(bc.B_content[0].B_name[:], name)
            bc.B_content[0].B_inodo = childIno
            
            offset := int64(sb.S_block_start) + int64(newBlockIdx)*64
            if _, err := f.Seek(offset, io.SeekStart); err != nil {
                return err
            }
            if err := binary.Write(f, binary.LittleEndian, &bc); err != nil {
                return err
            }
            
            if err := writeInode(f, sb, dirIno, &ino); err != nil {
                return err
            }
            
            return nil
        }

        bc, ok := structs.LeerBloqueCarpeta(f, sb, blockIdx)
        if !ok {
            continue
        }

        for j := range bc.B_content {
            if bc.B_content[j].B_inodo == -1 {
                copy(bc.B_content[j].B_name[:], name)
                bc.B_content[j].B_inodo = childIno

                offset := int64(sb.S_block_start) + int64(blockIdx)*64
                if _, err := f.Seek(offset, io.SeekStart); err != nil {
                    return err
                }
                return binary.Write(f, binary.LittleEndian, &bc)
            }
        }
    }

    return fmt.Errorf("no hay espacio en el directorio")
}

func allocInode(f *os.File, sb *structs.SuperBloque) (int32, error) {
    bm := structs.BitMapInodos(f, sb)
    if bm == nil {
        return -1, fmt.Errorf("no se pudo leer bitmap de inodos")
    }

    idx := nextFreeIndex(bm)
    if idx == -1 {
        return -1, fmt.Errorf("no hay inodos disponibles")
    }

    bm[idx] = 1
    if _, err := f.Seek(int64(sb.S_bm_inode_start), io.SeekStart); err != nil {
        return -1, err
    }
    if _, err := f.Write(bm); err != nil {
        return -1, err
    }

    sb.S_free_inodes_count--
    return idx, nil
}

func allocBlock(f *os.File, sb *structs.SuperBloque) (int32, error) {
    bm := structs.GetBitmapBlocks(f, sb)
    if bm == nil {
        return -1, fmt.Errorf("no se pudo leer bitmap de bloques")
    }

    idx := nextFreeIndex(bm)
    if idx == -1 {
        return -1, fmt.Errorf("no hay bloques disponibles")
    }

    bm[idx] = 1
    if _, err := f.Seek(int64(sb.S_bm_block_start), io.SeekStart); err != nil {
        return -1, err
    }
    if _, err := f.Write(bm); err != nil {
        return -1, err
    }

    sb.S_free_blocks_count--
    return idx, nil
}

func nextFreeIndex(bm []byte) int32 {
    for i, b := range bm {
        if b == 0 {
            return int32(i)
        }
    }
    return -1
}

func readInode(f *os.File, sb *structs.SuperBloque, idx int32) (structs.Inodo, error) {
    var ino structs.Inodo
    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return ino, err
    }
    err := binary.Read(f, binary.LittleEndian, &ino)
    return ino, err
}

func getMountByID(id string) *structs.PartitionMount {
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == id {
            return &structs.Particiones_Montadas[i]
        }
    }
    return nil
}

func writeSuperBlock(f *os.File, sb *structs.SuperBloque, startPos int64) error {
    if _, err := f.Seek(startPos, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, sb)
}

func writeInode(f *os.File, sb *structs.SuperBloque, idx int32, ino *structs.Inodo) error {
    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, ino)
}

func writeDirBlock(f *os.File, sb *structs.SuperBloque, blockIdx int32, bc *structs.BCarpeta) error {
    offset := int64(sb.S_block_start) + int64(blockIdx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, bc)
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