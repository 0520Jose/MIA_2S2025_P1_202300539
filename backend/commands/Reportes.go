package commands

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "backend/structs"
    "io"
    "encoding/binary"
    "sort"
	"time"
)

func Rep(params map[string]string) string {
    name, existeName := params["-name"]
    path, existePath := params["-path"]
    id, existeID := params["-id"]

    if !existeName || !existePath || !existeID {
        return "Error: parámetros -name, -path y -id son obligatorios"
    }

    name = strings.ToLower(name)

    dir := filepath.Dir(path)
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        os.MkdirAll(dir, 0755)
    }

    switch name {
	case "mbr":
		return generarReporteMBR(path, id)
	case "disk":
		return generarReporteDISK(path, id)
	case "inode":
		return generarReporteInode(path, id)
	case "block":
		return generarReporteBlock(path, id)
	case "bm_inode":
		return generarReporteBMInode(path, id)
	case "bm_block":
		return generarReporteBMBlock(path, id)
	case "tree":
		return generarReporteTree(path, id)
	case "sb":
		return generarReporteSB(path, id)
	case "file":
		return generarReporteFile(path, id, params["-path_file_ls"])
	case "ls":
		if err := generarReporteLS(id, params["-path_file_ls"], path); err != nil {
            return fmt.Sprintf("Error: %v", err)
        }
		return fmt.Sprintf("Reporte ls generado en %s", path)
	default:
		return fmt.Sprintf("Error: reporte %s no válido", name)
	}
}

func generarReporteMBR(path, id string) string {
    disk, _, mbr, err := structs.SistemaArchivos_ID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    dot := "digraph G {\n"
    dot += "node [shape=plaintext, style=filled, fillcolor=\"#f9f9f9\"]\n"
    dot += "ReporteMBR [label=<\n"
    dot += "<table border='1' cellborder='1' cellspacing='0' bgcolor='#e3f2fd'>\n"
    dot += "<tr><td colspan='2' bgcolor='#1976d2'><font color='white'><b>REPORTE DE MBR</b></font></td></tr>\n"
    dot += fmt.Sprintf("<tr><td bgcolor='#bbdefb'>mbr_tamano</td><td>%d</td></tr>\n", mbr.Mbr_tamano)
    dot += fmt.Sprintf("<tr><td bgcolor='#bbdefb'>mbr_fecha_creacion</td><td>%s</td></tr>\n", strings.Trim(string(mbr.Mbr_fecha_creacion[:]), "\x00"))
    dot += fmt.Sprintf("<tr><td bgcolor='#bbdefb'>mbr_disk_signature</td><td>%d</td></tr>\n", mbr.Mbr_dsk_signature)
    dot += fmt.Sprintf("<tr><td bgcolor='#bbdefb'>dsk_fit</td><td>%c</td></tr>\n", mbr.Dsk_fit)

    for i, part := range mbr.Mbr_partitions {
        if part.Part_s > 0 {
            dot += fmt.Sprintf("<tr><td colspan='2' bgcolor='#64b5f6'><b>Partición %d</b></td></tr>\n", i+1)
            dot += fmt.Sprintf("<tr><td bgcolor='#90caf9'>part_status</td><td>%d</td></tr>\n", part.Part_status)
            
            tipoParticion := "Primaria"
            colorTipo := "#43a047"
            if part.Part_type == 'e' || part.Part_type == 'E' {
                tipoParticion = "Extendida"
                colorTipo = "#fbc02d"
            }
            dot += fmt.Sprintf("<tr><td bgcolor='#90caf9'>part_type</td><td bgcolor='%s'>%s (%c)</td></tr>\n", colorTipo, tipoParticion, part.Part_type)
            dot += fmt.Sprintf("<tr><td bgcolor='#90caf9'>part_fit</td><td>%c</td></tr>\n", part.Part_fit)
            dot += fmt.Sprintf("<tr><td bgcolor='#90caf9'>part_start</td><td>%d</td></tr>\n", part.Part_start)
            dot += fmt.Sprintf("<tr><td bgcolor='#90caf9'>part_size</td><td>%d</td></tr>\n", part.Part_s)
            dot += fmt.Sprintf("<tr><td bgcolor='#90caf9'>part_name</td><td>%s</td></tr>\n", strings.Trim(string(part.Part_name[:]), "\x00"))
            
            if part.Part_type == 'e' || part.Part_type == 'E' {
                dot += leerParticionesLogicas(disk, part.Part_start)
            }
        }
    }

    dot += "</table>\n>];\n}\n"

    dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
    err = os.WriteFile(dotFile, []byte(dot), 0644)
    if err != nil {
        return fmt.Sprintf("Error escribiendo DOT: %v", err)
    }

    format := strings.TrimPrefix(filepath.Ext(path), ".")
    cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
    err = cmd.Run()
    if err != nil {
        return fmt.Sprintf("Error ejecutando Graphviz: %v", err)
    }

    return fmt.Sprintf("Reporte MBR generado en %s", path)
}

