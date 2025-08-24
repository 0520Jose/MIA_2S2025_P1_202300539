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
    //"html"
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
	//case "block":
	//	return generarReporteBlock(path, id)
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
		return generarReporteLS(path, id, params["-path_file_ls"])
	default:
		return fmt.Sprintf("Error: reporte %s no válido", name)
	}
}

func generarReporteMBR(path, id string) string {
    disk, _, mbr, err := structs.GetFileSystemByID(id)
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

    // Para depurar: imprimir el contenido del DOT
    fmt.Println("=== CONTENIDO DOT ===")
    fmt.Println(dot)
    fmt.Println("=== FIN DOT ===")

    format := strings.TrimPrefix(filepath.Ext(path), ".")
    cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Sprintf("Error ejecutando Graphviz: %v\nOutput: %s\nDOT file: %s", err, string(output), dotFile)
    }

    return fmt.Sprintf("Reporte generado en %s", path)
}

func generarReporteDISK(path, id string) string {
	disk, _, mbr, err := structs.GetFileSystemByID(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer disk.Close()

	// Obtener información del archivo para el nombre del disco
	diskName := "Disco1.dsk" // Puedes extraer esto del path si es necesario
	
	dot := "digraph G {\n"
	dot += "node [shape=plaintext]\n"
	dot += "ReporteDISK [label=<\n"
	dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
	
	// Título del disco
	dot += fmt.Sprintf("<tr><td colspan='20' bgcolor='#E8F4FD'><b>%s</b></td></tr>\n", diskName)
	
	// Fila principal
	dot += "<tr>"
	
	// MBR (siempre presente)
	dot += "<td bgcolor='#D1ECF1' height='80'>MBR</td>"
	
	total := float64(mbr.Mbr_tamano)
	sizeofMBR := int32(1024) // Tamaño típico del MBR
	
	// Ordenar particiones por posición
	var particiones []structs.Partition
	for _, part := range mbr.Mbr_partitions {
		if part.Part_s > 0 {
			particiones = append(particiones, part)
		}
	}
	
	// Ordenar por posición de inicio
	sort.Slice(particiones, func(i, j int) bool {
		return particiones[i].Part_start < particiones[j].Part_start
	})
	
	lastEnd := sizeofMBR
	
	for _, part := range particiones {
		// Verificar si hay espacio libre antes de esta partición
		if part.Part_start > lastEnd {
			freeSpace := part.Part_start - lastEnd
			porcentajeLibre := float64(freeSpace) / total * 100
			dot += fmt.Sprintf("<td bgcolor='#F8F9FA' height='80'>Libre<br/>%.0f%% del disco</td>", porcentajeLibre)
		}
		
		porcentaje := float64(part.Part_s) / total * 100
		//partName := strings.Trim(string(part.Part_name[:]), "\x00")
		
		if part.Part_type == 'e' || part.Part_type == 'E' {
			// Partición extendida - crear subtabla
			dot += "<td bgcolor='#FFE5B4' height='80'>"
			dot += "<table border='1' cellborder='1' cellspacing='0' style='width:100%;'>"
			dot += fmt.Sprintf("<tr><td colspan='10' bgcolor='#FFD93D'><b>Extendida</b></td></tr>")
			dot += "<tr>"
			
			// Procesar particiones lógicas dentro de la extendida
			logicas := obtenerParticionesLogicasParaDisk(disk, part.Part_start, part.Part_s, total)
			dot += logicas
			
			dot += "</tr></table>"
			dot += "</td>"
		} else {
			// Partición primaria
			dot += fmt.Sprintf("<td bgcolor='#C8E6C9' height='80'>Primaria<br/>%.0f%% del disco</td>", porcentaje)
		}
		
		lastEnd = part.Part_start + part.Part_s
	}
	
	// Espacio libre al final
	if lastEnd < int32(total) {
		freeSpace := int32(total) - lastEnd
		porcentajeLibre := float64(freeSpace) / total * 100
		dot += fmt.Sprintf("<td bgcolor='#F8F9FA' height='80'>Libre<br/>%.0f%% del disco</td>", porcentajeLibre)
	}
	
	dot += "</tr></table>\n>];\n}\n"

	// Escribir archivo DOT
	dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
	err = os.WriteFile(dotFile, []byte(dot), 0644)
	if err != nil {
		return fmt.Sprintf("Error escribiendo DOT: %v", err)
	}

	// Generar imagen
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
	disk, sb, _, err := structs.GetSuperBlockByID(id)
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
		inode, usado := structs.GetInode(disk, sb, i)
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
	
	// Generar imagen
	format := strings.TrimPrefix(filepath.Ext(path), ".")
	cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
	err = cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error ejecutando Graphviz: %v", err)
	}
	
	return fmt.Sprintf("Reporte Inode generado en %s", path)
}

