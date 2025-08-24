package structs

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "strings"
)

var Particiones_Montadas []PartitionMount

type PartitionMount struct {
    Id        string
    Path      string
    Partition Partition
    UserFile  string
}

type MBR struct {
    Mbr_tamano         int32
    Mbr_fecha_creacion [16]byte
    Mbr_dsk_signature  int32
    Dsk_fit            byte
    Mbr_partitions     [4]Partition
}

type Partition struct {
    Part_status      byte
    Part_type        byte
    Part_fit         byte
    Part_start       int32
    Part_s           int32
    Part_name        [16]byte
    Part_correlative int32
    Part_id          [4]byte
}

type EBR struct {
    Part_mount byte
    Part_fit   byte
    Part_start int32
    Part_s     int32
    Part_next  int32
    Part_name  [16]byte
}

type SuperBloque struct {
    S_filesystem_type   int32
    S_inodes_count      int32
    S_blocks_count      int32
    S_free_blocks_count int32
    S_free_inodes_count int32
    S_mtime             [17]byte
    S_umtime            [17]byte
    S_mnt_count         int32
    S_magic             int32
    S_inode_s           int32
    S_block_s           int32
    S_first_ino         int32
    S_first_blo         int32
    S_bm_inode_start    int32
    S_bm_block_start    int32
    S_inode_start       int32
    S_block_start       int32
}

type Inodo struct {
    I_uid   int32
    I_gid   int32
    I_s     int32
    I_atime [17]byte
    I_ctime [17]byte
    I_mtime [17]byte
    I_block [15]int32
    I_type  [1]byte
    I_perm  [3]byte
}

type BContent struct {
    B_name  [12]byte
    B_inodo int32
}

type BCarpeta struct {
    B_content [4]BContent
}

type BArchivo struct {
    B_content [64]byte
}

type BApuntadores struct {
    B_pointers [16]int32
}

type Bitmap []byte

type Bloque struct {
    Data [64]byte
}

func (m *MBR) WriteToFile(file *os.File) error {
    file.Seek(0, 0)
    return binary.Write(file, binary.LittleEndian, m)
}

func LeerMBR(archivo *os.File) (MBR, error) {
    archivo.Seek(0, 0)
    var mbr MBR
    err := binary.Read(archivo, binary.LittleEndian, &mbr)
    return mbr, err
}

func GetMountedPartitionByID(id string, mbr MBR) Partition {
    for _, pm := range Particiones_Montadas {
        if pm.Id == id {
            return pm.Partition
        }
    }
    return Partition{}
}

func GetDiskPathByID(id string) string {
    for _, pm := range Particiones_Montadas {
        if pm.Id == id {
            return pm.Path
        }
    }
    return ""
}

func GetFileSystemByID(id string) (*os.File, *SuperBloque, *MBR, error) {
    for _, pm := range Particiones_Montadas {
        if pm.Id == id {
            f, err := os.OpenFile(pm.Path, os.O_RDONLY, 0)
            if err != nil {
                return nil, nil, nil, fmt.Errorf("no se pudo abrir el disco: %v", err)
            }

            var mbr MBR
            if _, err := f.Seek(0, io.SeekStart); err != nil {
                f.Close()
                return nil, nil, nil, fmt.Errorf("no se pudo buscar MBR: %v", err)
            }
            if err := binary.Read(f, binary.LittleEndian, &mbr); err != nil {
                f.Close()
                return nil, nil, nil, fmt.Errorf("no se pudo leer MBR: %v", err)
            }

            partStart := pm.Partition.Part_start
            if partStart <= 0 {
                f.Close()
                return nil, nil, nil, fmt.Errorf("part_start inválido: %d", partStart)
            }

            var sb SuperBloque
            if _, err := f.Seek(int64(partStart), io.SeekStart); err != nil {
                f.Close()
                return nil, nil, nil, fmt.Errorf("no se pudo buscar superbloque: %v", err)
            }
            if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
                f.Close()
                return nil, nil, nil, fmt.Errorf("no se pudo leer superbloque: %v", err)
            }
            if sb.S_magic != 0xEF53 {
                f.Close()
                return nil, nil, nil, fmt.Errorf("superbloque inválido")
            }

            return f, &sb, &mbr, nil
        }
    }
    return nil, nil, nil, fmt.Errorf("partición no montada: %s", id)
}

func GetSuperBlockByID(id string) (*os.File, *SuperBloque, *MBR, error) {
    return GetFileSystemByID(id)
}

func GetBitmapInodes(f *os.File, sb *SuperBloque) []byte {
    n := int(sb.S_inodes_count)
    if n <= 0 {
        return nil
    }
    bitmap := make([]byte, n)
    if _, err := f.Seek(int64(sb.S_bm_inode_start), io.SeekStart); err != nil {
        return nil
    }
    if _, err := io.ReadFull(f, bitmap); err != nil {
        return nil
    }
    return bitmap
}

