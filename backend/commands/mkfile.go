package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "strings"
)

const (
    DIRECT_BLOCKS = 12
    INDIRECT_SIMPLE = 12
    INDIRECT_DOUBLE = 13
    INDIRECT_TRIPLE = 14
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
    if len(nombre) > 12 {
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
    ino.I_type[0] = 1
    ino.I_perm = [3]byte{6, 6, 4}

    if err := asignarBloquesArchivo(disk, sb, &ino, data); err != nil {
        return "Error al asignar bloques: " + err.Error()
    }

    if err := writeInode(disk, sb, inodeIdx, &ino); err != nil {
        return "Error al escribir inodo de archivo: " + err.Error()
    }

    if err := addDirEntry(disk, sb, padreIno, nombre, inodeIdx); err != nil {
        return "Error al agregar entrada en directorio: " + err.Error()
    }

    if err := writeSuperBlock(disk, sb, int64(pm.Partition.Part_start)); err != nil {
        return "Error al actualizar superbloque: " + err.Error()
    }

    return "Archivo creado exitosamente"
}

func asignarBloquesArchivo(f *os.File, sb *structs.SuperBloque, ino *structs.Inodo, data []byte) error {
    blocksNeeded := (len(data) + 63) / 64
    blockIndex := 0

    for i := 0; i < DIRECT_BLOCKS && blockIndex < blocksNeeded; i++ {
        blk, err := allocBlock(f, sb)
        if err != nil {
            return err
        }
        ino.I_block[i] = blk

        start := blockIndex * 64
        end := start + 64
        if end > len(data) {
            end = len(data)
        }

        var blockData structs.BArchivo
        copy(blockData.B_content[:], data[start:end])

        if err := structs.EscribirBloqueArchivo(f, sb, blk, &blockData); err != nil {
            return err
        }
        blockIndex++
    }

    if blockIndex < blocksNeeded {
        blk, err := allocBlock(f, sb)
        if err != nil {
            return err
        }
        ino.I_block[INDIRECT_SIMPLE] = blk

        var pointers structs.BApuntadores
        for i := range pointers.B_pointers {
            pointers.B_pointers[i] = -1
        }

        ptrIndex := 0
        for blockIndex < blocksNeeded && ptrIndex < 16 {
            dataBlk, err := allocBlock(f, sb)
            if err != nil {
                return err
            }
            pointers.B_pointers[ptrIndex] = dataBlk

            start := blockIndex * 64
            end := start + 64
            if end > len(data) {
                end = len(data)
            }

            var blockData structs.BArchivo
            copy(blockData.B_content[:], data[start:end])

            if err := structs.EscribirBloqueArchivo(f, sb, dataBlk, &blockData); err != nil {
                return err
            }
            blockIndex++
            ptrIndex++
        }

        if err := structs.EscribirBloqueApuntadores(f, sb, blk, &pointers); err != nil {
            return err
        }
    }

    if blockIndex < blocksNeeded {
        blk, err := allocBlock(f, sb)
        if err != nil {
            return err
        }
        ino.I_block[INDIRECT_DOUBLE] = blk

        var level1Pointers structs.BApuntadores
        for i := range level1Pointers.B_pointers {
            level1Pointers.B_pointers[i] = -1
        }

        level1Index := 0
        for blockIndex < blocksNeeded && level1Index < 16 {
            level2Blk, err := allocBlock(f, sb)
            if err != nil {
                return err
            }
            level1Pointers.B_pointers[level1Index] = level2Blk

            var level2Pointers structs.BApuntadores
            for i := range level2Pointers.B_pointers {
                level2Pointers.B_pointers[i] = -1
            }

            level2Index := 0
            for blockIndex < blocksNeeded && level2Index < 16 {
                dataBlk, err := allocBlock(f, sb)
                if err != nil {
                    return err
                }
                level2Pointers.B_pointers[level2Index] = dataBlk

                start := blockIndex * 64
                end := start + 64
                if end > len(data) {
                    end = len(data)
                }

                var blockData structs.BArchivo
                copy(blockData.B_content[:], data[start:end])

                if err := structs.EscribirBloqueArchivo(f, sb, dataBlk, &blockData); err != nil {
                    return err
                }
                blockIndex++
                level2Index++
            }

            if err := structs.EscribirBloqueApuntadores(f, sb, level2Blk, &level2Pointers); err != nil {
                return err
            }
            level1Index++
        }

        if err := structs.EscribirBloqueApuntadores(f, sb, blk, &level1Pointers); err != nil {
            return err
        }
    }

    return nil
}

func readPointerBlock(f *os.File, sb *structs.SuperBloque, blk int32) (structs.BApuntadores, error) {
    var pointers structs.BApuntadores
    off := int64(sb.S_block_start) + int64(blk)*int64(sb.S_block_s)
    if _, err := f.Seek(off, io.SeekStart); err != nil {
        return pointers, err
    }
    err := binary.Read(f, binary.LittleEndian, &pointers)
    return pointers, err
}

func writePointerBlock(f *os.File, sb *structs.SuperBloque, blk int32, pointers *structs.BApuntadores) error {
    off := int64(sb.S_block_start) + int64(blk)*int64(sb.S_block_s)
    if _, err := f.Seek(off, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, pointers)
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