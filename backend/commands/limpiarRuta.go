package commands

import (
    "errors"
    "strings"
)

func limpiarRuta(ruta string) (string, error) {
    ruta = strings.TrimSpace(ruta)
    
    if len(ruta) == 0 {
        return "", errors.New("ruta vacía")
    }

    primerCar := ruta[0]
    ultimoCar := ruta[len(ruta)-1]

    // Verificar comillas balanceadas
    if primerCar == '"' && ultimoCar != '"' {
        return "", errors.New("comillas no balanceadas")
    }

    if primerCar != '"' && ultimoCar == '"' {
        return "", errors.New("comillas no balanceadas")
    }

    // Si tiene comillas balanceadas, removerlas
    if primerCar == '"' && ultimoCar == '"' {
        return ruta[1 : len(ruta)-1], nil
    }

    // Rechazar comillas simples
    if primerCar == '\'' || ultimoCar == '\'' {
        return "", errors.New("comillas simples no permitidas")
    }
    
    return ruta, nil
}