package commands

import (
    "strings"
    "backend/structs"
    "os"
    "fmt"
    "encoding/binary"
)

var currentUser *UserSession

type UserSession struct {
    Username    string
    PartitionID string
    Group       string
}

func Login(args map[string]string) string {
    if currentUser != nil {
        return "Error: ya hay un usuario logueado. Debe hacer logout primero."
    }

    user, okUser := args["-user"]
    pass, okPass := args["-pass"]
    id, okID := args["-id"]

    if !okUser || !okPass || !okID {
        return "Error: parámetros faltantes. Uso: login -user= -pass= -id="
    }

    // Validaciones adicionales
    if strings.TrimSpace(user) == "" {
        return "Error: el nombre de usuario no puede estar vacío"
    }
    if strings.TrimSpace(pass) == "" {
        return "Error: la contraseña no puede estar vacía"
    }
    if strings.TrimSpace(id) == "" {
        return "Error: el ID de partición no puede estar vacío"
    }

    fmt.Printf("[DEBUG] Intentando login para usuario: %s en partición: %s\n", user, id)

    contenido := LeerArchivoUsersTXT(id)
    if strings.HasPrefix(contenido, "Error:") {
        return contenido
    }

    if strings.TrimSpace(contenido) == "" {
        return "Error: el archivo users.txt está vacío o corrupto"
    }

    var usersTXT structs.UsersTXT
    usersTXT.FromString(contenido)

    usr := usersTXT.GetUsuario(user)
    if usr == nil {
        return fmt.Sprintf("Error: el usuario '%s' no existe.", user)
    }
    
    if usr.Contrasena != pass {
        return "Error: autenticación fallida. Contraseña incorrecta."
    }

    currentUser = &UserSession{
        Username:    usr.Nombre,
        PartitionID: id,
        Group:       usr.Grupo,
    }

    fmt.Printf("[INFO] Usuario %s logueado exitosamente en partición %s\n", usr.Nombre, id)
    return fmt.Sprintf("Inicio de sesión exitoso como: %s (Grupo: %s)", usr.Nombre, usr.Grupo)
}

func Logout() string {
    if currentUser == nil {
        return "Error: no hay ningún usuario logueado."
    }
    
    username := currentUser.Username
    fmt.Printf("[INFO] Logout para usuario: %s\n", username)
    currentUser = nil
    return "Logout exitoso para el usuario: " + username
}

func GetCurrentUser() *UserSession {
    return currentUser
}

func LeerArchivoUsersTXT(id string) string {
    fmt.Printf("[DEBUG] Buscando partición montada con ID: %s\n", id)
    
    for _, pm := range structs.Particiones_Montadas {
        if pm.Id == id {
            fmt.Printf("[DEBUG] Partición encontrada: %s\n", pm.Path)
            path := pm.Path
            partition := pm.Partition

            fs := CargarSistemaEXT2(path, partition)
            if fs == nil {
                return "Error: no se pudo cargar el sistema de archivos"
            }
            defer fs.File.Close()

            contenido, err := fs.LeerArchivoUsersTXT()
            if err != nil {
                return "Error: no se pudo leer el archivo users.txt -> " + err.Error()
            }

            fmt.Printf("[DEBUG] users.txt leído exitosamente (%d bytes)\n", len(contenido))
            return contenido
        }
    }
    return "Error: partición con id " + id + " no está montada"
}

func CargarSistemaEXT2(path string, partition structs.Partition) *EXT2FileSystem {
    fmt.Printf("[DEBUG] Cargando sistema EXT2 desde: %s, start: %d\n", path, partition.Part_start)
    
    file, err := os.OpenFile(path, os.O_RDWR, 0666)
    if err != nil {
        fmt.Printf("[ERROR] Error al abrir el archivo de disco: %v\n", err)
        return nil
    }

    fs := &EXT2FileSystem{
        File:      file,
        Path:      path,
        Partition: partition,
    }

    err = fs.LoadSuperBlock()
    if err != nil {
        fmt.Printf("[ERROR] Error al cargar el superbloque: %v\n", err)
        file.Close()
        return nil
    }

    fmt.Printf("[INFO] Sistema EXT2 cargado exitosamente (magic: 0x%X)\n", fs.SuperBlock.S_magic)
    return fs
}

