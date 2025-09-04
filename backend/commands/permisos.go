package commands

import "backend/structs"

func EsRoot() bool {
    return usuarioActual != nil && usuarioActual.Username == "root"
}

func CategoriaInodo(ino *structs.Inodo) int {
    if usuarioActual == nil {
        return 2
    }
    if int(ino.I_uid) == usuarioActual.UID {
        return 0
    }
    if int(ino.I_gid) == usuarioActual.GID {
        return 1
    }
    return 2
}

func Permisos(ino *structs.Inodo, need int) bool {
    if EsRoot() {
        return true
    }
    var d byte
    switch CategoriaInodo(ino) {
    case 0:
        d = ino.I_perm[0]
    case 1:
        d = ino.I_perm[1]
    default:
        d = ino.I_perm[2]
    }
    val := int(d - '0')
    return (val & need) == need
}