func leerParticionesLogicas(disk *os.File, startExtendida int32) string {
    var dot string
    ebrPos := int64(startExtendida)
    logicalNum := 1

    for ebrPos > 0 {
        var ebr structs.EBR
        if _, err := disk.Seek(ebrPos, io.SeekStart); err != nil {
            break
        }
        if err := binary.Read(disk, binary.LittleEndian, &ebr); err != nil {
            break
        }

        if ebr.Part_mount == 1 && ebr.Part_s > 0 {
            dot += fmt.Sprintf("<tr><td colspan='2' bgcolor='#ffd54f'><b>Partición Lógica %d</b></td></tr>\n", logicalNum)
            dot += fmt.Sprintf("<tr><td bgcolor='#ffe082'>part_status</td><td>%d</td></tr>\n", ebr.Part_mount)
            dot += fmt.Sprintf("<tr><td bgcolor='#ffe082'>part_next</td><td>%d</td></tr>\n", ebr.Part_next)
            dot += fmt.Sprintf("<tr><td bgcolor='#ffe082'>part_fit</td><td>%c</td></tr>\n", ebr.Part_fit)
            dot += fmt.Sprintf("<tr><td bgcolor='#ffe082'>part_start</td><td>%d</td></tr>\n", ebr.Part_start)
            dot += fmt.Sprintf("<tr><td bgcolor='#ffe082'>part_size</td><td>%d</td></tr>\n", ebr.Part_s)
            dot += fmt.Sprintf("<tr><td bgcolor='#ffe082'>part_name</td><td>%s</td></tr>\n", strings.Trim(string(ebr.Part_name[:]), "\x00"))
            logicalNum++
        }

        if ebr.Part_next == -1 || ebr.Part_next == 0 {
            break
        }
        ebrPos = int64(ebr.Part_next)
    }

    return dot
}

func generarArchivoDOT(dot, path string) string {
    dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
    err := os.WriteFile(dotFile, []byte(dot), 0644)
    if err != nil {
        return fmt.Sprintf("Error escribiendo DOT: %v", err)
    }

    format := strings.TrimPrefix(filepath.Ext(path), ".")
    cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Sprintf("Error ejecutando Graphviz: %v\nOutput: %s\nDOT file: %s", err, string(output), dotFile)
    }

    return fmt.Sprintf("Reporte generado en %s", path)
}