type EXT2FileSystem struct {
    File       *os.File
    Path       string
    Partition  structs.Partition
    SuperBlock structs.SuperBloque
}

func (fs *EXT2FileSystem) LoadSuperBlock() error {
    fmt.Printf("[DEBUG] Cargando superbloque desde posición: %d\n", fs.Partition.Part_start)
    
    // Verificar que el archivo esté abierto correctamente
    stat, err := fs.File.Stat()
    if err != nil {
        return fmt.Errorf("error al obtener información del archivo: %v", err)
    }
    fmt.Printf("[DEBUG] Tamaño del archivo: %d bytes\n", stat.Size())
    
    // Verificar que la posición esté dentro del archivo
    if int64(fs.Partition.Part_start) >= stat.Size() {
        return fmt.Errorf("posición del superbloque (%d) está fuera del archivo (tamaño: %d)", 
            fs.Partition.Part_start, stat.Size())
    }
    
    _, err = fs.File.Seek(int64(fs.Partition.Part_start), 0)
    if err != nil {
        return fmt.Errorf("error al posicionarse en superbloque: %v", err)
    }
    
    // Verificar posición actual
    currentPos, _ := fs.File.Seek(0, 1)
    fmt.Printf("[DEBUG] Posición actual antes de leer: %d\n", currentPos)
    
    // Leer más bytes para debug
    fs.File.Seek(int64(fs.Partition.Part_start), 0)
    debugBytes := make([]byte, 64) // Leer más bytes para debug
    n, err := fs.File.Read(debugBytes)
    if err != nil {
        return fmt.Errorf("error al leer bytes de debug: %v", err)
    }
    fmt.Printf("[DEBUG] Primeros %d bytes del superbloque: %v\n", n, debugBytes[:n])
    
    // Regresar a la posición original y leer el superbloque
    _, err = fs.File.Seek(int64(fs.Partition.Part_start), 0)
    if err != nil {
        return fmt.Errorf("error al reposicionarse: %v", err)
    }
    
    err = binary.Read(fs.File, binary.LittleEndian, &fs.SuperBlock)
    if err != nil {
        return fmt.Errorf("error al leer superbloque: %v", err)
    }
    
    fmt.Printf("[DEBUG] Magic number leído: 0x%X (esperado: 0xEF53)\n", fs.SuperBlock.S_magic)
    fmt.Printf("[DEBUG] Otros campos del superbloque:\n")
    fmt.Printf("        S_inodes_count: %d\n", fs.SuperBlock.S_inodes_count)
    fmt.Printf("        S_blocks_count: %d\n", fs.SuperBlock.S_blocks_count)
    fmt.Printf("        S_inode_start: %d\n", fs.SuperBlock.S_inode_start)
    fmt.Printf("        S_block_start: %d\n", fs.SuperBlock.S_block_start)
    
    if fs.SuperBlock.S_magic != 0xEF53 {
        return fmt.Errorf("sistema de archivos no válido (magic: 0x%X). La partición no está formateada o hay un error en mkfs", fs.SuperBlock.S_magic)
    }
    return nil
}

