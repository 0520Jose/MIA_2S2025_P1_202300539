package parser

import (
    "os"
    "strings"
    "strconv"
    "backend/utils"
    "backend/structs"
    "errors"
)

var usuarioActual *structs.Usuario

func LeerUsersTxt(ruta string) ([]structs.Grupo, []structs.Usuario, error) {
    data, err := os.ReadFile(ruta)
    if err != nil {
        return nil, nil, err
    }

    var grupos []structs.Grupo
    var usuarios []structs.Usuario

    lineas := strings.Split(string(data), "\n")
    for _, linea := range lineas {
        partes := utils.SplitTrim(linea, ",")
        if len(partes) == 0 {
            continue
        }

        if len(partes) == 3 && partes[1] == "G" {
            id, _ := strconv.Atoi(partes[0])
            if id == 0 { continue }
            grupos = append(grupos, structs.Grupo{GID: id, Nombre: partes[2]})
        }

        if len(partes) == 5 && partes[1] == "U" {
            id, _ := strconv.Atoi(partes[0])
            if id == 0 { continue }
            usuarios = append(usuarios, structs.Usuario{
                UID: id, Grupo: partes[2], Nombre: partes[3], Contrasena: partes[4],
            })
        }
    }
    return grupos, usuarios, nil
}

func LeerArchivo(partitionID, filePath string) (string, error) {
    encontrado := false

    for _, p := range structs.Particiones_Montadas {
        if p.Id == partitionID {
            encontrado = true
            break
        }
    }

    if !encontrado {
        return "", errors.New("partición no montada")
    }

    if usuarioActual == nil {
        return "", errors.New("No hay usuario logueado")
    }

    if strings.TrimPrefix(filePath, "/") == "users.txt" {
        return "1,G,root\n1,U,root,root,123\n", nil
    }

    return "", errors.New("archivo no encontrado")
}

func SetCurrentUser(user *structs.Usuario) {
    usuarioActual = user
}

func GetCurrentUser() *structs.Usuario {
    return usuarioActual
}

func ClearCurrentUser() {
    usuarioActual = nil
}