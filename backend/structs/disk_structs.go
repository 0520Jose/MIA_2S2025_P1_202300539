package structs

import (
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "strings"
    "path/filepath"
)

const (
    DIRECT_BLOCKS     = 12
    INDIRECT_SIMPLE   = 12
    INDIRECT_DOUBLE   = 13
    INDIRECT_TRIPLE   = 14
    POINTERS_PER_BLOCK = 16
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

type InfoArchivo struct {
    Nombre       string
    Tipo         string
    Permisos     string
    Propietario  string
    Grupo        string
    Size         int32
    Creacion     string
    Modificacion string
}

func LeerMBR(diskPath string) (MBR, error) {
    var mbr MBR
    f, err := os.Open(diskPath)
    if err != nil {
        return mbr, err
    }
    defer f.Close()
    
    if _, err := f.Seek(0, io.SeekStart); err != nil {
        return mbr, err
    }
    
    err = binary.Read(f, binary.LittleEndian, &mbr)
    return mbr, err
}

func ObtenerBloqueBinario(f *os.File, sb *SuperBloque, idx int) ([]byte, bool) {
    if idx < 0 || idx >= int(sb.S_blocks_count) {
        return nil, false
    }

    bm := GetBitmapBlocks(f, sb)
    if bm == nil || idx >= len(bm) || bm[idx] == 0 {
        return nil, false
    }

    data := make([]byte, 64)
    offset := int64(sb.S_block_start) + int64(idx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return nil, false
    }
    if _, err := f.Read(data); err != nil {
        return nil, false
    }
    return data, true
}

func ReadBinaryStruct(f *os.File, data interface{}) error {
    return binary.Read(f, binary.LittleEndian, data)
}

func ObtenerBloqueArchivo(f *os.File, sb *SuperBloque, idx int) (BArchivo, bool) {
    var ba BArchivo
    if idx < 0 || idx >= int(sb.S_blocks_count) {
        return ba, false
    }

    bm := GetBitmapBlocks(f, sb)
    if bm == nil || idx >= len(bm) || bm[idx] == 0 {
        return ba, false
    }

    offset := int64(sb.S_block_start) + int64(idx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return ba, false
    }
    if err := binary.Read(f, binary.LittleEndian, &ba); err != nil {
        return ba, false
    }
    return ba, true
}

func ObtenerBloqueCarpeta(f *os.File, sb *SuperBloque, idx int) (BCarpeta, bool) {
    var bc BCarpeta
    if idx < 0 || idx >= int(sb.S_blocks_count) {
        return bc, false
    }

    bm := GetBitmapBlocks(f, sb)
    if bm == nil || idx >= len(bm) || bm[idx] == 0 {
        return bc, false
    }

    offset := int64(sb.S_block_start) + int64(idx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return bc, false
    }
    if err := binary.Read(f, binary.LittleEndian, &bc); err != nil {
        return bc, false
    }
    return bc, true
}

func ObtenerBloqueApuntadores(f *os.File, sb *SuperBloque, idx int) (BApuntadores, bool) {
    var bp BApuntadores
    if idx < 0 || idx >= int(sb.S_blocks_count) {
        return bp, false
    }

    bm := GetBitmapBlocks(f, sb)
    if bm == nil || idx >= len(bm) || bm[idx] == 0 {
        return bp, false
    }

    offset := int64(sb.S_block_start) + int64(idx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return bp, false
    }
    if err := binary.Read(f, binary.LittleEndian, &bp); err != nil {
        return bp, false
    }
    return bp, true
}

func LeerBloqueArchivo(f *os.File, sb *SuperBloque, idx int32) (BArchivo, bool) {
    var ba BArchivo
    if idx < 0 {
        return ba, false
    }
    
    block, ok := ObtenerBloqueArchivo(f, sb, int(idx))
    if !ok {
        return ba, false
    }
    return block, true
}

func LeerBloqueCarpeta(f *os.File, sb *SuperBloque, idx int32) (BCarpeta, bool) {
    var bc BCarpeta
    if idx < 0 {
        return bc, false
    }
    
    block, ok := ObtenerBloqueCarpeta(f, sb, int(idx))
    if !ok {
        return bc, false
    }
    return block, true
}

func ObtenerInodo(f *os.File, sb *SuperBloque, idx int) (Inodo, bool) {
    var ino Inodo
    if idx < 0 || idx >= int(sb.S_inodes_count) {
        return ino, false
    }

    bm := BitMapInodos(f, sb)
    if bm == nil || idx >= len(bm) || bm[idx] == 0 {
        return ino, false
    }

    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return ino, false
    }
    if err := binary.Read(f, binary.LittleEndian, &ino); err != nil {
        return ino, false
    }
    return ino, true
}

