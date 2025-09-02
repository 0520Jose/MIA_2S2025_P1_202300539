package commands

import (
    "backend/structs"
    "encoding/binary"
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

    var pm *structs.PartitionMount
    for i := range structs.Particiones_Montadas {
        if structs.Particiones_Montadas[i].Id == id {
            pm = &structs.Particiones_Montadas[i]
            break
        }
    }
    if pm == nil {
        return fmt.Sprintf("Error: partición con ID %s no está montada", id)
    }

    f, err := os.Open(pm.Path)
    if err != nil {
        return fmt.Sprintf("Error al abrir el disco: %v", err)
    }
    defer f.Close()

    sb, err := CargarSuperBloque(f, pm.Partition.Part_start)
    if err != nil {
        return fmt.Sprintf("Error al cargar el superbloque: %v", err)
    }

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

func CargarSuperBloque(f *os.File, partStart int32) (*structs.SuperBloque, error) {
    if _, err := f.Seek(int64(partStart), 0); err != nil {
        return nil, fmt.Errorf("error al buscar superbloque: %v", err)
    }
    var sb structs.SuperBloque
    if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
        return nil, fmt.Errorf("error al leer superbloque: %v", err)
    }
    if sb.S_magic != 0xEF53 {
        return nil, fmt.Errorf("sistema de archivos no válido (magic: 0x%X)", sb.S_magic)
    }
    return &sb, nil
}



func firstNonEmpty(a, b string) string {
    if strings.TrimSpace(a) != "" {
        return a
    }
    return b
}

func unquoteValue(s string) string {
    v := strings.TrimSpace(s)
    if len(v) >= 2 && ((strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
        (strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'"))) {
        return v[1 : len(v)-1]
    }
    return v
}
