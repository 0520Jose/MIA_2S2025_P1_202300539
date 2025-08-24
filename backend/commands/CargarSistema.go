package commands

import (
    "backend/structs"
    "fmt"
    "os"
)

func CargarSistemaEXT2(partitionID string) (*os.File, *structs.SuperBloque, error) {
    var pm *structs.PartitionMount
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == partitionID {
            pm = &structs.Particiones_Montadas[i]
            break
        }
    }
    if pm == nil {
        return nil, nil, fmt.Errorf("partición con ID %s no está montada", partitionID)
    }

    f, err := os.OpenFile(pm.Path, os.O_RDWR, 0666)
    if err != nil {
        return nil, nil, fmt.Errorf("error al abrir disco: %v", err)
    }

    sb, err := CargarSuperBloque(f, pm.Partition.Part_start)
    if err != nil {
        f.Close()
        return nil, nil, err
    }
    return f, sb, nil
}

func unquoteValue_(s string) string {
    if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
        return s[1 : len(s)-1]
    }
    return s
}