func SistemaArchivos_ID(id string) (*os.File, Partition, MBR, error) {
    var particion Partition
    var mbr MBR

    for _, montada := range Particiones_Montadas {
        if montada.Id == id {
            f, err := os.OpenFile(montada.Path, os.O_RDWR, 0755)
            if err != nil {
                return nil, particion, mbr, err
            }

            if _, err := f.Seek(0, io.SeekStart); err != nil {
                f.Close()
                return nil, particion, mbr, err
            }

            if err := binary.Read(f, binary.LittleEndian, &mbr); err != nil {
                f.Close()
                return nil, particion, mbr, err
            }

            return f, montada.Partition, mbr, nil
        }
    }

    return nil, particion, mbr, fmt.Errorf("partición con ID %s no encontrada", id)
}

func SuperBloque_ID(id string) (*os.File, *SuperBloque, Partition, error) {
    f, particion, _, err := SistemaArchivos_ID(id)
    if err != nil {
        return nil, nil, particion, err
    }

    sb := &SuperBloque{}
    if _, err := f.Seek(int64(particion.Part_start), io.SeekStart); err != nil {
        f.Close()
        return nil, nil, particion, err
    }

    if err := binary.Read(f, binary.LittleEndian, sb); err != nil {
        f.Close()
        return nil, nil, particion, err
    }

    return f, sb, particion, nil
}

func NombreDisco_ID(id string) string {
    for _, montada := range Particiones_Montadas {
        if montada.Id == id {
            return filepath.Base(montada.Path)
        }
    }
    return "disco_desconocido"
}

func BitMapInodos(f *os.File, sb *SuperBloque) []byte {
    bm := make([]byte, sb.S_inodes_count)
    if _, err := f.Seek(int64(sb.S_bm_inode_start), io.SeekStart); err != nil {
        return nil
    }
    if _, err := f.Read(bm); err != nil {
        return nil
    }
    return bm
}

func GetBitmapBlocks(f *os.File, sb *SuperBloque) []byte {
    bm := make([]byte, sb.S_blocks_count)
    if _, err := f.Seek(int64(sb.S_bm_block_start), io.SeekStart); err != nil {
        return nil
    }
    if _, err := f.Read(bm); err != nil {
        return nil
    }
    return bm
}