func generarReporteDISK(path, id string) string {
	disk, _, mbr, err := structs.SistemaArchivos_ID(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer disk.Close()

	nombreDisco := structs.NombreDisco_ID(id)
	
	dot := "digraph G {\n"
	dot += "node [shape=plaintext]\n"
	dot += "ReporteDISK [label=<\n"
	dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
	
	dot += fmt.Sprintf("<tr><td colspan='20' bgcolor='#E8F4FD'><b>%s</b></td></tr>\n", nombreDisco)
	
	dot += "<tr>"
	
	dot += "<td bgcolor='#D1ECF1' height='80'>MBR</td>"
	
	total := float64(mbr.Mbr_tamano)
	sizeofMBR := int32(1024)
	
	var particiones []structs.Partition
	for _, part := range mbr.Mbr_partitions {
		if part.Part_s > 0 {
			particiones = append(particiones, part)
		}
	}
	
	sort.Slice(particiones, func(i, j int) bool {
		return particiones[i].Part_start < particiones[j].Part_start
	})
	
	lastEnd := sizeofMBR
	
	for _, part := range particiones {
		if part.Part_start > lastEnd {
			freeSpace := part.Part_start - lastEnd
			porcentajeLibre := float64(freeSpace) / total * 100
			dot += fmt.Sprintf("<td bgcolor='#F8F9FA' height='80'>Libre<br/>%.0f%% del disco</td>", porcentajeLibre)
		}
		
		porcentaje := float64(part.Part_s) / total * 100
		
		if part.Part_type == 'e' || part.Part_type == 'E' {
			dot += "<td bgcolor='#FFE5B4' height='80'>"
			dot += "<table border='1' cellborder='1' cellspacing='0' style='width:100%;'>"
			dot += fmt.Sprintf("<tr><td colspan='10' bgcolor='#FFD93D'><b>Extendida</b></td></tr>")
			dot += "<tr>"
			logicas := obtenerParticionesLogicasParaDisk(disk, part.Part_start, part.Part_s, total)
			dot += logicas
			
			dot += "</tr></table>"
			dot += "</td>"
		} else {
			dot += fmt.Sprintf("<td bgcolor='#C8E6C9' height='80'>Primaria<br/>%.0f%% del disco</td>", porcentaje)
		}
		
		lastEnd = part.Part_start + part.Part_s
	}
	
	if lastEnd < int32(total) {
		freeSpace := int32(total) - lastEnd
		porcentajeLibre := float64(freeSpace) / total * 100
		dot += fmt.Sprintf("<td bgcolor='#F8F9FA' height='80'>Libre<br/>%.0f%% del disco</td>", porcentajeLibre)
	}
	
	dot += "</tr></table>\n>];\n}\n"

	dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
	err = os.WriteFile(dotFile, []byte(dot), 0644)
	if err != nil {
		return fmt.Sprintf("Error escribiendo DOT: %v", err)
	}

	format := strings.TrimPrefix(filepath.Ext(path), ".")
	cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
	err = cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error ejecutando Graphviz: %v", err)
	}

	return fmt.Sprintf("Reporte DISK generado en %s", path)
}

func obtenerParticionesLogicasParaDisk(disk *os.File, startExtendida int32, sizeExtendida int32, totalDisk float64) string {
	var result string
	ebrPos := int64(startExtendida)
	currentPos := startExtendida
	
	for ebrPos > 0 {
		var ebr structs.EBR
		if _, err := disk.Seek(ebrPos, io.SeekStart); err != nil {
			break
		}
		if err := binary.Read(disk, binary.LittleEndian, &ebr); err != nil {
			break
		}
		
		result += "<td bgcolor='#D1ECF1'>EBR</td>"
		
		if ebr.Part_mount == 1 && ebr.Part_s > 0 {
			ebrSize := int32(1024)
			if ebr.Part_start > currentPos + ebrSize {
				freeSpace := ebr.Part_start - currentPos - ebrSize
				porcentajeLibre := float64(freeSpace) / totalDisk * 100
				if porcentajeLibre > 0 {
					result += fmt.Sprintf("<td bgcolor='#F8F9FA'>Libre<br/>%.0f%% del Disco</td>", porcentajeLibre)
				}
			}
			
			porcentajeLogica := float64(ebr.Part_s) / totalDisk * 100
			result += fmt.Sprintf("<td bgcolor='#C8E6C9'>Lógica<br/>%.0f%% del Disco</td>", porcentajeLogica)
			
			currentPos = ebr.Part_start + ebr.Part_s
		}
		
		if ebr.Part_next == -1 || ebr.Part_next == 0 {
			break
		}
		ebrPos = int64(ebr.Part_next)
	}
	
	extendedEnd := startExtendida + sizeExtendida
	if currentPos < extendedEnd {
		freeAtEnd := extendedEnd - currentPos
		porcentajeLibreFin := float64(freeAtEnd) / totalDisk * 100
		if porcentajeLibreFin > 0 {
			result += fmt.Sprintf("<td bgcolor='#F8F9FA'>Libre<br/>%.0f%% del disco</td>", porcentajeLibreFin)
		}
	}
	
	return result
}

