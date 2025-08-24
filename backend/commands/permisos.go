package commands

import "backend/structs"

const (
    permRead  = 4
    permWrite = 2
    permExec  = 1
)

func isRoot() bool {
    return currentUser != nil && currentUser.Username == "root"
}

func inodeCategory(ino *structs.Inodo) int {
    if currentUser == nil {
        return 2
    }
    if int(ino.I_uid) == currentUser.UID {
        return 0
    }
    if int(ino.I_gid) == currentUser.GID {
        return 1
    }
    return 2
}

func hasPerm(ino *structs.Inodo, need int) bool {
    if isRoot() {
        return true
    }
    var d byte
    switch inodeCategory(ino) {
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