func GetBitmapBlocks(f *os.File, sb *SuperBloque) []byte {
    n := int(sb.S_blocks_count)
    if n <= 0 {
        return nil
    }
    bitmap := make([]byte, n)
    if _, err := f.Seek(int64(sb.S_bm_block_start), io.SeekStart); err != nil {
        return nil
    }
    if _, err := io.ReadFull(f, bitmap); err != nil {
        return nil
    }
    return bitmap
}

func GetInode(f *os.File, sb *SuperBloque, idx int) (Inodo, bool) {
    var inode Inodo

    if idx < 0 || idx >= int(sb.S_inodes_count) {
        return inode, false
    }

    bm := GetBitmapInodes(f, sb)
    if bm == nil || idx >= len(bm) || bm[idx] == 0 {
        return inode, false
    }

    offset := int64(sb.S_inode_start) + int64(idx)*int64(binary.Size(inode))
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return inode, false
    }
    if err := binary.Read(f, binary.LittleEndian, &inode); err != nil {
        return inode, false
    }
    return inode, true
}

func GetBlock(f *os.File, sb *SuperBloque, idx int) (Bloque, bool) {
    var block Bloque
    if idx < 0 || idx >= int(sb.S_blocks_count) {
        return block, false
    }

    bm := GetBitmapBlocks(f, sb)
    if bm == nil || idx >= len(bm) || bm[idx] == 0 {
        return block, false
    }

    offset := int64(sb.S_block_start) + int64(idx)*int64(binary.Size(block))
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return block, false
    }
    if err := binary.Read(f, binary.LittleEndian, &block); err != nil {
        return block, false
    }
    return block, true
}

func (b Bloque) ContentString() string {
    return string(b.Data[:])
}

func trimBytes(b []byte) string {
    if i := bytes.IndexByte(b, 0); i >= 0 {
        b = b[:i]
    }
    return strings.TrimSpace(string(b))
}

func isDir(in Inodo) bool  { return len(in.I_type) > 0 && in.I_type[0] == 0 }
func isFile(in Inodo) bool { return len(in.I_type) > 0 && in.I_type[0] == 1 }

func readBlockAsCarpeta(f *os.File, sb *SuperBloque, idx int32) (BCarpeta, bool) {
    var bc BCarpeta
    if idx < 0 {
        return bc, false
    }
    block, ok := GetBlock(f, sb, int(idx))
    if !ok {
        return bc, false
    }
    rdr := bytes.NewReader(block.Data[:])
    if err := binary.Read(rdr, binary.LittleEndian, &bc); err != nil {
        return bc, false
    }
    return bc, true
}

func readBlockAsArchivo(f *os.File, sb *SuperBloque, idx int32) (BArchivo, bool) {
    var ba BArchivo
    if idx < 0 {
        return ba, false
    }
    block, ok := GetBlock(f, sb, int(idx))
    if !ok {
        return ba, false
    }
    rdr := bytes.NewReader(block.Data[:])
    if err := binary.Read(rdr, binary.LittleEndian, &ba); err != nil {
        return ba, false
    }
    return ba, true
}

func listDir(f *os.File, sb *SuperBloque, dir Inodo) map[string]int32 {
    entries := make(map[string]int32)
    bmIn := GetBitmapInodes(f, sb)

    for i := 0; i < 12; i++ {
        blk := dir.I_block[i]
        if blk < 0 {
            continue
        }
        bc, ok := readBlockAsCarpeta(f, sb, blk)
        if !ok {
            continue
        }
        for _, c := range bc.B_content {
            name := trimBytes(c.B_name[:])
            ino := c.B_inodo
            if name == "" || name == "." || name == ".." || ino < 0 {
                continue
            }
            if bmIn != nil && int(ino) < len(bmIn) && bmIn[int(ino)] != 0 {
                entries[name] = ino
            }
        }
    }
    return entries
}

func resolvePath(f *os.File, sb *SuperBloque, path string) (int, Inodo, bool) {
    comps := []string{}
    for _, p := range strings.Split(path, "/") {
        p = strings.TrimSpace(p)
        if p != "" {
            comps = append(comps, p)
        }
    }

    currIdx := 0
    curr, ok := GetInode(f, sb, currIdx)
    if !ok {
        return -1, Inodo{}, false
    }

    if len(comps) == 0 {
        return currIdx, curr, true
    }

    for _, name := range comps {
        if !isDir(curr) {
            return -1, Inodo{}, false
        }
        ents := listDir(f, sb, curr)
        nextIdx32, exists := ents[name]
        if !exists {
            return -1, Inodo{}, false
        }
        next, ok := GetInode(f, sb, int(nextIdx32))
        if !ok {
            return -1, Inodo{}, false
        }
        currIdx = int(nextIdx32)
        curr = next
    }
    return currIdx, curr, true
}

