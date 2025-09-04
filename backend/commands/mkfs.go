package commands

import (
    "backend/structs"
    "bytes"
    "encoding/binary"
    "fmt"
    "os"
    "strings"
    "time"
)

func Mkfs(params map[string]string) string {
    allowed := map[string]struct{}{"-id": {}, "-type": {}}
    normalized := make(map[string]string, len(params))
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if _, ok := allowed[lk]; !ok {
            return fmt.Sprintf("Error: parámetro no reconocido: %s", k)
        }
        normalized[lk] = v
    }
    id := strings.TrimSpace(normalized["-id"])
    if id == "" {
        return "Error: parámetro -id es obligatorio"
    }
    tipo := "full"
    if t, ok := normalized["-type"]; ok && strings.TrimSpace(t) != "" {
        if strings.ToLower(strings.TrimSpace(t)) != "full" {
            return "Error: tipo de formateo no válido, solo 'full'"
        }
    }

    var pm *structs.PartitionMount
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == id {
            pm = &structs.Particiones_Montadas[i]
            break
        }
    }
    if pm == nil {
        return fmt.Sprintf("Error: no existe una partición montada con id '%s'", id)
    }

    f, err := os.OpenFile(pm.Path, os.O_RDWR, 0666)
    if err != nil {
        return "Error al abrir el disco: " + err.Error()
    }
    defer f.Close()

    inicio := pm.Partition.Part_start
    tam := pm.Partition.Part_s
    if tam <= 0 {
        return "Error: tamaño de partición inválido"
    }

    if tipo == "full" {
        zero := make([]byte, 1024*1024)
        remaining := int64(tam)
        pos := int64(inicio)
        for remaining > 0 {
            chunk := int64(len(zero))
            if chunk > remaining {
                chunk = remaining
            }
            if _, err := f.WriteAt(zero[:chunk], pos); err != nil {
                return "Error al limpiar la partición: " + err.Error()
            }
            pos += chunk
            remaining -= chunk
        }
    }

    inodeSize := int32(binary.Size(structs.Inodo{}))
    blockSize := int32(binary.Size(structs.BArchivo{}))
    sbSize := int32(binary.Size(structs.SuperBloque{}))

    numerador := tam - sbSize
    denominador := int32(4) + inodeSize + 3*blockSize
    if denominador <= 0 || numerador <= denominador {
        return "Error: partición demasiado pequeña para EXT2"
    }
    n := numerador / denominador
    if n < 3 {
        n = 3
    }

    var sb structs.SuperBloque
    sb.S_filesystem_type = 2
    sb.S_magic = 0xEF53
    copy(sb.S_mtime[:], fecha17())
    copy(sb.S_umtime[:], fecha17())
    sb.S_mnt_count = 1

    sb.S_inodes_count = n
    sb.S_blocks_count = 3 * n
    sb.S_inode_s = inodeSize
    sb.S_block_s = blockSize

    sb.S_bm_inode_start = inicio + sbSize
    sb.S_bm_block_start = sb.S_bm_inode_start + sb.S_inodes_count
    sb.S_inode_start = sb.S_bm_block_start + sb.S_blocks_count
    sb.S_block_start = sb.S_inode_start + sb.S_inodes_count*sb.S_inode_s

    usedInodes := int32(3)
    usedBlocks := int32(3)
    sb.S_free_inodes_count = sb.S_inodes_count - usedInodes
    sb.S_free_blocks_count = sb.S_blocks_count - usedBlocks
    sb.S_first_ino = usedInodes
    sb.S_first_blo = usedBlocks

    if _, err := f.Seek(int64(inicio), 0); err != nil {
        return "Error al posicionar superbloque: " + err.Error()
    }
    if err := binary.Write(f, binary.LittleEndian, &sb); err != nil {
        return "Error al escribir superbloque: " + err.Error()
    }

    if _, err := f.Seek(int64(sb.S_bm_inode_start), 0); err != nil {
        return "Error al posicionar bm inodos: " + err.Error()
    }
    bmInodos := bytes.Repeat([]byte{0}, int(sb.S_inodes_count))
    bmInodos[0] = 1
    bmInodos[1] = 1
    bmInodos[2] = 1
    if _, err := f.Write(bmInodos); err != nil {
        return "Error al escribir bm inodos: " + err.Error()
    }

    if _, err := f.Seek(int64(sb.S_bm_block_start), 0); err != nil {
        return "Error al posicionar bm bloques: " + err.Error()
    }
    bmBloques := bytes.Repeat([]byte{0}, int(sb.S_blocks_count))
    bmBloques[0] = 1
    bmBloques[1] = 1
    bmBloques[2] = 1
    if _, err := f.Write(bmBloques); err != nil {
        return "Error al escribir bm bloques: " + err.Error()
    }

    var inoRoot structs.Inodo
    inoRoot.I_uid = 1
    inoRoot.I_gid = 1
    inoRoot.I_s = int32(binary.Size(structs.BCarpeta{}))
    copy(inoRoot.I_atime[:], fecha17())
    copy(inoRoot.I_ctime[:], fecha17())
    copy(inoRoot.I_mtime[:], fecha17())
    for i := range inoRoot.I_block {
        inoRoot.I_block[i] = -1
    }
    inoRoot.I_block[0] = 0
    inoRoot.I_type[0] = 0
    inoRoot.I_perm = [3]byte{7, 5, 5}

    var inoHome structs.Inodo
    inoHome.I_uid = 1
    inoHome.I_gid = 1
    inoHome.I_s = int32(binary.Size(structs.BCarpeta{}))
    copy(inoHome.I_atime[:], fecha17())
    copy(inoHome.I_ctime[:], fecha17())
    copy(inoHome.I_mtime[:], fecha17())
    for i := range inoHome.I_block {
        inoHome.I_block[i] = -1
    }
    inoHome.I_block[0] = 1
    inoHome.I_type[0] = 0
    inoHome.I_perm = [3]byte{7, 5, 5}

    contenidoUsers := "1,G,root\n1,U,root,root,123\n"
    var inoUsers structs.Inodo
    inoUsers.I_uid = 1
    inoUsers.I_gid = 1
    inoUsers.I_s = int32(len(contenidoUsers))
    copy(inoUsers.I_atime[:], fecha17())
    copy(inoUsers.I_ctime[:], fecha17())
    copy(inoUsers.I_mtime[:], fecha17())
    for i := range inoUsers.I_block {
        inoUsers.I_block[i] = -1
    }
    inoUsers.I_block[0] = 2
    inoUsers.I_type[0] = 1
    inoUsers.I_perm = [3]byte{6, 6, 4}

    if _, err := f.Seek(int64(sb.S_inode_start)+0*int64(sb.S_inode_s), 0); err != nil {
        return "Error al posicionar inodo raíz: " + err.Error()
    }
    if err := binary.Write(f, binary.LittleEndian, &inoRoot); err != nil {
        return "Error al escribir inodo raíz: " + err.Error()
    }
    
    if _, err := f.Seek(int64(sb.S_inode_start)+1*int64(sb.S_inode_s), 0); err != nil {
        return "Error al posicionar inodo home: " + err.Error()
    }
    if err := binary.Write(f, binary.LittleEndian, &inoHome); err != nil {
        return "Error al escribir inodo home: " + err.Error()
    }
    
    if _, err := f.Seek(int64(sb.S_inode_start)+2*int64(sb.S_inode_s), 0); err != nil {
        return "Error al posicionar inodo users.txt: " + err.Error()
    }
    if err := binary.Write(f, binary.LittleEndian, &inoUsers); err != nil {
        return "Error al escribir inodo users.txt: " + err.Error()
    }

    var bdir structs.BCarpeta
    for i := range bdir.B_content {
        bdir.B_content[i].B_inodo = -1
        for j := range bdir.B_content[i].B_name {
            bdir.B_content[i].B_name[j] = 0
        }
    }
    bdir.B_content[0].B_inodo = 0
    copy(bdir.B_content[0].B_name[:], ".")
    bdir.B_content[1].B_inodo = 0
    copy(bdir.B_content[1].B_name[:], "..")
    bdir.B_content[2].B_inodo = 1
    copy(bdir.B_content[2].B_name[:], "home")

    if _, err := f.Seek(int64(sb.S_block_start)+0*int64(sb.S_block_s), 0); err != nil {
        return "Error al posicionar bloque carpeta raíz: " + err.Error()
    }
    if err := binary.Write(f, binary.LittleEndian, &bdir); err != nil {
        return "Error al escribir bloque carpeta raíz: " + err.Error()
    }

    var bhome structs.BCarpeta
    for i := range bhome.B_content {
        bhome.B_content[i].B_inodo = -1
        for j := range bhome.B_content[i].B_name {
            bhome.B_content[i].B_name[j] = 0
        }
    }
    bhome.B_content[0].B_inodo = 1
    copy(bhome.B_content[0].B_name[:], ".")
    bhome.B_content[1].B_inodo = 0
    copy(bhome.B_content[1].B_name[:], "..")
    bhome.B_content[2].B_inodo = 2
    copy(bhome.B_content[2].B_name[:], "users.txt")

    if _, err := f.Seek(int64(sb.S_block_start)+1*int64(sb.S_block_s), 0); err != nil {
        return "Error al posicionar bloque carpeta home: " + err.Error()
    }
    if err := binary.Write(f, binary.LittleEndian, &bhome); err != nil {
        return "Error al escribir bloque carpeta home: " + err.Error()
    }

    var bUsers structs.BArchivo
    copy(bUsers.B_content[:], []byte(contenidoUsers))
    if _, err := f.Seek(int64(sb.S_block_start)+2*int64(sb.S_block_s), 0); err != nil {
        return "Error al posicionar bloque users.txt: " + err.Error()
    }
    if err := binary.Write(f, binary.LittleEndian, &bUsers); err != nil {
        return "Error al escribir bloque users.txt: " + err.Error()
    }

    if err := ValidarSistemaEXT2(id); err != nil {
        return fmt.Sprintf("Error: formateo falló - %v", err)
    }

    return fmt.Sprintf("Sistema de archivos EXT2 creado correctamente en la partición con ID %s", id)
}

func fecha17() string {
    return time.Now().Format("2006-01-02 15:04:05.999999999")
}