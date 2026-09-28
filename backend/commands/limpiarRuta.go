package commands

import (
    "errors"
    "strings"
)

func LimpiarRuta(ruta string) (string, error) {
    ruta = strings.TrimSpace(ruta)

    if len(ruta) == 0 {
        return "", errors.New("ruta vacía")
    }

    primerCar := ruta[0]
    ultimoCar := ruta[len(ruta)-1]

    if primerCar == '"' && ultimoCar != '"' {
        return "", errors.New("comillas no balanceadas")
    }
    if primerCar != '"' && ultimoCar == '"' {
        return "", errors.New("comillas no balanceadas")
    }

    if primerCar == '"' && ultimoCar == '"' {
        inner := strings.TrimSpace(ruta[1 : len(ruta)-1])
        if inner == "" {
            return "", errors.New("ruta vacía")
        }
        return inner, nil
    }

    if primerCar == '\'' || ultimoCar == '\'' {
        return "", errors.New("comillas simples no permitidas")
    }

    return ruta, nil
}