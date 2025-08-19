package commands

import (
    "backend/structs"
    "bytes"
    "encoding/binary"
    "os"
    "time"
    "unsafe"
)

func Mkfs(params map[string]string) string {
    id, existe := params["-id"]
    if !existe {
        return "Error: parámetro -id es obligatorio"
    }

    Type := "full"
    if t, existe := params["-type"]; existe {
        Type = t
        if Type != "full" {
            return "Error: -type debe ser 'full'"
        }
    }

    part := structs.GetMountedPartitionByID(id, structs.MBR{})
    if part.Part_s <= 0 || part.Part_start < 0 {
        return "Error: partición no encontrada o inválida con ID " + id
    }
    
    var partidaMontada *structs.PartitionMount
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == id {
            partidaMontada = &structs.Particiones_Montadas[i]
            break
        }
    }
    if partidaMontada == nil {
        return "Error: partición con ID " + id + " no está montada"
    }

    diskPath := structs.GetDiskPathByID(id)
    if diskPath == "" {
        return "Error: no se pudo obtener la ruta del disco para ID " + id
    }

    if _, err := os.Stat(diskPath); os.IsNotExist(err) {
        return "Error: el archivo de disco no existe: " + diskPath
    }

    diskFile, err := os.OpenFile(diskPath, os.O_RDWR, 0666)
    if err != nil {
        return "Error al abrir el disco: " + err.Error()
    }
    defer diskFile.Close()

    if Type == "full" {
        zero := make([]byte, part.Part_s)
        _, err := diskFile.WriteAt(zero, int64(part.Part_start))
        if err != nil {
            return "Error al limpiar la partición: " + err.Error()
        }
        err = diskFile.Sync()
        if err != nil {
            return "Error al sincronizar limpieza: " + err.Error()
        }
    }

    n := int32(part.Part_s)
    sbSize := int32(unsafe.Sizeof(structs.SuperBloque{}))
    inodoSize := int32(unsafe.Sizeof(structs.Inodo{}))
    bloqueSize := int32(unsafe.Sizeof(structs.BArchivo{}))

    if n < sbSize+100 {
        return "Error: partición muy pequeña (tamaño: " + string(rune(n)) + " bytes)"
    }

    espacioDisponible := n - sbSize
    bytesPerInode := int32(1 + 3 + inodoSize + 3*bloqueSize)
    numInodos := espacioDisponible / bytesPerInode
    
    if numInodos < 16 {
        numInodos = 16
    }
    if numInodos > 1024 {
        numInodos = 1024
    }
    
    numBloques := numInodos * 3
    if numBloques < 32 {
        numBloques = 32
    }

    espacioRequerido := sbSize + numInodos + numBloques + (numInodos * inodoSize) + (numBloques * bloqueSize)
    if espacioRequerido > n {
        factor := float32(n) / float32(espacioRequerido)
        numInodos = int32(float32(numInodos) * factor * 0.9)
        numBloques = numInodos * 3
        espacioRequerido = sbSize + numInodos + numBloques + (numInodos * inodoSize) + (numBloques * bloqueSize)
        
        if espacioRequerido > n {
            return "Error: no hay espacio suficiente"
        }
    }

    sb := structs.SuperBloque{}
    sb.S_filesystem_type = 2
    sb.S_inodes_count = numInodos
    sb.S_blocks_count = numBloques
    sb.S_free_inodes_count = numInodos - 2
    sb.S_free_blocks_count = numBloques - 2
    sb.S_mnt_count = 1
    sb.S_magic = 0xEF53
    sb.S_inode_s = inodoSize
    sb.S_block_s = bloqueSize
    sb.S_first_ino = 2
    sb.S_first_blo = 2
    sb.S_bm_inode_start = part.Part_start + sbSize
    sb.S_bm_block_start = part.Part_start + sbSize + numInodos
    sb.S_inode_start = part.Part_start + sbSize + numInodos + numBloques
    sb.S_block_start = part.Part_start + sbSize + numInodos + numBloques + (numInodos * inodoSize)
    timeStr := time.Now().Format("2006-01-02 15:04:05")
    copy(sb.S_mtime[:], timeStr)

    var buf bytes.Buffer
    binary.Write(&buf, binary.LittleEndian, &sb)
    sbBytes := buf.Bytes()
    if int32(len(sbBytes)) != sbSize {
        padded := make([]byte, sbSize)
        copy(padded, sbBytes)
        sbBytes = padded
    }
    diskFile.WriteAt(sbBytes, int64(part.Part_start))
    diskFile.Sync()

    for i := int32(0); i < numInodos; i++ {
        diskFile.WriteAt([]byte{0}, int64(sb.S_bm_inode_start+i))
    }
    for i := int32(0); i < numBloques; i++ {
        diskFile.WriteAt([]byte{0}, int64(sb.S_bm_block_start+i))
    }

    var inodo0 structs.Inodo
    inodo0.I_uid = 1
    inodo0.I_gid = 1
    copy(inodo0.I_atime[:], timeStr)
    copy(inodo0.I_ctime[:], timeStr)
    copy(inodo0.I_mtime[:], timeStr)
    copy(inodo0.I_perm[:], "664")
    for i := range inodo0.I_block {
        inodo0.I_block[i] = -1
    }
    inodo0.I_block[0] = 0

    var inodo1 structs.Inodo
    inodo1.I_uid = 1
    inodo1.I_gid = 1
    contenido := []byte("1,G,root\n1,U,root,root,123\n")
    inodo1.I_s = int32(len(contenido))
    copy(inodo1.I_atime[:], timeStr)
    copy(inodo1.I_ctime[:], timeStr)
    copy(inodo1.I_mtime[:], timeStr)
    copy(inodo1.I_perm[:], "664")
    for i := range inodo1.I_block {
        inodo1.I_block[i] = -1
    }
    inodo1.I_block[0] = 1

    var bloqueCarpeta structs.BCarpeta
    copy(bloqueCarpeta.B_content[0].B_name[:], ".")
    bloqueCarpeta.B_content[0].B_inodo = 0
    copy(bloqueCarpeta.B_content[1].B_name[:], "..")
    bloqueCarpeta.B_content[1].B_inodo = 0
    copy(bloqueCarpeta.B_content[2].B_name[:], "users.txt")
    bloqueCarpeta.B_content[2].B_inodo = 1

    var bloqueArchivo structs.BArchivo
    copy(bloqueArchivo.B_content[:], contenido)

    diskFile.WriteAt([]byte{1}, int64(sb.S_bm_inode_start))
    diskFile.WriteAt([]byte{1}, int64(sb.S_bm_inode_start+1))
    diskFile.WriteAt([]byte{1}, int64(sb.S_bm_block_start))
    diskFile.WriteAt([]byte{1}, int64(sb.S_bm_block_start+1))

    buf.Reset()
    binary.Write(&buf, binary.LittleEndian, &inodo0)
    diskFile.WriteAt(buf.Bytes(), int64(sb.S_inode_start))
    buf.Reset()
    binary.Write(&buf, binary.LittleEndian, &inodo1)
    diskFile.WriteAt(buf.Bytes(), int64(sb.S_inode_start+inodoSize))

    buf.Reset()
    binary.Write(&buf, binary.LittleEndian, &bloqueCarpeta)
    diskFile.WriteAt(buf.Bytes(), int64(sb.S_block_start))
    buf.Reset()
    binary.Write(&buf, binary.LittleEndian, &bloqueArchivo)
    diskFile.WriteAt(buf.Bytes(), int64(sb.S_block_start+bloqueSize))

    return "Sistema de archivos creado exitosamente en la partición " + string(part.Part_name[:]) + " con ID " + id
}
