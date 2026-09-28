package server

import (
    "backend/commands"
    "encoding/json"
    "log"
    "net/http"
    "strings"
)

type CommandRequest struct {
    Comando string `json:"comando"`
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }

    var req CommandRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Error de decodificación del cuerpo de la solicitud", http.StatusBadRequest)
        return
    }

    var fullOutput strings.Builder
    lines := strings.Split(req.Comando, "\n")
    for _, raw := range lines {
        line := strings.TrimRight(raw, "\r")
        trimmed := strings.TrimSpace(line)

        if trimmed == "" {
            fullOutput.WriteString("\n")
            continue
        }
        
        if strings.HasPrefix(trimmed, "#") {
            fullOutput.WriteString(line + "\n")
            continue
        }

        var command string
        var comment string
        
        if commentIndex := strings.Index(line, "#"); commentIndex != -1 {
            command = strings.TrimSpace(line[:commentIndex])
            comment = line[commentIndex:]
        } else {
            command = trimmed
        }

        if command != "" {
            output := commands.ExecuteCommand(command)
            if output != "" {
                fullOutput.WriteString(output + "\n")
            }
        }

        if comment != "" {
            fullOutput.WriteString(comment + "\n")
        }
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"salida": fullOutput.String()})
}

func IniciarAPIServer(port string) {
    http.HandleFunc("/execute", executeHandler)

    log.Println("Iniciando el servidor API en el puerto", port)
    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatalf("Error al iniciar el servidor: %v", err)
    }
}