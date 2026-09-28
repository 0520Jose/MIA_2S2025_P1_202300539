package Utilities

import (
    "encoding/binary"
    "fmt"
    "os"
    "path/filepath"
)

func CreateFile(name string) error {
    dir := filepath.Dir(name)
    if err := os.MkdirAll(dir, os.ModePerm); err != nil {
        fmt.Println("Error creando directorio: ", err)
        return err
    }
    if _, err := os.Stat(name); os.IsNotExist(err) {
        file, err := os.Create(name)
        if err != nil {
            fmt.Println("Error creating file:", err)
            return err
        }
        defer file.Close()
    }
    return nil
}

func OpenFile(name string) (*os.File, error) {
    file, err := os.OpenFile(name, os.O_RDWR, 0644)
    if err != nil {
        fmt.Println("Error open file:", err)
        return nil, err
    }
    return file, nil
}

func WriteObject(file *os.File, data interface{}, position int64) error {
    file.Seek(position, 0)
    err := binary.Write(file, binary.LittleEndian, data)
    if err != nil {
        fmt.Println("Error escribiendo el archivo:", err)
        return err
    }
    return nil
}

func ReadObject(file *os.File, data interface{}, position int64) error {
    file.Seek(position, 0)
    err := binary.Read(file, binary.LittleEndian, data)
    if err != nil {
        fmt.Println("Error leyendo el archivo:", err)
        return err
    }
    return nil
}