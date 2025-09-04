package commands

import "strings"

func ExecuteCommand(fullCommand string) string {
    tokens := splitFields(fullCommand)
    if len(tokens) == 0 {
        return "Comando vacío"
    }
    command := strings.ToLower(tokens[0])
    args := parseArgs(tokens[1:])

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
    case "rmgrp":
        return Rmgrp(args)
    case "mkusr":
        return Mkusr(args)
    case "rmusr":
        return Rmusr(args)
    case "chgrp":
        return Chgrp(args)
    case "mkfile":
        return Mkfile(args)
    case "mkdir":
        return Mkdir(args)
    case "rep":
        return Rep(args)
    default:
        return "Comando no reconocido"
    }
}

func parseArgs(args []string) map[string]string {
    params := make(map[string]string)
    for _, arg := range args {
        if strings.Contains(arg, "=") {
            kv := strings.SplitN(arg, "=", 2)
            key := strings.ToLower(kv[0])
            val := unquoteValue(kv[1])
            params[key] = val
        } else {
            params[strings.ToLower(arg)] = ""
        }
    }
    return params
}

func unquoteValue(s string) string {
    v := strings.TrimSpace(s)
    if len(v) >= 2 && ((strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
        (strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'"))) {
        return v[1 : len(v)-1]
    }
    return v
}

func splitFields(s string) []string {
    var out []string
    var buf strings.Builder
    inQuotes := false
    esc := false
    for _, r := range s {
        switch {
        case esc:
            buf.WriteRune(r)
            esc = false
        case r == '\\':
            esc = true
        case r == '"':
            inQuotes = !inQuotes
        case r == ' ' || r == '\t' || r == '\n':
            if inQuotes {
                buf.WriteRune(r)
            } else {
                if buf.Len() > 0 {
                    out = append(out, buf.String())
                    buf.Reset()
                }
            }
        default:
            buf.WriteRune(r)
        }
    }
    if buf.Len() > 0 {
        out = append(out, buf.String())
    }
    return out
}