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
        http.Error(w, "Error decoding request body", http.StatusBadRequest)
        return
    }
    lines := strings.Split(req.Comando, "\n")
    var fullOutput strings.Builder
    for _, line := range lines {
        trimmedLine := strings.TrimSpace(line)
        if trimmedLine != "" {
            output := commands.ExecuteCommand(trimmedLine)
            fullOutput.WriteString(output + "\n")
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"salida": strings.TrimSpace(fullOutput.String())})
}

func StartAPIServer(port string) {
    http.HandleFunc("/execute", executeHandler)

    log.Println("Starting API server on port", port)
    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}