func GenerateTreeGraph(f *os.File, sb *SuperBloque, s int) string {
    var b strings.Builder
    b.WriteString("digraph G {\n")
    b.WriteString("  node [shape=plaintext, fontname=\"Arial\"];\n")
    b.WriteString("  rankdir=TB;\n")
    
    visited := map[int]bool{}
    visitedBlocks := map[int]bool{}
    
    var dfs func(idx int)
    
    dfs = func(idx int) {
        if visited[idx] {
            return
        }
        visited[idx] = true
        
        ino, ok := GetInode(f, sb, idx)
        if !ok {
            return
        }
        
        // Tabla inodo con encabezado mostrando número y tipo
        b.WriteString(fmt.Sprintf("  inode%d [label=<\n", idx))
        b.WriteString("    <table border='1' cellborder='1' cellspacing='0' bgcolor='lightblue'>\n")
        b.WriteString(fmt.Sprintf("      <tr><td colspan='2'><b>INODO %d</b></td></tr>\n", idx))
        b.WriteString(fmt.Sprintf("      <tr><td>i_type</td><td>%d</td></tr>\n", ino.I_type[0]))
        if 0 < 15 {
            b.WriteString(fmt.Sprintf("      <tr><td>ap0 (directo)</td><td>%d</td></tr>\n", ino.I_block[0]))
        }
        if 12 < 15 {
            b.WriteString(fmt.Sprintf("      <tr><td>ap1 (indirecto)</td><td>%d</td></tr>\n", ino.I_block[12]))
        }
        if 13 < 15 {
            b.WriteString(fmt.Sprintf("      <tr><td>ap2 (doble indirecto)</td><td>%d</td></tr>\n", ino.I_block[13]))
        }
        b.WriteString(fmt.Sprintf("      <tr><td>i_perm</td><td>%d</td></tr>\n", ino.I_perm))
        b.WriteString("    </table>\n")
        b.WriteString("  >];\n")
        
        if isDir(ino) {
            for i := 0; i < 12; i++ {
                blockIdx := ino.I_block[i]
                if blockIdx >= 0 && !visitedBlocks[int(blockIdx)] {
                    generateDirectoryBlock(f, sb, &b, int(blockIdx), idx)
                    visitedBlocks[int(blockIdx)] = true
                    b.WriteString(fmt.Sprintf("  inode%d -> block%d;\n", idx, blockIdx))
                    
                    bc, ok := readBlockAsCarpeta(f, sb, blockIdx)
                    if ok {
                        for _, content := range bc.B_content {
                            name := trimBytes(content.B_name[:])
                            childIdx := content.B_inodo
                            if name != "" && name != "." && name != ".." && childIdx >= 0 {
                                b.WriteString(fmt.Sprintf("  block%d -> inode%d;\n", blockIdx, childIdx))
                                dfs(int(childIdx))
                            }
                        }
                    }
                }
            }
        } else if isFile(ino) {
            for i := 0; i < 12; i++ {
                blockIdx := ino.I_block[i]
                if blockIdx >= 0 && !visitedBlocks[int(blockIdx)] {
                    generateFileBlock(f, sb, &b, int(blockIdx), idx, s)
                    visitedBlocks[int(blockIdx)] = true
                    b.WriteString(fmt.Sprintf("  inode%d -> block%d;\n", idx, blockIdx))
                }
            }
            // Bloques de apuntadores: mostrar solo 2
            for i := 12; i < 15; i++ {
                blockIdx := ino.I_block[i]
                if blockIdx >= 0 && !visitedBlocks[int(blockIdx)] {
                    generatePointerBlock(f, sb, &b, int(blockIdx), idx)
                    visitedBlocks[int(blockIdx)] = true
                    b.WriteString(fmt.Sprintf("  inode%d -> block%d;\n", idx, blockIdx))
                }
            }
        }
    }
    
    dfs(0)
    b.WriteString("}\n")
    return b.String()
}

func generateFileBlock(f *os.File, sb *SuperBloque, b *strings.Builder, blockIdx, inodeIdx int, s int) {
    ba, ok := readBlockAsArchivo(f, sb, int32(blockIdx))
    if !ok {
        return
    }
    content := trimBytes(ba.B_content[:])
    if len(content) > 3 {
        content = content[:3]
    }

    b.WriteString(fmt.Sprintf("  block%d [label=<\n", blockIdx))
    b.WriteString("    <table border='1' cellborder='1' cellspacing='0' bgcolor='yellow'>\n")
    b.WriteString(fmt.Sprintf("      <tr><td><b>b. archivo %d</b></td></tr>\n", blockIdx))
    b.WriteString(fmt.Sprintf("      <tr><td>%s</td></tr>\n", content))
    b.WriteString("    </table>\n")
    b.WriteString("  >];\n")
}