func generarReporteInode(path, id string) string {
	disk, sb, _, err := structs.SuperBloque_ID(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer disk.Close()
	
	dot := "digraph G {\n"
	dot += "node [shape=plaintext]\n"
	dot += "rankdir=LR\n"
	
	var inodosUsados []int
	var conexiones string

	for i := 0; i < int(sb.S_inodes_count); i++ {
		inode, usado := structs.ObtenerInodo(disk, sb, i)
		if usado {
			inodosUsados = append(inodosUsados, i)

			dot += fmt.Sprintf("Inode%d [label=<\n", i)
			dot += "<table border='1' cellborder='1' cellspacing='0' bgcolor='white'>\n"
			dot += fmt.Sprintf("<tr><td colspan='2' bgcolor='#E8F4FD'><b>Inodo %d</b></td></tr>\n", i)
			dot += fmt.Sprintf("<tr bgcolor='white'><td><b>i_uid</b></td><td>%d</td></tr>\n", inode.I_uid)
			dot += fmt.Sprintf("<tr bgcolor='white'><td><b>i_size</b></td><td>%d</td></tr>\n", inode.I_s)
			dot += fmt.Sprintf("<tr bgcolor='white'><td><b>i_atime</b></td><td>30/11/2015 14:25</td></tr>\n")
			dot += "<tr bgcolor='white'><td colspan='2'>.</td></tr>\n"
			dot += fmt.Sprintf("<tr bgcolor='white'><td><b>i_block_1</b></td><td>%d</td></tr>\n", inode.I_block[0])
			dot += fmt.Sprintf("<tr bgcolor='white'><td><b>i_block_2</b></td><td>%d</td></tr>\n", inode.I_block[1])
			dot += "<tr bgcolor='white'><td colspan='2'>.</td></tr>\n"
			dot += fmt.Sprintf("<tr bgcolor='white'><td><b>i_perm</b></td><td>%d</td></tr>\n", inode.I_perm)
			dot += "<tr bgcolor='white'><td colspan='2'>.</td></tr>\n"		
			dot += "</table>\n>];\n"
		}
	}
	
	for i := 0; i < len(inodosUsados)-1; i++ {
		conexiones += fmt.Sprintf("Inode%d -> Inode%d [color=\"#4A90E2\" style=\"solid\"];\n", 
			inodosUsados[i], inodosUsados[i+1])
	}
	
	dot += conexiones
	dot += "}\n"
	
	dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
	err = os.WriteFile(dotFile, []byte(dot), 0644)
	if err != nil {
		return fmt.Sprintf("Error escribiendo DOT: %v", err)
	}
	
	format := strings.TrimPrefix(filepath.Ext(path), ".")
	cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
	err = cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error ejecutando Graphviz: %v", err)
	}
	
	return fmt.Sprintf("Reporte Inode generado en %s", path)
}

