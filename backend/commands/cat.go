package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "sort"
    "strconv"
    "strings"
)

func Cat(params map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    
    fileParams := make(map[int]string)
    hasFiles := false
    
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if lk == "-file" {
            fileParams[1] = strings.TrimSpace(v)
            hasFiles = true
        } else if strings.HasPrefix(lk, "-file") {
            numStr := strings.TrimPrefix(lk, "-file")
            if num, err := strconv.Atoi(numStr); err == nil && num > 0 {
                fileParams[num] = strings.TrimSpace(v)
                hasFiles = true
            }
        }
    }
    
    if !hasFiles {
        return "Error: parámetro -file es obligatorio."
    }

    var fileNumbers []int
    for num := range fileParams {
        fileNumbers = append(fileNumbers, num)
    }
    sort.Ints(fileNumbers)
    
    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()
    
    var results []string
    
    for _, num := range fileNumbers {
        path := unquoteValue(fileParams[num])
        if path == "" {
            continue
        }
        
        if !strings.HasPrefix(path, "/") {
            results = append(results, fmt.Sprintf("Error: ruta inválida '%s' (debe iniciar con /)", path))
            continue
        }
        
        if path == "/users.txt" {
            path = "/home/users.txt"
        }
        
        inoIdx, err := findInodeByPath(disk, sb, path)
        if err != nil {
            results = append(results, fmt.Sprintf("Error: %s -> %v", path, err))
            continue
        }
        
        content, err := readFileContentWithIndirect(disk, sb, inoIdx)
        if err != nil {
            results = append(results, fmt.Sprintf("Error: %s -> %v", path, err))
            continue
        }
        
        results = append(results, content)
    }
    
    return strings.Join(results, "\n")
}

func findInodeByPath(f *os.File, sb *structs.SuperBloque, path string) (int32, error) {
    parts := strings.Split(path, "/")
    curr := int32(0)
    
    if len(parts) <= 1 || (len(parts) == 2 && parts[1] == "") {
        return curr, nil
    }
    
    for i, name := range parts[1:] {
        name = strings.TrimSpace(name)
        if name == "" {
            continue
        }
        
        ino, err := readInode(f, sb, curr)
        if err != nil {
            return -1, fmt.Errorf("leer inodo %d: %v", curr, err)
        }
        
        if i < len(parts[1:])-1 {
            if ino.I_type[0] != 0 {
                return -1, fmt.Errorf("%s no es un directorio", name)
            }
            if !Permisos(&ino, permExec) {
                return -1, fmt.Errorf("permiso denegado al acceder directorio %s", name)
            }
        }
        
        found := int32(-1)
        for _, blockIdx := range ino.I_block {
            if blockIdx < 0 {
                continue
            }
            
            dir, err := readDirBlock(f, sb, blockIdx)
            if err != nil {
                continue
            }
            
            for _, entry := range dir.B_content {
                if entry.B_inodo < 0 {
                    continue
                }
                entryName := strings.TrimRight(string(entry.B_name[:]), "\x00")
                if entryName == name {
                    found = entry.B_inodo
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

func readFileContentWithIndirect(f *os.File, sb *structs.SuperBloque, inoIdx int32) (string, error) {
    ino, err := readInode(f, sb, inoIdx)
    if err != nil {
        return "", err
    }
    
    if ino.I_type[0] != 1 {
        return "", fmt.Errorf("no es un archivo")
    }
    
    if !Permisos(&ino, permRead) {
        return "", fmt.Errorf("permiso denegado para leer el archivo")
    }
    
    var contenido []byte
    remaining := int(ino.I_s)
    
    for i := 0; i < DIRECT_BLOCKS && remaining > 0; i++ {
        if ino.I_block[i] == -1 {
            break
        }
        
        bloque, err := readFileBlock(f, sb, ino.I_block[i])
        if err != nil {
            return "", err
        }
        
        chunk := 64
        if chunk > remaining {
            chunk = remaining
        }
        contenido = append(contenido, bloque.B_content[:chunk]...)
        remaining -= chunk
    }
    

    if remaining > 0 && ino.I_block[INDIRECT_SIMPLE] != -1 {
        pointers, err := readPointerBlock(f, sb, ino.I_block[INDIRECT_SIMPLE])
        if err != nil {
            return "", err
        }
        
        for i := 0; i < 16 && remaining > 0; i++ {
            if pointers.B_pointers[i] == -1 {
                break
            }
            
            bloque, err := readFileBlock(f, sb, pointers.B_pointers[i])
            if err != nil {
                return "", err
            }
            
            chunk := 64
            if chunk > remaining {
                chunk = remaining
            }
            contenido = append(contenido, bloque.B_content[:chunk]...)
            remaining -= chunk
        }
    }
    
    return strings.TrimRight(string(contenido), "\x00"), nil
}

func readDirBlock(f *os.File, sb *structs.SuperBloque, blockIdx int32) (structs.BCarpeta, error) {
    var bc structs.BCarpeta
    offset := int64(sb.S_block_start) + int64(blockIdx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return bc, err
    }
    err := binary.Read(f, binary.LittleEndian, &bc)
    return bc, err
}

func firstNonEmpty(a, b string) string {
    if strings.TrimSpace(a) != "" {
        return a
    }
    return b
}