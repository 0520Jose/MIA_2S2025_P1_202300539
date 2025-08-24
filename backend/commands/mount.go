package commands

import (
    "backend/structs"
    "fmt"
    "os"
    "strings"
)

var (
    particionesMontadas []structs.PartitionMount
    contadorDiscos      = make(map[string]int)    // path -> correlativo (inicia en 0, se incrementa antes de usar)
    letrasDiscos        = make(map[string]string) // path -> letra asignada
    letraActual         = 'A'
)

func Mount(params map[string]string) string {
    // Validar parámetros permitidos y normalizar llaves a minúsculas
    allowed := map[string]struct{}{
        "-path": {}, "-name": {},
    }
    normalized := make(map[string]string, len(params))
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if _, ok := allowed[lk]; !ok {
            return fmt.Sprintf("Error: parámetro no reconocido: %s\n", k)
        }
        normalized[lk] = v
    }

    // Parámetros obligatorios
    path, ok := normalized["-path"]
    if !ok || strings.TrimSpace(path) == "" {
        return "Error: parámetros -path y -name son obligatorios\n"
    }
    name, ok := normalized["-name"]
    if !ok || strings.TrimSpace(name) == "" {
        return "Error: parámetros -path y -name son obligatorios\n"
    }

    // Soporte comillas dobles en path
    var err error
    path, err = limpiarRuta(path)
    if err != nil {
        return fmt.Sprintf("Error en la ruta: %v\n", err)
    }

    // Verificar existencia del archivo de disco
    archivo, err := os.OpenFile(path, os.O_RDONLY, 0)
    if err != nil {
        return fmt.Sprintf("Error abriendo el disco: %v\n", err)
    }
    defer archivo.Close()

    // Leer MBR
    mbr, err := structs.LeerMBR(archivo)
    if err != nil {
        return fmt.Sprintf("Error leyendo MBR: %v\n", err)
    }

    // Buscar partición primaria por nombre
    var particion structs.Partition
    encontrada := false
    for _, p := range mbr.Mbr_partitions {
        partitionName := strings.TrimRight(string(p.Part_name[:]), "\x00")
        if partitionName == name && p.Part_type == 'P' {
            particion = p
            encontrada = true
            break
        }
    }
    if !encontrada {
        return "Error: no se encontró una partición primaria con ese nombre\n"
    }

    // Evitar montar duplicado mismo path+name
    for _, pm := range particionesMontadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        if pm.Path == path && pmName == name {
            return "Error: partición ya montada\n"
        }
    }

    // Generar letra por disco y correlativo por disco (inicia en 1)
    carnetUltimos2 := "39" // últimos dos dígitos del carnet
    letra, exists := letrasDiscos[path]
    if !exists {
        letra = string(letraActual)
        letrasDiscos[path] = letra
        letraActual++
        contadorDiscos[path] = 0
    }
    contadorDiscos[path]++
    correlativo := contadorDiscos[path]

    id := fmt.Sprintf("%s%d%s", carnetUltimos2, correlativo, letra)

    // Actualizar estado SOLO en memoria (no escribir al disco)
    particion.Part_status = 1
    particion.Part_correlative = int32(correlativo)
    copy(particion.Part_id[:], id)

    // Registrar montaje en memoria
    particionesMontadas = append(particionesMontadas, structs.PartitionMount{
        Id:        id,
        Path:      path,
        Partition: particion,
    })
    structs.Particiones_Montadas = particionesMontadas

    // Salida: listar montajes actuales
    var output strings.Builder
    for _, pm := range particionesMontadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        output.WriteString(fmt.Sprintf(" - %s: %s (%s)\n", pm.Id, pmName, pm.Path))
    }
    return output.String()
}