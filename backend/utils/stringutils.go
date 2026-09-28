package utils

import (
	"strconv"
	"strings"
)

func SplitTrim(s, sep string) []string {
	raw := strings.Split(s, sep)
	var out []string
	for _, v := range raw {
		trim := strings.TrimSpace(v)
		if trim != "" {
			out = append(out, trim)
		}
	}
	return out
}


func ParseID(idStr string) int {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1
	}
	return id
}


func CreateFile(name string) error {
	dir := filepath.Dir(name)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Println("Error creando directorio: ", err)
		return err
	}
	if _, err := os.Stat(name); os.IsNotExist(err) {
		file, err := os.Create(name)
		if err != nil {
			fmt.Println("Error creando archivo:", err)
			return err
		}
		defer file.Close()
	}
	return nil
}

func OpenFile(name string) (*os.File, error) {
	file, err := os.OpenFile(name, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error abriendo archivo:", err)
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
		fmt.Println("Error leyendo el objeto del archivo binario", err)
		return err
	}
	return nil
}