func GenerarReporteArbol(f *os.File, sb *SuperBloque, s int) string {
    var b strings.Builder
    b.WriteString("digraph G {\n")
    b.WriteString("  node [shape=plaintext, fontname=\"Helvetica\"];\n")
    b.WriteString("  rankdir=TB;\n")
    b.WriteString("  edge [color=\"#4a90e2\" penwidth=2 arrowhead=vee];\n")
    
    visited := map[int]bool{}
    visitedBlocks := map[int]bool{}
    
    var dfs func(idx int)
    
    dfs = func(idx int) {
        if visited[idx] {
            return
        }
        visited[idx] = true
        
        ino, ok := ObtenerInodo(f, sb, idx)
        if !ok {
            return
        }
        
        b.WriteString(fmt.Sprintf("  inode%d [label=<\n", idx))
        b.WriteString("    <table border='1' cellborder='1' cellspacing='0' cellpadding='6' bgcolor='#f9f9f9'>\n")
        b.WriteString(fmt.Sprintf("      <tr><td colspan='2' bgcolor='#4a90e2'><font color='white'><b>INODO %d</b></font></td></tr>\n", idx))
        b.WriteString(fmt.Sprintf("      <tr><td><b>i_type</b></td><td>%d</td></tr>\n", ino.I_type[0]))
        
        for i := 0; i < DIRECT_BLOCKS; i++ {
            b.WriteString(fmt.Sprintf("      <tr><td PORT='ap%d'><b>ap%d (directo)</b></td><td>%d</td></tr>\n",
                i, i, ino.I_block[i]))
        }
        
        b.WriteString(fmt.Sprintf("      <tr><td PORT='ap%d'><b>ap%d (indirecto)</b></td><td>%d</td></tr>\n", INDIRECT_SIMPLE, INDIRECT_SIMPLE, ino.I_block[INDIRECT_SIMPLE]))
        b.WriteString(fmt.Sprintf("      <tr><td PORT='ap%d'><b>ap%d (doble ind.)</b></td><td>%d</td></tr>\n", INDIRECT_DOUBLE, INDIRECT_DOUBLE, ino.I_block[INDIRECT_DOUBLE]))
        b.WriteString(fmt.Sprintf("      <tr><td PORT='ap%d'><b>ap%d (triple ind.)</b></td><td>%d</td></tr>\n", INDIRECT_TRIPLE, INDIRECT_TRIPLE, ino.I_block[INDIRECT_TRIPLE]))
        
        b.WriteString(fmt.Sprintf("      <tr><td><b>i_perm</b></td><td>%d</td></tr>\n", ino.I_perm))
        b.WriteString("    </table>\n")
        b.WriteString("  >];\n")
        
        if EsCarpeta(ino) {
            for i := 0; i < DIRECT_BLOCKS; i++ {
                blockIdx := ino.I_block[i]
                if blockIdx >= 0 && !visitedBlocks[int(blockIdx)] {
                    GenerarBloqueCarpeta(f, sb, &b, int(blockIdx), idx)
                    visitedBlocks[int(blockIdx)] = true
                    b.WriteString(fmt.Sprintf("  inode%d:ap%d -> block%d;\n", idx, i, blockIdx))
                    
                    bc, ok := LeerBloqueCarpeta(f, sb, blockIdx)
                    if ok {
                        for _, content := range bc.B_content {
                            name := trimBytes(content.B_name[:])
                            childIdx := content.B_inodo
                            if name != "" && name != "." && name != ".." && childIdx >= 0 {
                                b.WriteString(fmt.Sprintf("  block%d:child%d -> inode%d;\n", blockIdx, childIdx, childIdx))
                                dfs(int(childIdx))
                            }
                        }
                    }
                }
            }
        } else if EsArchivo(ino) {
            for i := 0; i < DIRECT_BLOCKS; i++ {
                blockIdx := ino.I_block[i]
                if blockIdx >= 0 && !visitedBlocks[int(blockIdx)] {
                    GenerarBloqueArchivo(f, sb, &b, int(blockIdx), idx, s)
                    visitedBlocks[int(blockIdx)] = true
                    b.WriteString(fmt.Sprintf("  inode%d:ap%d -> block%d;\n", idx, i, blockIdx))
                }
            }
            
            for i := INDIRECT_SIMPLE; i <= INDIRECT_TRIPLE && i < len(ino.I_block); i++ {
                blockIdx := ino.I_block[i]
                if blockIdx >= 0 && !visitedBlocks[int(blockIdx)] {
                    GenerarBloquePuntero(f, sb, &b, int(blockIdx), idx)
                    visitedBlocks[int(blockIdx)] = true
                    b.WriteString(fmt.Sprintf("  inode%d:ap%d -> block%d;\n", idx, i, blockIdx))
                    
                    procesarBloqueIndirecto(f, sb, &b, int(blockIdx), idx, &visitedBlocks, s)
                }
            }
        }
    }
    
    dfs(0)
    b.WriteString("}\n")
    return b.String()
}

func GenerarBloquePuntero(f *os.File, sb *SuperBloque, b *strings.Builder, blockIdx, inodeIdx int) {
    pointers, ok := ObtenerBloqueApuntadores(f, sb, blockIdx)
    if !ok {
        return
    }

    b.WriteString(fmt.Sprintf("  block%d [label=<\n", blockIdx))
    b.WriteString("    <table border='1' cellborder='1' cellspacing='0' cellpadding='6' bgcolor='#eafaf1'>\n")
    b.WriteString(fmt.Sprintf("      <tr><td colspan='2' bgcolor='#27ae60'><font color='white'><b>Bloque apuntadores %d</b></font></td></tr>\n", blockIdx))

    for i := 0; i < 16; i++ {
        ptr := pointers.B_pointers[i]
        b.WriteString(fmt.Sprintf("      <tr><td PORT='ap%d'><b>ap_%d</b></td><td>%d</td></tr>\n", i, i, ptr))
    }

    b.WriteString("    </table>\n")
    b.WriteString("  >];\n")
}