func generatePointerBlock(f *os.File, sb *SuperBloque, b *strings.Builder, blockIdx, inodeIdx int) {
    block, ok := GetBlock(f, sb, blockIdx)
    if !ok {
        return
    }

    var pointers BApuntadores
    rdr := bytes.NewReader(block.Data[:])
    if err := binary.Read(rdr, binary.LittleEndian, &pointers); err != nil {
        return
    }

    b.WriteString(fmt.Sprintf("  block%d [label=<\n", blockIdx))
    b.WriteString("    <table border='1' cellborder='1' cellspacing='0' bgcolor='lightgreen'>\n")
    b.WriteString(fmt.Sprintf("      <tr><td colspan='2'><b>b. apuntadores %d</b></td></tr>\n", blockIdx))

    for i := 0; i < 2 && i < len(pointers.B_pointers); i++ {
        ptr := pointers.B_pointers[i]
        b.WriteString(fmt.Sprintf("      <tr><td>ap_%d</td><td>%d</td></tr>\n", i, ptr))
    }

    b.WriteString("    </table>\n")
    b.WriteString("  >];\n")
}

func generateDirectoryBlock(f *os.File, sb *SuperBloque, b *strings.Builder, blockIdx, inodeIdx int) {
    bc, ok := readBlockAsCarpeta(f, sb, int32(blockIdx))
    if !ok {
        return
    }
    
    b.WriteString(fmt.Sprintf("  block%d [label=<\n", blockIdx))
    b.WriteString("    <table border='1' cellborder='1' cellspacing='0' bgcolor='salmon'>\n")
    b.WriteString(fmt.Sprintf("      <tr><td colspan='2'><b>b. carpeta %d</b></td></tr>\n", blockIdx))
    b.WriteString("      <tr><td><b>b_name</b></td><td><b>b_inodo</b></td></tr>\n")
    
    for _, content := range bc.B_content {
        name := trimBytes(content.B_name[:])
        if name != "" {
            b.WriteString(fmt.Sprintf("      <tr><td>%s</td><td>%d</td></tr>\n", name, content.B_inodo))
        } else {
            b.WriteString("      <tr><td></td><td></td></tr>\n")
        }
    }
    
    b.WriteString("    </table>\n")
    b.WriteString("  >];\n")
}


func ReadFileFromFS(id, path string) (string, error) {
    f, sb, _, err := GetFileSystemByID(id)
    if err != nil {
        return "", err
    }
    defer f.Close()

    _, ino, ok := resolvePath(f, sb, path)
    if !ok {
        return "", fmt.Errorf("ruta no encontrada: %s", path)
    }
    if !isFile(ino) {
        return "", fmt.Errorf("no es un archivo: %s", path)
    }

    var data []byte
    remaining := int(ino.I_s)
    for i := 0; i < 12 && remaining > 0; i++ {
        blk := ino.I_block[i]
        if blk < 0 {
            continue
        }
        ba, ok := readBlockAsArchivo(f, sb, blk)
        if !ok {
            break
        }
        chunk := 64
        if remaining < chunk {
            chunk = remaining
        }
        data = append(data, ba.B_content[:chunk]...)
        remaining -= chunk
    }
    return string(data), nil
}

type DirEntry struct {
    Nombre       string
    Tipo         string
    Permisos     string
    Propietario  string
    Grupo        string
    Creacion     string
    Modificacion string
}

func ListDirectoryFS(id, path string) ([]DirEntry, error) {
    f, sb, _, err := GetFileSystemByID(id)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    _, ino, ok := resolvePath(f, sb, path)
    if !ok {
        return nil, fmt.Errorf("ruta no encontrada: %s", path)
    }
    if !isDir(ino) {
        return nil, fmt.Errorf("no es un directorio: %s", path)
    }

    ents := listDir(f, sb, ino)
    out := make([]DirEntry, 0, len(ents))
    for name, idx := range ents {
        child, ok := GetInode(f, sb, int(idx))
        if !ok {
            continue
        }
        tipo := "Carpeta"
        if isFile(child) {
            tipo = "Archivo"
        }
        permisos := string([]byte{child.I_perm[0], child.I_perm[1], child.I_perm[2]})
        de := DirEntry{
            Nombre:       name,
            Tipo:         tipo,
            Permisos:     permisos,
            Propietario:  fmt.Sprintf("%d", child.I_uid),
            Grupo:        fmt.Sprintf("%d", child.I_gid),
            Creacion:     trimBytes(child.I_ctime[:]),
            Modificacion: trimBytes(child.I_mtime[:]),
        }
        out = append(out, de)
    }
    return out, nil
}