func generarReporteBlock(path string, id string) string {
    f, sb, _, err := structs.SuperBloque_ID(id)
    if err != nil {
        return fmt.Sprintf("Error obteniendo FS: %v", err)
    }
    defer f.Close()

    dir := filepath.Dir(path)
    base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Sprintf("Error creando directorio %s: %v", dir, err)
    }

    dotPath := filepath.Join(dir, base+".dot")
    imgPath := path

    tiposBloques := make(map[int]string)

    for i := 0; i < int(sb.S_inodes_count); i++ {
        inode, usado := structs.ObtenerInodo(f, sb, i)
        if !usado {
            continue
        }

        esArchivo := inode.I_type[0] == 1
        tipoBloque := "archivo"
        if !esArchivo {
            tipoBloque = "carpeta"
        }

        for j := 0; j < 12; j++ {
            if inode.I_block[j] >= 0 {
                tiposBloques[int(inode.I_block[j])] = tipoBloque
            }
        }

        if inode.I_block[12] >= 0 {
            tiposBloques[int(inode.I_block[12])] = "apuntadores"

            bp, ok := structs.ObtenerBloqueApuntadores(f, sb, int(inode.I_block[12]))
            if ok {
                for _, ptr := range bp.B_pointers {
                    if ptr >= 0 {
                        tiposBloques[int(ptr)] = tipoBloque
                    }
                }
            }
        }

        if inode.I_block[13] >= 0 {
            tiposBloques[int(inode.I_block[13])] = "apuntadores"
            
            bp1, ok := structs.ObtenerBloqueApuntadores(f, sb, int(inode.I_block[13]))
            if ok {
                for _, ptr1 := range bp1.B_pointers {
                    if ptr1 >= 0 {
                        tiposBloques[int(ptr1)] = "apuntadores"
                        
                        bp2, ok := structs.ObtenerBloqueApuntadores(f, sb, int(ptr1))
                        if ok {
                            for _, ptr2 := range bp2.B_pointers {
                                if ptr2 >= 0 {
                                    tiposBloques[int(ptr2)] = tipoBloque
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    var b strings.Builder
    b.WriteString("digraph G {\n")
    b.WriteString("  node [shape=plaintext fontname=\"Arial\"];\n")
    b.WriteString("  rankdir=LR;\n")
    b.WriteString("  bgcolor=transparent;\n\n")
    
    bitmapBloques := structs.GetBitmapBlocks(f, sb)
    if bitmapBloques == nil {
        return "Error: No se pudo obtener el bitmap de bloques"
    }

    bloquesUsados := obtenerBloquesUsados(bitmapBloques, int(sb.S_blocks_count))
    
    for _, blockNum := range bloquesUsados {
        tipoBloque, existe := tiposBloques[blockNum]
        if !existe {
            tipoBloque = detectarTipoBloqueContenido(f, sb, blockNum)
        }

        switch tipoBloque {
        case "carpeta":
            bc, ok := structs.LeerBloqueCarpeta(f, sb, int32(blockNum))
            if ok {
                b.WriteString(generarTablaCarpeta(int32(blockNum), &bc))
            }

        case "archivo":
            ba, ok := structs.LeerBloqueArchivo(f, sb, int32(blockNum))
            if ok {
                b.WriteString(generarTablaArchivo(int32(blockNum), &ba))
            }

        case "apuntadores":
            bp, ok := structs.ObtenerBloqueApuntadores(f, sb, blockNum)
            if ok {
                b.WriteString(generarTablaApuntadores(int32(blockNum), &bp))
            }
        }
    }
    
    b.WriteString("\n")
    
    for i := 0; i < len(bloquesUsados)-1; i++ {
        b.WriteString(fmt.Sprintf("  Block%d -> Block%d [style=invis];\n", 
            bloquesUsados[i], bloquesUsados[i+1]))
    }

    b.WriteString("}\n")

    if err := os.WriteFile(dotPath, []byte(b.String()), 0644); err != nil {
        return fmt.Sprintf("Error creando archivo DOT: %v", err)
    }

    cmd := exec.Command("dot", "-Tjpg", dotPath, "-o", imgPath)
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Sprintf("Error generando imagen JPG: %v\nOutput: %s", err, string(output))
    }

    return fmt.Sprintf("Reporte generado: %s", imgPath)
}

func detectarTipoBloqueContenido(f *os.File, sb *structs.SuperBloque, blockNum int) string {
    bc, okDir := structs.LeerBloqueCarpeta(f, sb, int32(blockNum))
    if okDir {
        validEntries := 0
        for _, content := range bc.B_content {
            if content.B_inodo >= 0 && content.B_inodo < sb.S_inodes_count {
                nombre := strings.TrimRight(string(content.B_name[:]), "\x00")
                if len(nombre) > 0 && esNombreValido(nombre) {
                    validEntries++
                }
            }
        }
        if validEntries > 0 {
            return "carpeta"
        }
    }

    bp, okPtr := structs.ObtenerBloqueApuntadores(f, sb, blockNum)
    if okPtr {
        validPointers := 0
        for _, ptr := range bp.B_pointers {
            if (ptr >= 0 && ptr < sb.S_blocks_count) || ptr == -1 {
                validPointers++
            }
        }
        if validPointers >= 4 {
            return "apuntadores"
        }
    }

    return "archivo"
}

func obtenerBloquesUsados(bitmap []byte, totalBloques int) []int {
    bloquesUsados := make([]int, 0)
    
    for i := 0; i < totalBloques && i < len(bitmap); i++ {
        if bitmap[i] != 0 {
            bloquesUsados = append(bloquesUsados, i)
        }
    }
    
    return bloquesUsados
}

func esNombreValido(nombre string) bool {
    if nombre == "." || nombre == ".." {
        return true
    }
    
    for _, c := range nombre {
        if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || 
             (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_' || c == ' ') {
            return false
        }
    }
    return true
}

func generarTablaCarpeta(num int32, carpeta *structs.BCarpeta) string {
    var tabla strings.Builder
    tabla.WriteString(fmt.Sprintf("  Block%d [label=<\n", num))
    tabla.WriteString("    <table border='1' cellborder='1' cellspacing='0' bgcolor='white'>\n")
    tabla.WriteString(fmt.Sprintf("      <tr><td colspan='2' bgcolor='#4CAF50'><font color='white'><b>Bloque Carpeta %d</b></font></td></tr>\n", num))
    tabla.WriteString("      <tr><td bgcolor='#E8F5E8'><b>b_name</b></td><td bgcolor='#E8F5E8'><b>b_inodo</b></td></tr>\n")

    for _, c := range carpeta.B_content {
        nombre := strings.TrimRight(string(c.B_name[:]), "\x00")
        if c.B_inodo >= 0 {
            tabla.WriteString(fmt.Sprintf("      <tr><td>%s</td><td>%d</td></tr>\n", nombre, c.B_inodo))
        }
    }

    tabla.WriteString("    </table>>];\n\n")
    return tabla.String()
}

func generarTablaArchivo(num int32, archivo *structs.BArchivo) string {
    contenido := strings.TrimRight(string(archivo.B_content[:]), "\x00")
    
    if len(contenido) > 50 {
        contenido = contenido[:47] + "..."
    }
    
    contenido = strings.ReplaceAll(contenido, "&", "&amp;")
    contenido = strings.ReplaceAll(contenido, "<", "&lt;")
    contenido = strings.ReplaceAll(contenido, ">", "&gt;")
    contenido = strings.ReplaceAll(contenido, "\"", "&quot;")
    contenido = strings.ReplaceAll(contenido, "\n", "<br/>")
    
    var tabla strings.Builder
    tabla.WriteString(fmt.Sprintf("  Block%d [label=<\n", num))
    tabla.WriteString("    <table border='1' cellborder='1' cellspacing='0' bgcolor='white'>\n")
    tabla.WriteString(fmt.Sprintf("      <tr><td bgcolor='#2196F3'><font color='white'><b>Bloque Archivo %d</b></font></td></tr>\n", num))
    tabla.WriteString(fmt.Sprintf("      <tr><td align='left'><font face='monospace' point-size='10'>%s</font></td></tr>\n", contenido))
    tabla.WriteString("    </table>>];\n\n")

    return tabla.String()
}

func generarTablaApuntadores(num int32, apuntadores *structs.BApuntadores) string {
    var tabla strings.Builder
    tabla.WriteString(fmt.Sprintf("  Block%d [label=<\n", num))
    tabla.WriteString("    <table border='1' cellborder='1' cellspacing='0' bgcolor='white'>\n")
    tabla.WriteString(fmt.Sprintf("      <tr><td bgcolor='#FF9800'><font color='white'><b>Bloque Apuntadores %d</b></font></td></tr>\n", num))
    tabla.WriteString("      <tr><td align='left'>\n")
    
    tabla.WriteString("        <table border='0' cellborder='1' cellspacing='0'>\n")
    for i := 0; i < 16; i += 4 {
        tabla.WriteString("          <tr>")
        for j := 0; j < 4 && i+j < 16; j++ {
            if apuntadores.B_pointers[i+j] != -1 {
                tabla.WriteString(fmt.Sprintf("<td>%d</td>", apuntadores.B_pointers[i+j]))
            } else {
                tabla.WriteString("<td>-1</td>")
            }
        }
        tabla.WriteString("</tr>\n")
    }
    tabla.WriteString("        </table>\n")
    
    tabla.WriteString("      </td></tr>\n")
    tabla.WriteString("    </table>>];\n\n")
    
    return tabla.String()
}

func generarReporteBMInode(path, id string) string {
    disk, sb, _, err := structs.SuperBloque_ID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    bm := structs.BitMapInodos(disk, sb)
    var contenido strings.Builder
    
    contenido.WriteString("BITMAP DE INODOS\n")
    contenido.WriteString("================\n\n")
    
    for i, b := range bm {
        contenido.WriteString(fmt.Sprintf("%d", b))
        if (i+1)%20 == 0 {
            contenido.WriteString("\n")
        }
    }
    
    if len(bm)%20 != 0 {
        contenido.WriteString("\n")
    }

    return escribirTexto(path, contenido.String())
}

func generarReporteBMBlock(path, id string) string {
    disk, sb, _, err := structs.SuperBloque_ID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    bm := structs.GetBitmapBlocks(disk, sb)
    var contenido strings.Builder
    
    contenido.WriteString("BITMAP DE BLOQUES\n")
    contenido.WriteString("=================\n\n")
    
    for i, b := range bm {
        contenido.WriteString(fmt.Sprintf("%d", b))
        if (i+1)%20 == 0 {
            contenido.WriteString("\n")
        }
    }
    
    if len(bm)%20 != 0 {
        contenido.WriteString("\n")
    }

    return escribirTexto(path, contenido.String())
}

func generarReporteTree(path, id string) string {
    disk, sb, _, err := structs.SuperBloque_ID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    dot := structs.GenerarReporteArbol(disk, sb, 50)

    dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
    err = os.WriteFile(dotFile, []byte(dot), 0644)
    if err != nil {
        return fmt.Sprintf("Error escribiendo DOT: %v", err)
    }
    
    format := strings.TrimPrefix(filepath.Ext(path), ".")
    cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
    err = cmd.Run()
    if err != nil {
        return fmt.Sprintf("Error ejecutando Graphviz: %v", err)
    }
    
    return fmt.Sprintf("Reporte Tree generado en %s", path)
}

func generarReporteSB(path, id string) string {
    disk, sb, _, err := structs.SuperBloque_ID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    var dot strings.Builder
    dot.WriteString("digraph G {\nnode [shape=plaintext]\nSB [label=<\n")
    dot.WriteString("<table border='1' cellborder='1' cellspacing='0'>\n")
    dot.WriteString("<tr><td colspan='2' bgcolor='#1976d2'><font color='white'><b>Reporte de SUPERBLOQUE</b></font></td></tr>\n")

    nombreDisco := structs.NombreDisco_ID(id)
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_nombre_hd</font></td><td>%s</td></tr>\n", nombreDisco))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_filesystem_type</font></td><td>%d</td></tr>\n", sb.S_filesystem_type))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_inodos_count</font></td><td>%d</td></tr>\n", sb.S_inodes_count))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_bloques_count</font></td><td>%d</td></tr>\n", sb.S_blocks_count))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_inodos_free</font></td><td>%d</td></tr>\n", sb.S_free_inodes_count))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_bloques_free</font></td><td>%d</td></tr>\n", sb.S_free_blocks_count))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_date_creacion</font></td><td>%s</td></tr>\n", strings.Trim(string(sb.S_mtime[:]), "\x00")))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_date_ultimo_montaje</font></td><td>%s</td></tr>\n", strings.Trim(string(sb.S_umtime[:]), "\x00")))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_montajes_count</font></td><td>%d</td></tr>\n", sb.S_mnt_count))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_magic_num</font></td><td>%d</td></tr>\n", sb.S_magic))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_size_struct_inodo</font></td><td>%d</td></tr>\n", sb.S_inode_s))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_size_struct_bloque</font></td><td>%d</td></tr>\n", sb.S_block_s))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_first_free_bit_tabla_inodos</font></td><td>%d</td></tr>\n", sb.S_first_ino))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_first_free_bit_bloques</font></td><td>%d</td></tr>\n", sb.S_first_blo))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_ap_bitmap_inodos</font></td><td>%d</td></tr>\n", sb.S_bm_inode_start))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_ap_bitmap_bloques</font></td><td>%d</td></tr>\n", sb.S_bm_block_start))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_ap_inodos</font></td><td>%d</td></tr>\n", sb.S_inode_start))
    dot.WriteString(fmt.Sprintf("<tr><td bgcolor='#2e7d32'><font color='white'>sb_ap_bloques</font></td><td>%d</td></tr>\n", sb.S_block_start))

    dot.WriteString("</table>\n>];\n}\n")
    return generarArchivoDOT(dot.String(), path)
}