/*
func generarReporteBlock(path, id string) string {
	disk, sb, _, err := structs.GetSuperBlockByID(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer disk.Close()
	
	dot := "digraph G {\n"
	dot += "node [shape=plaintext]\n"
	dot += "rankdir=LR\n" // Organizar horizontalmente
	
	var bloquesUsados []int
	var conexiones string
	
	// Crear mapa de tipos de bloques basado en inodos
	tiposBloque := make(map[int]string)
	for i := 0; i < int(sb.S_inodes_count); i++ {
		inode, usado := structs.GetInode(disk, sb, i)
		if usado {
			isDirectory := len(inode.I_type) > 0 && inode.I_type[0] == 0
			
			// Revisar todos los bloques del inodo
			for j := 0; j < 15; j++ {
				blockIdx := inode.I_block[j]
				if blockIdx > 0 {
					if isDirectory {
						tiposBloque[int(blockIdx)] = "carpeta"
					} else {
						// Para archivos, revisar si es bloque de datos o apuntadores
						if j < 12 {
							tiposBloque[int(blockIdx)] = "archivo"
						} else {
							tiposBloque[int(blockIdx)] = "apuntadores"
						}
					}
				}
			}
		}
	}
	
	// Buscar todos los bloques usados
	for i := 0; i < int(sb.S_blocks_count); i++ {
		block, usado := structs.GetBlock(disk, sb, i)
		if usado {
			bloquesUsados = append(bloquesUsados, i)
			
			// Determinar tipo de bloque usando el mapa
			tipoBloque, contenido := determinarTipoBloquePorInodo(disk, sb, block, i, tiposBloque[i])
			
			// Crear nodo para cada bloque
			dot += fmt.Sprintf("Block%d [label=<\n", i)
			dot += "<table border='1' cellborder='1' cellspacing='0' bgcolor='white'>\n"
			dot += fmt.Sprintf("<tr><td colspan='2' bgcolor='#E8F4FD'><b>%s</b></td></tr>\n", tipoBloque)
			
			// Contenido específico según el tipo de bloque
			dot += contenido
			
			dot += "</table>\n>];\n"
		}
	}
	
	// Crear conexiones entre bloques (flechas como en la imagen)
	for i := 0; i < len(bloquesUsados)-1; i++ {
		conexiones += fmt.Sprintf("Block%d -> Block%d [color=\"#4A90E2\" style=\"solid\"];\n", 
			bloquesUsados[i], bloquesUsados[i+1])
	}
	
	dot += conexiones
	dot += "}\n"
	
	// Escribir archivo DOT
	dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
	err = os.WriteFile(dotFile, []byte(dot), 0644)
	if err != nil {
		return fmt.Sprintf("Error escribiendo DOT: %v", err)
	}
	
	// Generar imagen
	format := strings.TrimPrefix(filepath.Ext(path), ".")
	cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
	err = cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error ejecutando Graphviz: %v", err)
	}
	
	return fmt.Sprintf("Reporte Block generado en %s", path)
}

// Función para determinar el tipo de bloque y generar su contenido
func determinarTipoBloque(block interface{}, indice int) (string, string) {
	switch b := block.(type) {
	case *structs.BCarpeta:
		return fmt.Sprintf("Bloque Carpeta %d", indice), generarContenidoCarpeta(b)
	case *structs.BArchivo:
		return fmt.Sprintf("Bloque Archivo %d", indice), generarContenidoArchivo(b)
	case *structs.BApuntadores:
		return fmt.Sprintf("Bloque Apuntadores %d", indice), generarContenidoApuntadores(b)
	default:
		// Si no se puede determinar el tipo, mostrar contenido genérico
		return fmt.Sprintf("Bloque %d", indice), "<tr bgcolor='white'><td colspan='2'>Contenido no identificado</td></tr>\n"
	}
}

// Generar contenido para bloque de carpeta
func generarContenidoCarpeta(carpeta *structs.BCarpeta) string {
	var contenido string
	contenido += "<tr bgcolor='white'><td><b>b_name</b></td><td><b>b_inodo</b></td></tr>\n"
	
	for _, content := range carpeta.B_content {
		name := strings.Trim(string(content.B_name[:]), "\x00")
		if name != "" {
			contenido += fmt.Sprintf("<tr bgcolor='white'><td>%s</td><td>%d</td></tr>\n", name, content.B_inodo)
		}
	}
	
	return contenido
}

// Generar contenido para bloque de archivo
func generarContenidoArchivo(archivo *structs.BArchivo) string {
	var contenido string
	
	// Mostrar contenido del archivo como texto continuo
	data := strings.Trim(string(archivo.B_content[:]), "\x00")
	if data != "" {
		// Dividir en líneas para mejor visualización
		contenido += fmt.Sprintf("<tr bgcolor='white'><td colspan='2'>%s</td></tr>\n", 
			html.EscapeString(data))
	} else {
		contenido += "<tr bgcolor='white'><td colspan='2'>Archivo vacío</td></tr>\n"
	}
	
	return contenido
}

// Generar contenido para bloque de apuntadores
func generarContenidoApuntadores(apuntadores *structs.BApuntadores) string {
	var contenido string
	var valores []string
	
	for _, pointer := range apuntadores.B_pointers {
		valores = append(valores, fmt.Sprintf("%d", pointer))
	}
	
	// Mostrar apuntadores en formato similar a la imagen
	// Dividir en múltiples líneas para mejor visualización
	lineasPorFila := 6
	for i := 0; i < len(valores); i += lineasPorFila {
		fin := i + lineasPorFila
		if fin > len(valores) {
			fin = len(valores)
		}
		linea := strings.Join(valores[i:fin], ", ")
		contenido += fmt.Sprintf("<tr bgcolor='white'><td colspan='2'>%s,</td></tr>\n", linea)
	}
	
	return contenido
}

func determinarTipoBloquePorInodo(disk *os.File, sb *structs.SuperBloque, block interface{}, indice int, tipoEsperado string) (string, string) {
    // Si tenemos información del tipo esperado desde el mapa de inodos
    if tipoEsperado != "" {
        switch tipoEsperado {
        case "carpeta":
            if carpeta, ok := block.(*structs.BCarpeta); ok {
                return fmt.Sprintf("Bloque Carpeta %d", indice), generarContenidoCarpeta(carpeta)
            }
        case "archivo":
            if archivo, ok := block.(*structs.BArchivo); ok {
                return fmt.Sprintf("Bloque Archivo %d", indice), generarContenidoArchivo(archivo)
            }
        case "apuntadores":
            if apuntadores, ok := block.(*structs.BApuntadores); ok {
                return fmt.Sprintf("Bloque Apuntadores %d", indice), generarContenidoApuntadores(apuntadores)
            }
        }
    }
    
    // Si no tenemos tipo esperado, intentar determinar por el contenido del bloque
    return determinarTipoBloque(block, indice)
}*/


