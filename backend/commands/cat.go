package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "os"
    "sort"
    "strings"
)

func Cat(params map[string]string) string {
    sess := GetCurrentUser()
    if sess == nil {
        return "Error: No hay una sesión activa."
    }

    files := make([]string, 0)
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if lk == "-file" || strings.HasPrefix(lk, "-file") {
            p := strings.TrimSpace(v)
            if p != "" {
                files = append(files, p)
            }
        }
    }
    if len(files) == 0 {
        return "Error: debe especificar al menos un parámetro -file o -fileN."
    }
    sort.Strings(files)

    var pm *structs.PartitionMount
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == sess.PartitionID {
            pm = &structs.Particiones_Montadas[i]
            break
        }
    }
    if pm == nil {
        return "Error: la partición de la sesión no está montada."
    }

    f, err := os.Open(pm.Path)
    if err != nil {
        return fmt.Sprintf("Error al abrir el disco: %v", err)
    }
    defer f.Close()

    sb, err := readSuperBlock(f, pm.Partition.Part_start)
    if err != nil {
        return fmt.Sprintf("Error al leer superbloque: %v", err)
    }

    var out []string
    for _, path := range files {
        if !strings.HasPrefix(path, "/") {
            out = append(out, fmt.Sprintf("Error: ruta inválida '%s' (debe iniciar con /)", path))
            continue
        }
        inoIdx, err := findInodeByPath(f, sb, path)
        if err != nil {
            out = append(out, fmt.Sprintf("Error: %s -> %v", path, err))
            continue
        }
        content, err := readFileContent(f, sb, inoIdx)
        if err != nil {
            out = append(out, fmt.Sprintf("Error: %s -> %v", path, err))
            continue
        }
        out = append(out, content)
    }
    return strings.Join(out, "\n")
}

func readSuperBlock(f *os.File, partStart int32) (*structs.SuperBloque, error) {
    if _, err := f.Seek(int64(partStart), 0); err != nil {
        return nil, err
    }
    var sb structs.SuperBloque
    if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
        return nil, err
    }
    if sb.S_magic != 0xEF53 {
        return nil, fmt.Errorf("FS inválido (magic=0x%X)", sb.S_magic)
    }
    return &sb, nil
}

func readInode(f *os.File, sb *structs.SuperBloque, idx int32) (structs.Inodo, error) {
    var ino structs.Inodo
    off := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(off, 0); err != nil {
        return ino, err
    }
    if err := binary.Read(f, binary.LittleEndian, &ino); err != nil {
        return ino, err
    }
    return ino, nil
}

func readDirBlock(f *os.File, sb *structs.SuperBloque, blk int32) (structs.BCarpeta, error) {
    var dir structs.BCarpeta
    off := int64(sb.S_block_start) + int64(blk)*int64(sb.S_block_s)
    if _, err := f.Seek(off, 0); err != nil {
        return dir, err
    }
    if err := binary.Read(f, binary.LittleEndian, &dir); err != nil {
        return dir, err
    }
    return dir, nil
}

func readFileBlock(f *os.File, sb *structs.SuperBloque, blk int32) (structs.BArchivo, error) {
    var fb structs.BArchivo
    off := int64(sb.S_block_start) + int64(blk)*int64(sb.S_block_s)
    if _, err := f.Seek(off, 0); err != nil {
        return fb, err
    }
    if err := binary.Read(f, binary.LittleEndian, &fb); err != nil {
        return fb, err
    }
    return fb, nil
}

func findInodeByPath(f *os.File, sb *structs.SuperBloque, path string) (int32, error) {
    parts := strings.Split(path, "/")
    curr := int32(0)
    empty := true
    for _, p := range parts[1:] {
        if strings.TrimSpace(p) != "" {
            empty = false
            break
        }
    }
    if empty {
        return curr, nil
    }
    lastIdx := len(parts[1:]) - 1
    for i, name := range parts[1:] {
        name = strings.TrimSpace(name)
        if name == "" {
            continue
        }
        ino, err := readInode(f, sb, curr)
        if err != nil {
            return -1, fmt.Errorf("leer inodo %d: %v", curr, err)
        }
        if i < lastIdx {
            if ino.I_type[0] != '0' {
                return -1, fmt.Errorf("no es un directorio")
            }
            if !hasPerm(&ino, permExec) {
                return -1, fmt.Errorf("permiso denegado al recorrer directorio")
            }
        }
        found := int32(-1)
        for _, b := range ino.I_block {
            if b < 0 {
                continue
            }
            dir, err := readDirBlock(f, sb, b)
            if err != nil {
                return -1, fmt.Errorf("leer bloque de carpeta %d: %v", b, err)
            }
            for _, e := range dir.B_content {
                if e.B_inodo < 0 {
                    continue
                }
                ename := strings.TrimRight(string(e.B_name[:]), "\x00")
                if ename == name {
                    found = e.B_inodo
                    break
                }
            }
            if found >= 0 {
                break
            }
        }
        if found < 0 {
            return -1, fmt.Errorf("entrada no encontrada: %s", name)
        }
        curr = found
    }
    return curr, nil
}

func readFileContent(f *os.File, sb *structs.SuperBloque, inoIdx int32) (string, error) {
    ino, err := readInode(f, sb, inoIdx)
    if err != nil {
        return "", err
    }
    if ino.I_type[0] != '1' {
        return "", fmt.Errorf("no es un archivo")
    }
    if !hasPerm(&ino, permRead) {
        return "", fmt.Errorf("permiso denegado para leer el archivo")
    }
    var buf []byte
    remaining := int(ino.I_s)
    for _, b := range ino.I_block {
        if b < 0 {
            continue
        }
        fb, err := readFileBlock(f, sb, b)
        if err != nil {
            return "", err
        }
        if remaining > 0 {
            toCopy := remaining
            if toCopy > len(fb.B_content) {
                toCopy = len(fb.B_content)
            }
            buf = append(buf, fb.B_content[:toCopy]...)
            remaining -= toCopy
            if remaining <= 0 {
                break
            }
        } else {
            for _, c := range fb.B_content {
                if c == 0 {
                    break
                }
                buf = append(buf, c)
            }
        }
    }
    if int(ino.I_s) > 0 && len(buf) > int(ino.I_s) {
        buf = buf[:ino.I_s]
    }
    return strings.TrimRight(string(buf), "\x00"), nil
}