func GenerarBloqueCarpeta(f *os.File, sb *SuperBloque, b *strings.Builder, blockIdx, inodeIdx int) {
    bc, ok := LeerBloqueCarpeta(f, sb, int32(blockIdx))
    if !ok {
        return
    }

    b.WriteString(fmt.Sprintf("  block%d [label=<\n", blockIdx))
    b.WriteString("    <table border='1' cellborder='1' cellspacing='0' cellpadding='6' bgcolor='#fffbea'>\n")
    b.WriteString(fmt.Sprintf("      <tr><td colspan='2' bgcolor='#f39c12'><font color='white'><b>Bloque carpeta %d</b></font></td></tr>\n", blockIdx))

    for _, content := range bc.B_content {
        name := trimBytes(content.B_name[:])
        if name != "" {
            b.WriteString(fmt.Sprintf("      <tr><td PORT='child%d'><b>%s</b></td><td>%d</td></tr>\n", content.B_inodo, name, content.B_inodo))
        }
    }

    b.WriteString("    </table>\n")
    b.WriteString("  >];\n")
}

func GenerarBloqueArchivo(f *os.File, sb *SuperBloque, b *strings.Builder, blockIdx, inodeIdx, s int) {
    ba, ok := LeerBloqueArchivo(f, sb, int32(blockIdx))
    if !ok {
        return
    }

    content := trimBytes(ba.B_content[:])
    if len(content) > s {
        content = content[:s] + "..."
    }

    content = strings.ReplaceAll(content, "&", "&amp;")
    content = strings.ReplaceAll(content, "<", "&lt;")
    content = strings.ReplaceAll(content, ">", "&gt;")
    content = strings.ReplaceAll(content, "\"", "&quot;")

    b.WriteString(fmt.Sprintf("  block%d [label=<\n", blockIdx))
    b.WriteString("    <table border='1' cellborder='1' cellspacing='0' cellpadding='6' bgcolor='#fdecea'>\n")
    b.WriteString(fmt.Sprintf("      <tr><td bgcolor='#c0392b'><font color='white'><b>Bloque archivo %d</b></font></td></tr>\n", blockIdx))
    b.WriteString(fmt.Sprintf("      <tr><td PORT='content' align='left'><font face='monospace'>%s</font></td></tr>\n", content))
    b.WriteString("    </table>\n")
    b.WriteString("  >];\n")
}

func procesarBloqueIndirecto(f *os.File, sb *SuperBloque, b *strings.Builder, blockIdx, inodeIdx int, visitedBlocks *map[int]bool, s int) {
    pointers, ok := ObtenerBloqueApuntadores(f, sb, blockIdx)
    if !ok {
        return
    }

    for _, ptr := range pointers.B_pointers {
        if ptr >= 0 && !(*visitedBlocks)[int(ptr)] {
            GenerarBloqueArchivo(f, sb, b, int(ptr), inodeIdx, s)
            (*visitedBlocks)[int(ptr)] = true
            b.WriteString(fmt.Sprintf("  block%d -> block%d;\n", blockIdx, ptr))
        }
    }
}

func EsCarpeta(ino Inodo) bool {
    return ino.I_type[0] == 0
}

func EsArchivo(ino Inodo) bool {
    return ino.I_type[0] == 1
}

func trimBytes(b []byte) string {
    end := len(b)
    for i, v := range b {
        if v == 0 {
            end = i
            break
        }
    }
    return string(b[:end])
}

