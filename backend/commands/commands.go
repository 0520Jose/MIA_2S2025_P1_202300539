package commands

import (
    "strings"
)

func ExecuteCommand(fullCommand string) string {
    parts := strings.Fields(fullCommand)
    if len(parts) == 0 {
        return "Comando vacío"
    }

    command := parts[0]
    args := parseArgs(parts[1:])

    switch command {
    case "mkdisk":
        Mkdisk(args)
        return "Comando mkdisk ejecutado."
    case "fdisk":
        Fdisk(args)
        return "Comando fdisk ejecutado."
    case "mount":
        Mount(args)
        return "Comando mount ejecutado."
    case "rmdisk":
        Rmdisk(args)
        return "Comando rmdisk ejecutado."
    case "mounted":
        Mounted()
        return "Comando mounted ejecutado."
    default:
        return "Comando no reconocido: " + command
    }
}

func parseArgs(args []string) map[string]string {
    params := make(map[string]string)
    for _, arg := range args {
        if strings.Contains(arg, "=") {
            keyVal := strings.SplitN(arg, "=", 2)
            params[keyVal[0]] = strings.Trim(keyVal[1], "\"")
        }
    }
    return params
}