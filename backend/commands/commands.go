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
        return Mkdisk(args)
    case "fdisk":
        return Fdisk(args)
    case "mount":
        return Mount(args)
    case "rmdisk":
        return Rmdisk(args)
    case "mounted":
        return Mounted()
    case "mkfs":
        return Mkfs(args)
    case "login":
        return Login(args)
    case "logout":
        return Logout()
    case "cat":
        return Cat(args)
    case "mkgrp":
        return Mkgrp(args)
    default:
        return ""
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
