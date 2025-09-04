package commands

import (
    "backend/structs"
    "fmt"
    "os"
    "strconv"
    "strings"
)

var usuarioActual *UserSession

type UserSession struct {
    Username    string
    PartitionID string
    Group       string
    UID         int
    GID         int
}

func Login(params map[string]string) string {
    if usuarioActual != nil {
        return "Error: ya hay un usuario logueado"
    }

    allowed := map[string]struct{}{
        "-user": {}, "-usr": {}, "-pass": {}, "-pwd": {}, "-id": {},
    }
    normalized := make(map[string]string, len(params))
    for k, v := range params {
        lk := strings.ToLower(strings.TrimSpace(k))
        if _, ok := allowed[lk]; !ok {
            return fmt.Sprintf("Error: parámetro no reconocido: %s", k)
        }
        normalized[lk] = v
    }

    user := unquoteValue(firstNonEmpty(normalized["-user"], normalized["-usr"]))
    pass := unquoteValue(firstNonEmpty(normalized["-pass"], normalized["-pwd"]))
    id := unquoteValue(normalized["-id"])
    if strings.TrimSpace(user) == "" || strings.TrimSpace(pass) == "" || strings.TrimSpace(id) == "" {
        return "Error: parámetros -user/-usr, -pass/-pwd e -id son obligatorios"
    }

    f, sb, err := CargarSistemaEXT2(id)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer f.Close()

    contenido, err := LeerArchivoUsersTXT(f, sb)
    if err != nil {
        return fmt.Sprintf("Error: no se pudo leer el archivo users.txt -> %v", err)
    }

    lineas := strings.Split(strings.TrimSpace(contenido), "\n")

    grupos := make(map[string]string)
    for _, linea := range lineas {
        linea = strings.TrimSpace(linea)
        if linea == "" || strings.HasPrefix(linea, "#") {
            continue
        }
        partes := strings.Split(linea, ",")
        if len(partes) >= 3 && partes[1] == "G" {
            grupos[partes[0]] = partes[2]
        }
    }

    var uidInt, gidInt int
    var grupoNombre string
    encontrado := false

    for _, linea := range lineas {
        partes := strings.Split(strings.TrimSpace(linea), ",")
        if len(partes) >= 5 && partes[1] == "U" {
            uid := partes[0]
            grupoID := partes[2]
            nombreUsuario := partes[3]
            password := partes[4]

            if nombreUsuario == user && password == pass {
                uidInt, _ = strconv.Atoi(uid)
                gidInt, _ = strconv.Atoi(grupoID)
                grupoNombre = grupos[grupoID]
                if grupoNombre == "" {
                    grupoNombre = "unknown"
                }
                encontrado = true
                break
            }
        }
    }

    if !encontrado {
        return "Error: usuario o contraseña incorrectos"
    }

    usuarioActual = &UserSession{
        Username:    user,
        PartitionID: id,
        Group:       grupoNombre,
        UID:         uidInt,
        GID:         gidInt,
    }
    return "Login exitoso"
}

func Logout() string {
    if usuarioActual == nil {
        return "Error: no hay ningún usuario logueado"
    }
    usuarioActual = nil
    return "Logout exitoso"
}

func GetCurrentUser() *UserSession {
    return usuarioActual
}

func LeerArchivoUsersTXT(f *os.File, sb *structs.SuperBloque) (string, error) {
    ino, err := readInode(f, sb, 2)
    if err != nil {
        return "", err
    }
    
    if ino.I_type[0] != 1 {
        return "", fmt.Errorf("users.txt no es un archivo")
    }
    
    var contenido []byte
    remaining := int(ino.I_s)
    
    for i := 0; i < DIRECT_BLOCKS && remaining > 0; i++ {
        if ino.I_block[i] == -1 {
            break
        }
        
        bloque, err := readFileBlock(f, sb, ino.I_block[i])
        if err != nil {
            return "", err
        }
        
        chunk := 64
        if chunk > remaining {
            chunk = remaining
        }
        contenido = append(contenido, bloque.B_content[:chunk]...)
        remaining -= chunk
    }
    
    if remaining > 0 && ino.I_block[INDIRECT_SIMPLE] != -1 {
        pointers, err := readPointerBlock(f, sb, ino.I_block[INDIRECT_SIMPLE])
        if err != nil {
            return "", err
        }
        
        for i := 0; i < 16 && remaining > 0; i++ {
            if pointers.B_pointers[i] == -1 {
                break
            }
            
            bloque, err := readFileBlock(f, sb, pointers.B_pointers[i])
            if err != nil {
                return "", err
            }
            
            chunk := 64
            if chunk > remaining {
                chunk = remaining
            }
            contenido = append(contenido, bloque.B_content[:chunk]...)
            remaining -= chunk
        }
    }
    
    return string(contenido), nil
}

func readFileBlock(f *os.File, sb *structs.SuperBloque, blockIdx int32) (structs.BArchivo, error) {
    var bloque structs.BArchivo
    offset := int64(sb.S_block_start) + int64(blockIdx)*int64(sb.S_block_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return bloque, err
    }
    err := structs.ReadBinaryStruct(f, &bloque)
    return bloque, err
}