func LeerArchivoDeFS(id, rutaArchivo string) (string, error) {
    f, sb, _, err := SuperBloque_ID(id)
    if err != nil {
        return "", err
    }
    defer f.Close()

    ino, err := buscarInodoPorRuta(f, sb, rutaArchivo)
    if err != nil {
        return "", err
    }

    if EsCarpeta(ino) {
        return "", fmt.Errorf("la ruta especificada es una carpeta, no un archivo")
    }

    var contenido strings.Builder
    
    for i := 0; i < DIRECT_BLOCKS; i++ {
        blockIdx := ino.I_block[i]
        if blockIdx < 0 {
            break
        }
        
        ba, ok := LeerBloqueArchivo(f, sb, blockIdx)
        if ok {
            contenido.Write(ba.B_content[:])
        }
    }

    return strings.TrimRight(contenido.String(), "\x00"), nil
}

func buscarInodoPorRuta(f *os.File, sb *SuperBloque, ruta string) (Inodo, error) {
    var ino Inodo
    
    if ruta == "/" {
        ino, ok := ObtenerInodo(f, sb, 0)
        if !ok {
            return ino, fmt.Errorf("no se pudo obtener el inodo raíz")
        }
        return ino, nil
    }

    parts := strings.Split(strings.Trim(ruta, "/"), "/")
    currentIno, ok := ObtenerInodo(f, sb, 0)
    if !ok {
        return ino, fmt.Errorf("no se pudo obtener el inodo raíz")
    }

    for _, part := range parts {
        if part == "" {
            continue
        }

        foundIno := int32(-1)
        
        for i := 0; i < DIRECT_BLOCKS; i++ {
            blockIdx := currentIno.I_block[i]
            if blockIdx < 0 {
                break
            }
            
            bc, ok := LeerBloqueCarpeta(f, sb, blockIdx)
            if ok {
                for _, content := range bc.B_content {
                    name := trimBytes(content.B_name[:])
                    if name == part {
                        foundIno = content.B_inodo
                        break
                    }
                }
            }
            
            if foundIno >= 0 {
                break
            }
        }

        if foundIno < 0 {
            return ino, fmt.Errorf("archivo o directorio %s no encontrado", part)
        }

        var ok bool
        currentIno, ok = ObtenerInodo(f, sb, int(foundIno))
        if !ok {
            return ino, fmt.Errorf("no se pudo obtener el inodo %d", foundIno)
        }
    }

    return currentIno, nil
}

func EscribirBloqueArchivo(f *os.File, sb *SuperBloque, blockIdx int32, ba *BArchivo) error {
    offset := int64(sb.S_block_start) + int64(blockIdx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, ba)
}

func EscribirBloqueApuntadores(f *os.File, sb *SuperBloque, blockIdx int32, bp *BApuntadores) error {
    offset := int64(sb.S_block_start) + int64(blockIdx)*64
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return err
    }
    return binary.Write(f, binary.LittleEndian, bp)
}


func ListaCarpetasFS(id, rutaCarpeta string) ([]InfoArchivo, error) {
    f, sb, _, err := SuperBloque_ID(id)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    ino, err := buscarInodoPorRuta(f, sb, rutaCarpeta)
    if err != nil {
        return nil, err
    }

    if !EsCarpeta(ino) {
        return nil, fmt.Errorf("la ruta especificada no es una carpeta")
    }

    var archivos []InfoArchivo

    for i := 0; i < DIRECT_BLOCKS; i++ {
        blockIdx := ino.I_block[i]
        if blockIdx < 0 {
            break
        }
        
        bc, ok := LeerBloqueCarpeta(f, sb, blockIdx)
        if ok {
            for _, content := range bc.B_content {
                name := trimBytes(content.B_name[:])
                if name != "" && name != "." && name != ".." {
                    childIno, ok := ObtenerInodo(f, sb, int(content.B_inodo))
                    if ok {
                        info := InfoArchivo{
                            Nombre:       name,
                            Propietario:  fmt.Sprintf("%d", childIno.I_uid),
                            Grupo:        fmt.Sprintf("%d", childIno.I_gid),
                            Size:         childIno.I_s,
                            Permisos:     fmt.Sprintf("%o", childIno.I_perm),
                            Creacion:     trimBytes(childIno.I_ctime[:]),
                            Modificacion: trimBytes(childIno.I_mtime[:]),
                        }
                        
                        if EsCarpeta(childIno) {
                            info.Tipo = "d"
                        } else {
                            info.Tipo = "f"
                        }
                        
                        archivos = append(archivos, info)
                    }
                }
            }
        }
    }

    return archivos, nil
}