func generarReporteBMInode(path, id string) string {
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    bm := structs.GetBitmapInodes(disk, sb)
    var contenido strings.Builder
    
    // Agregar encabezado del reporte
    contenido.WriteString("BITMAP DE INODOS\n")
    contenido.WriteString("================\n\n")
    
    for i, b := range bm {
        contenido.WriteString(fmt.Sprintf("%d", b))
        if (i+1)%20 == 0 {
            contenido.WriteString("\n")
        }
    }
    
    // Asegurar que termine con nueva línea si no es múltiplo de 20
    if len(bm)%20 != 0 {
        contenido.WriteString("\n")
    }

    return writeTextFile(path, contenido.String())
}

func generarReporteBMBlock(path, id string) string {
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    bm := structs.GetBitmapBlocks(disk, sb)
    var contenido strings.Builder
    
    // Agregar encabezado del reporte
    contenido.WriteString("BITMAP DE BLOQUES\n")
    contenido.WriteString("=================\n\n")
    
    for i, b := range bm {
        contenido.WriteString(fmt.Sprintf("%d", b))
        if (i+1)%20 == 0 {
            contenido.WriteString("\n")
        }
    }
    
    // Asegurar que termine con nueva línea si no es múltiplo de 20
    if len(bm)%20 != 0 {
        contenido.WriteString("\n")
    }

    return writeTextFile(path, contenido.String())
}