func (fs *EXT2FileSystem) LeerArchivoUsersTXT() (string, error) {
    fmt.Printf("[DEBUG] Leyendo archivo users.txt (inodo 0)\n")
    
    inodo, err := fs.LeerInodo(0)
    if err != nil {
        return "", fmt.Errorf("error al leer inodo de users.txt: %v", err)
    }
    
    fmt.Printf("[DEBUG] Inodo users.txt - tipo: %d, tamaño: %d, bloque[0]: %d\n", 
        inodo.I_type, inodo.I_s, inodo.I_block[0])
    
    if inodo.I_type != [1]byte{1} {
        return "", fmt.Errorf("users.txt no es un archivo válido (tipo: %v)", inodo.I_type)
    }
    if inodo.I_block[0] == -1 {
        return "", fmt.Errorf("users.txt no tiene bloques asignados")
    }
    
    bloque, err := fs.LeerBloqueArchivo(inodo.I_block[0])
    if err != nil {
        return "", fmt.Errorf("error al leer bloque de users.txt: %v", err)
    }
    
    contenido := string(bloque.B_content[:inodo.I_s])
    fmt.Printf("[DEBUG] Contenido users.txt leído: %q\n", contenido)
    return contenido, nil
}

func (fs *EXT2FileSystem) LeerInodo(numero int32) (structs.Inodo, error) {
    var inodo structs.Inodo
    posicionInodo := int64(fs.SuperBlock.S_inode_start) + int64(numero)*int64(fs.SuperBlock.S_inode_s)
    
    fmt.Printf("[DEBUG] Leyendo inodo %d desde posición: %d\n", numero, posicionInodo)
    
    _, err := fs.File.Seek(posicionInodo, 0)
    if err != nil {
        return inodo, err
    }
    err = binary.Read(fs.File, binary.LittleEndian, &inodo)
    return inodo, err
}

func (fs *EXT2FileSystem) LeerBloqueArchivo(numero int32) (structs.BArchivo, error) {
    var bloque structs.BArchivo
    posicionBloque := int64(fs.SuperBlock.S_block_start) + int64(numero)*int64(fs.SuperBlock.S_block_s)
    
    fmt.Printf("[DEBUG] Leyendo bloque %d desde posición: %d\n", numero, posicionBloque)
    
    _, err := fs.File.Seek(posicionBloque, 0)
    if err != nil {
        return bloque, err
    }
    err = binary.Read(fs.File, binary.LittleEndian, &bloque)
    return bloque, err
}

func (fs *EXT2FileSystem) BuscarInodoPorNombre(nombre string) (structs.Inodo, int32, error) {
    raiz, err := fs.LeerInodo(1)
    if err != nil {
        return structs.Inodo{}, -1, fmt.Errorf("error al leer inodo raíz: %v", err)
    }

    for _, blk := range raiz.I_block {
        if blk == -1 {
            continue
        }
        var bloque structs.BCarpeta
        pos := int64(fs.SuperBlock.S_block_start) + int64(blk)*int64(fs.SuperBlock.S_block_s)
        fs.File.Seek(pos, 0)
        err := binary.Read(fs.File, binary.LittleEndian, &bloque)
        if err != nil {
            continue
        }
        for _, content := range bloque.B_content {
            n := strings.TrimRight(string(content.B_name[:]), "\x00")
            if n == nombre {
                inodo, err := fs.LeerInodo(content.B_inodo)
                return inodo, content.B_inodo, err
            }
        }
    }
    return structs.Inodo{}, -1, fmt.Errorf("archivo %s no encontrado", nombre)
}

func (fs *EXT2FileSystem) EscribirInodo(num int32, inodo structs.Inodo) error {
    pos := int64(fs.SuperBlock.S_inode_start) + int64(num)*int64(fs.SuperBlock.S_inode_s)
    _, err := fs.File.Seek(pos, 0)
    if err != nil {
        return err
    }
    return binary.Write(fs.File, binary.LittleEndian, &inodo)
}

func (fs *EXT2FileSystem) EscribirBloqueArchivo(num int32, bloque structs.BArchivo) error {
    pos := int64(fs.SuperBlock.S_block_start) + int64(num)*int64(fs.SuperBlock.S_block_s)
    _, err := fs.File.Seek(pos, 0)
    if err != nil {
        return err
    }
    return binary.Write(fs.File, binary.LittleEndian, &bloque)
}