func generarReporteFile(path, id, filePath string) string {
    contenido, err := structs.LeerArchivoDeFS(id, filePath)
    if err != nil {
        return fmt.Sprintf("Error leyendo archivo: %v", err)
    }

    nombre := filepath.Base(filePath)
    if contenido == "" {
        contenido = "<Archivo vacío>"
    }

    reporte := fmt.Sprintf(
        "REPORTE DE FILE\n================\n\nNombre: %s\nRuta: %s\nTamaño: %d bytes\n\nContenido:\n\n%s\n",
        nombre,
        filePath,
        len(contenido),
        contenido,
    )

    return escribirTexto(path, reporte)
}

func generarReporteLS(id, dirPath, outputPath string) error {
    listado, err := structs.ListaCarpetasFS(id, dirPath)
    if err != nil {
        return fmt.Errorf("error al listar directorio: %v", err)
    }

    dot := `digraph G {
    node [shape=plaintext]
    ls [label=<
    <TABLE BORDER="1" CELLBORDER="1" CELLSPACING="0">
    <TR>
        <TD><B>Permisos</B></TD>
        <TD><B>Propietario</B></TD>
        <TD><B>Grupo</B></TD>
        <TD><B>Tamaño (Bytes)</B></TD>
        <TD><B>Fecha Creación</B></TD>
        <TD><B>Hora Creación</B></TD>
        <TD><B>Fecha Modificación</B></TD>
        <TD><B>Hora Modificación</B></TD>
        <TD><B>Tipo</B></TD>
        <TD><B>Nombre</B></TD>
    </TR>
    `

    for _, f := range listado {
        size := fmt.Sprintf("%d", f.Size)

        fechaCreacion := formatearFecha(f.Creacion)
        horaCreacion := formatearHora(f.Creacion)

        fechaMod := formatearFecha(f.Modificacion)
        horaMod := formatearHora(f.Modificacion)

        dot += fmt.Sprintf(
            "    <TR><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD></TR>\n",
            f.Permisos, f.Propietario, f.Grupo, size,
            fechaCreacion, horaCreacion,
            fechaMod, horaMod,
            f.Tipo, f.Nombre,
        )
    }

    dot += `</TABLE>
    >];
}`

    dotPath := outputPath + ".dot"
    if err := os.WriteFile(dotPath, []byte(dot), 0644); err != nil {
        return fmt.Errorf("error escribiendo archivo dot: %v", err)
    }

    cmd := exec.Command("dot", "-Tpng", dotPath, "-o", outputPath)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("error generando imagen con dot: %v", err)
    }

    return nil
}

func formatearFecha(raw string) string {
    if len(raw) < 10 {
        return raw
    }
    t, err := time.Parse("2006-01-02 15:04:05", raw)
    if err != nil {
        return raw
    }
    return t.Format("02/01/2006")
}

func formatearHora(raw string) string {
    if len(raw) < 10 {
        return raw
    }
    t, err := time.Parse("2006-01-02 15:04:05", raw)
    if err != nil {
        return raw
    }
    return t.Format("15:04")
}

func escribirTexto(path, content string) string {
    err := os.WriteFile(path, []byte(content), 0644)
    if err != nil {
        return fmt.Sprintf("Error escribiendo archivo: %v", err)
    }
    return fmt.Sprintf("Reporte generado en %s", path)
}

