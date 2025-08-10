package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
)

// ===== ESTRUCTURAS DE DATOS =====

type CommandRequest struct {
    Commands string `json:"commands"`
}

type CommandResponse struct {
    Output  string `json:"output"`
    Success bool   `json:"success"`
    Message string `json:"message"`
    Error   string `json:"error,omitempty"`
}

type ErrorResponse struct {
    Error   string `json:"error"`
    Code    int    `json:"code"`
    Message string `json:"message"`
}

// ===== FUNCIONES PRINCIPALES =====

func enableCORS(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
    enableCORS(w)
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    if r.Method != "POST" {
        errorResponse := ErrorResponse{
            Error:   "Método no permitido",
            Code:    405,
            Message: "Solo se permiten peticiones POST",
        }
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(errorResponse)
        return
    }

    var req CommandRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errorResponse := ErrorResponse{
            Error:   "Error al decodificar JSON",
            Code:    400,
            Message: err.Error(),
        }
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(errorResponse)
        return
    }

    // Procesar comandos
    output := processCommands(req.Commands)
    
    response := CommandResponse{
        Output:  output,
        Success: true,
        Message: "Comandos procesados exitosamente",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func processCommands(commands string) string {
    lines := strings.Split(commands, "\n")
    var output strings.Builder
    
    output.WriteString("=== INICIANDO PROCESAMIENTO DE COMANDOS ===\n\n")
    
    for i, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        output.WriteString(fmt.Sprintf("[%d] Ejecutando: %s\n", i+1, line))
        
        result := executeCommand(line)
        output.WriteString(fmt.Sprintf("    → %s\n\n", result))
    }
    
    output.WriteString("=== PROCESAMIENTO COMPLETADO ===")
    return output.String()
}

func executeCommand(command string) string {
    parts := strings.Fields(command)
    if len(parts) == 0 {
        return "❌ Comando vacío"
    }
    
    cmd := strings.ToLower(parts[0])
    
    switch cmd {
    case "mkdisk":
        return "✅ Disco creado exitosamente (simulación)"
    case "rmdisk":
        return "✅ Disco eliminado exitosamente (simulación)"
    case "fdisk":
        return "✅ Partición creada exitosamente (simulación)"
    case "mount":
        return "✅ Sistema de archivos montado (simulación)"
    case "unmount":
        return "✅ Sistema de archivos desmontado (simulación)"
    case "mkfs":
        return "✅ Sistema de archivos creado (simulación)"
    case "login":
        return "✅ Usuario autenticado (simulación)"
    case "logout":
        return "✅ Sesión cerrada (simulación)"
    case "mkfile":
        return "✅ Archivo creado exitosamente (simulación)"
    case "mkdir":
        return "✅ Directorio creado exitosamente (simulación)"
    case "cat":
        return "✅ Mostrando contenido del archivo (simulación)"
    case "remove":
        return "✅ Elemento eliminado (simulación)"
    case "edit":
        return "✅ Archivo editado (simulación)"
    case "rename":
        return "✅ Elemento renombrado (simulación)"
    case "copy":
        return "✅ Elemento copiado (simulación)"
    case "move":
        return "✅ Elemento movido (simulación)"
    case "find":
        return "✅ Búsqueda completada (simulación)"
    case "chmod":
        return "✅ Permisos cambiados (simulación)"
    case "chown":
        return "✅ Propietario cambiado (simulación)"
    case "chgrp":
        return "✅ Grupo cambiado (simulación)"
    default:
        return fmt.Sprintf("⚠️ Comando no reconocido: %s", cmd)
    }
}

func main() {
    http.HandleFunc("/api/execute", executeHandler)
    
    fmt.Println("🚀 Servidor Go iniciado en http://localhost:8080")
    fmt.Println("📡 Esperando conexiones del frontend React...")
    fmt.Println("📁 Listo para procesar comandos EXT2")
    fmt.Println("==================================================")
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}