func generarReporteTree(path, id string) string {
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    dot := structs.GenerateTreeGraph(disk, sb, 50)

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
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    dot := "digraph G {\nnode [shape=plaintext]\nSB [label=<\n"
    dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
    dot += "<tr><td colspan='2'><b>SUPERBLOQUE</b></td></tr>\n"
    dot += fmt.Sprintf("<tr><td>s_inodes_count</td><td>%d</td></tr>\n", sb.S_inodes_count)
    dot += fmt.Sprintf("<tr><td>s_blocks_count</td><td>%d</td></tr>\n", sb.S_blocks_count)
    dot += fmt.Sprintf("<tr><td>s_free_blocks_count</td><td>%d</td></tr>\n", sb.S_free_blocks_count)
    dot += fmt.Sprintf("<tr><td>s_free_inodes_count</td><td>%d</td></tr>\n", sb.S_free_inodes_count)
    dot += fmt.Sprintf("<tr><td>s_mtime</td><td>%s</td></tr>\n", string(sb.S_mtime[:]))
    dot += "</table>\n>];\n}\n"

    return "Superbloque generado correctamente"
}

func generarReporteFile(path, id, filePath string) string {
    contenido, err := structs.ReadFileFromFS(id, filePath)
    if err != nil {
        return fmt.Sprintf("Error leyendo archivo: %v", err)
    }
    return writeTextFile(path, fmt.Sprintf("Archivo: %s\n\n%s", filePath, contenido))
}

func generarReporteLS(path, id, dirPath string) string {
    listado, err := structs.ListDirectoryFS(id, dirPath)
    if err != nil {
        return fmt.Sprintf("Error listando: %v", err)
    }

    dot := "digraph G {\nnode [shape=plaintext]\nLS [label=<\n"
    dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
    dot += "<tr><td><b>Nombre</b></td><td><b>Tipo</b></td><td><b>Permisos</b></td><td><b>Propietario</b></td><td><b>Grupo</b></td><td><b>Creación</b></td><td><b>Modificación</b></td></tr>\n"

    for _, f := range listado {
        dot += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
            f.Nombre, f.Tipo, f.Permisos, f.Propietario, f.Grupo, f.Creacion, f.Modificacion)
    }

    dot += "</table>\n>];\n}\n"

    return "Listado generado correctamente"
}

func writeTextFile(path, content string) string {
    err := os.WriteFile(path, []byte(content), 0644)
    if err != nil {
        return fmt.Sprintf("Error escribiendo archivo: %v", err)
    }
    return fmt.Sprintf("Reporte generado en %s", path)
}

