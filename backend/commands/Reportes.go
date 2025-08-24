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
    dot += "node [shape=plaintext]\n"
    dot += "ReporteMBR [label=<\n"
    dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
    dot += "<tr><td colspan='2'><b>REPORTE DE MBR</b></td></tr>\n"
    dot += fmt.Sprintf("<tr><td>mbr_tamano</td><td>%d</td></tr>\n", mbr.Mbr_tamano)
    dot += fmt.Sprintf("<tr><td>mbr_fecha_creacion</td><td>%s</td></tr>\n", strings.Trim(string(mbr.Mbr_fecha_creacion[:]), "\x00"))
    dot += fmt.Sprintf("<tr><td>mbr_disk_signature</td><td>%d</td></tr>\n", mbr.Mbr_dsk_signature)
    dot += fmt.Sprintf("<tr><td>dsk_fit</td><td>%c</td></tr>\n", mbr.Dsk_fit)

    // Particiones primarias y extendidas
    for i, part := range mbr.Mbr_partitions {
        if part.Part_s > 0 {
            dot += fmt.Sprintf("<tr><td colspan='2'><b>Partición %d</b></td></tr>\n", i+1)
            dot += fmt.Sprintf("<tr><td>part_status</td><td>%d</td></tr>\n", part.Part_status)
            
            // Determinar tipo de partición
            tipoParticion := "Primaria"
            if part.Part_type == 'e' || part.Part_type == 'E' {
                tipoParticion = "Extendida"
            }
            dot += fmt.Sprintf("<tr><td>part_type</td><td>%s (%c)</td></tr>\n", tipoParticion, part.Part_type)
            dot += fmt.Sprintf("<tr><td>part_fit</td><td>%c</td></tr>\n", part.Part_fit)
            dot += fmt.Sprintf("<tr><td>part_start</td><td>%d</td></tr>\n", part.Part_start)
            dot += fmt.Sprintf("<tr><td>part_size</td><td>%d</td></tr>\n", part.Part_s)
            dot += fmt.Sprintf("<tr><td>part_name</td><td>%s</td></tr>\n", strings.Trim(string(part.Part_name[:]), "\x00"))
            dot += fmt.Sprintf("<tr><td>part_correlative</td><td>%d</td></tr>\n", part.Part_correlative)
            
            // Si es extendida, leer particiones lógicas
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
    ebrPos := startExtendida
    logicalNum := 1

    for ebrPos != -1 && ebrPos != 0 {
        // Leer EBR en la posición actual
        var ebr structs.EBR
        if _, err := disk.Seek(int64(ebrPos), io.SeekStart); err != nil {
            break
        }
        if err := binary.Read(disk, binary.LittleEndian, &ebr); err != nil {
            break
        }

        // Si el EBR tiene una partición lógica válida
        if ebr.Part_s > 0 {
            dot += fmt.Sprintf("<tr><td colspan='2'><b>Partición Lógica %d</b></td></tr>\n", logicalNum)
            dot += fmt.Sprintf("<tr><td>part_status</td><td>%d</td></tr>\n", ebr.Part_mount)
            dot += fmt.Sprintf("<tr><td>part_type</td><td>Lógica</td></tr>\n")
            dot += fmt.Sprintf("<tr><td>part_fit</td><td>%c</td></tr>\n", ebr.Part_fit)
            dot += fmt.Sprintf("<tr><td>part_start</td><td>%d</td></tr>\n", ebr.Part_start)
            dot += fmt.Sprintf("<tr><td>part_size</td><td>%d</td></tr>\n", ebr.Part_s)
            dot += fmt.Sprintf("<tr><td>part_name</td><td>%s</td></tr>\n", strings.Trim(string(ebr.Part_name[:]), "\x00"))
            logicalNum++
        }

        // Ir al siguiente EBR
        ebrPos = ebr.Part_next
        if ebrPos == 0 || ebrPos == -1 {
            break
        }
    }

    return dot
}

func generarReporteDISK(path, id string) string {
	disk, _, mbr, err := structs.GetFileSystemByID(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer disk.Close()

	dot := "digraph G {\n"
	dot += "node [shape=plaintext]\n"
	dot += "ReporteDISK [label=<\n"
	dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
	dot += "<tr><td colspan='100'><b>DISCO</b></td></tr>\n<tr>"

	dot += "<td>MBR</td>"

	total := float64(mbr.Mbr_tamano)
	for _, part := range mbr.Mbr_partitions {
		if part.Part_s > 0 {
			porcentaje := float64(part.Part_s) / total * 100
			label := fmt.Sprintf("%s<br/>%.2f%%", strings.Trim(string(part.Part_name[:]), "\x00"), porcentaje)
			if part.Part_type == 'e' || part.Part_type == 'E' {
				dot += "<td>Extendida</td>"
			} else {
				dot += fmt.Sprintf("<td>%s</td>", label)
			}
		}
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

func generarReporteInode(path, id string) string {
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    dot := "digraph G {\nnode [shape=plaintext]\nInodes [label=<\n"
    dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
    dot += "<tr><td colspan='7'><b>Reporte de Inodos</b></td></tr>\n"

    for i := 0; i < int(sb.S_inodes_count); i++ {
        inode, usado := structs.GetInode(disk, sb, i)
        if usado {
            dot += fmt.Sprintf("<tr><td colspan='7'><b>Inodo %d</b></td></tr>\n", i)
            dot += fmt.Sprintf("<tr><td>UID</td><td>%d</td></tr>\n", inode.I_uid)
            dot += fmt.Sprintf("<tr><td>GID</td><td>%d</td></tr>\n", inode.I_gid)
            dot += fmt.Sprintf("<tr><td>Size</td><td>%d</td></tr>\n", inode.I_s)
            dot += fmt.Sprintf("<tr><td>Type</td><td>%c</td></tr>\n", inode.I_type)
        }
    }

    dot += "</table>\n>];\n}\n"

    return generarArchivoDOT(dot, path)
}

func generarReporteBlock(path, id string) string {
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    dot := "digraph G {\nnode [shape=plaintext]\nBlocks [label=<\n"
    dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
    dot += "<tr><td colspan='2'><b>Reporte de Bloques</b></td></tr>\n"

    for i := 0; i < int(sb.S_blocks_count); i++ {
        block, usado := structs.GetBlock(disk, sb, i)
        if usado {
            dot += fmt.Sprintf("<tr><td colspan='2'><b>Bloque %d</b></td></tr>\n", i)
            dot += fmt.Sprintf("<tr><td>Contenido</td><td>%s</td></tr>\n", block.ContentString())
        }
    }

    dot += "</table>\n>];\n}\n"
    return generarArchivoDOT(dot, path)
}

func generarReporteBMInode(path, id string) string {
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    bm := structs.GetBitmapInodes(disk, sb)
    var contenido strings.Builder
    for i, b := range bm {
        contenido.WriteString(fmt.Sprintf("%d", b))
        if (i+1)%20 == 0 {
            contenido.WriteString("\n")
        }
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
    for i, b := range bm {
        contenido.WriteString(fmt.Sprintf("%d", b))
        if (i+1)%20 == 0 {
            contenido.WriteString("\n")
        }
    }

    return writeTextFile(path, contenido.String())
}

func generarReporteTree(path, id string) string {
    disk, sb, _, err := structs.GetSuperBlockByID(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    dot := "digraph G {\nnode [shape=record];\n"
    dot += structs.GenerateTreeGraph(disk, sb)
    dot += "}\n"

    return generarArchivoDOT(dot, path)
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

    return generarArchivoDOT(dot, path)
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
    return generarArchivoDOT(dot, path)
}

func generarArchivoDOT(dot, path string) string {
    dotFile := strings.TrimSuffix(path, filepath.Ext(path)) + ".dot"
    err := os.WriteFile(dotFile, []byte(dot), 0644)
    if err != nil {
        return fmt.Sprintf("Error escribiendo DOT: %v", err)
    }
    format := strings.TrimPrefix(filepath.Ext(path), ".")
    cmd := exec.Command("dot", "-T"+format, dotFile, "-o", path)
    err = cmd.Run()
    if err != nil {
        return fmt.Sprintf("Error ejecutando Graphviz: %v", err)
    }
    return fmt.Sprintf("Reporte generado en %s", path)
}

func writeTextFile(path, content string) string {
    err := os.WriteFile(path, []byte(content), 0644)
    if err != nil {
        return fmt.Sprintf("Error escribiendo archivo: %v", err)
    }
    return fmt.Sprintf("Reporte generado en %s", path)
}

