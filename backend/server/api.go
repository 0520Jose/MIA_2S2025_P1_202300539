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
    for _, line := range strings.Split(req.Comando, "\n") {
        trimmedLine := strings.TrimSpace(line)
        if trimmedLine != "" {
            output := commands.ExecuteCommand(trimmedLine)
            if output != "" {
                fullOutput.WriteString(output + "\n")
            }
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"salida": strings.TrimSpace(fullOutput.String())})
}

func StartAPIServer(port string) {
    http.HandleFunc("/execute", executeHandler)

    log.Println("Iniciando el servidor API en el puerto", port)
    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatalf("Error al iniciar el servidor: %v", err)
    }
}