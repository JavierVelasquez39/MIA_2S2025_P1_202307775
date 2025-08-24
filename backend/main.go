package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"godisk-backend/Comandos"
	"godisk-backend/Utils"
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

// processCommands ahora preserva comentarios y líneas en blanco tal como aparecen.
// Numera únicamente los comandos ejecutables (ignorando comentarios y líneas vacías).
func processCommands(commands string) string {
	lines := strings.Split(commands, "\n")
	var output strings.Builder

	output.WriteString("=== INICIANDO PROCESAMIENTO DE COMANDOS ===\n\n")

	cmdIndex := 0
	for _, raw := range lines {
		// preservar exactamente la representación del archivo:
		// quitar sólo el retorno CR si existe, mantener espacios y tabs.
		line := strings.TrimRight(raw, "\r")

		// líneas en blanco -> preservar
		if strings.TrimSpace(line) == "" {
			output.WriteString("\n")
			continue
		}

		// comentarios (líneas que después de quitar espacios a la izquierda empiezan con '#')
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			// mostrar tal como aparece en el archivo (sin modificar)
			output.WriteString(line + "\n")
			continue
		}

		// comando ejecutable
		cmdIndex++
		output.WriteString(fmt.Sprintf("[%d] Ejecutando: %s\n", cmdIndex, strings.TrimSpace(line)))
		result := executeCommand(strings.TrimSpace(line))
		output.WriteString(fmt.Sprintf("    → %s\n\n", result))
	}

	output.WriteString("=== PROCESAMIENTO COMPLETADO ===")
	return output.String()
}

// Nuevo handler para subir/ejecutar scripts (.smia).
// Soporta multipart/form-data (campo "script") o JSON { "commands": "..." }.
func executeScriptHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		errorResponse := ErrorResponse{
			Error:   "Método no permitido",
			Code:    405,
			Message: "Solo se permiten peticiones POST",
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	contentType := r.Header.Get("Content-Type")
	var commands string

	if strings.HasPrefix(contentType, "multipart/") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			errorResponse := ErrorResponse{
				Error:   "Error al parsear multipart",
				Code:    400,
				Message: err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
		file, fh, err := r.FormFile("script")
		if err != nil {
			errorResponse := ErrorResponse{
				Error:   "Campo 'script' no encontrado",
				Code:    400,
				Message: err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
		defer file.Close()

		// Validar extensión .smia
		if !strings.HasSuffix(strings.ToLower(fh.Filename), ".smia") {
			errorResponse := ErrorResponse{
				Error:   "Extensión inválida",
				Code:    400,
				Message: "Se requiere extensión .smia",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
		b, err := io.ReadAll(file)
		if err != nil {
			errorResponse := ErrorResponse{
				Error:   "Error al leer archivo",
				Code:    500,
				Message: err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
		commands = string(b)
	} else {
		// JSON body { "commands": "..." }
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
		commands = req.Commands
	}

	out := processCommands(commands)

	resp := CommandResponse{
		Output:  out,
		Success: true,
		Message: "Script procesado",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func executeCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "❌ Comando vacío"
	}

	cmd := strings.ToUpper(parts[0])

	// Extraer tokens del comando
	commandLine := strings.TrimSpace(strings.TrimPrefix(command, parts[0]))
	tokens := Utils.SepararTokens(commandLine)

	// DEBUG: mostrar commandLine y tokens para diagnosticar parseo de flags
	fmt.Printf("🔧 DEBUG: commandLine='%s' -> tokens=%v\n", commandLine, tokens)

	// Si el comando original contiene "-p" pero tokens no, añadirlo (evita que se pierda el flag)
	if strings.Contains(command, " -p") || strings.HasPrefix(command, "-p") || strings.Contains(command, "-p ") {
		hasP := false
		for _, t := range tokens {
			if t == "-p" || t == "p" {
				hasP = true
				break
			}
		}
		if !hasP {
			tokens = append(tokens, "-p")
			fmt.Printf("🔧 DEBUG: auto-added token '-p' -> tokens=%v\n", tokens)
		}
	}

	switch cmd {
	case "MKDISK":
		return Comandos.ValidarDatosMKDISK(tokens)
	case "RMDISK":
		return Comandos.RMDISK(tokens)
	case "FDISK":
		return Comandos.ValidarDatosFDISK(tokens)
	case "MOUNT":
		return Comandos.ValidarDatosMOUNT(tokens)
	case "MOUNTED":
		return Comandos.ValidarDatosMOUNTED(tokens)
	case "MKFS":
		return Comandos.ValidarDatosMKFS(tokens)
	case "REP":
		return Comandos.ValidarDatosREP(tokens)
	case "CAT":
		return Comandos.ValidarDatosCAT(tokens)
	case "LOGIN":
		return Comandos.ValidarDatosLOGIN(tokens)
	case "LOGOUT":
		return Comandos.ValidarDatosLOGOUT(tokens)
	case "MKGRP":
		return Comandos.ValidarDatosMKGRP(tokens)
	case "RMGRP":
		return Comandos.ValidarDatosRMGRP(tokens)
	case "MKUSR":
		return Comandos.ValidarDatosMKUSR(tokens)
	case "RMUSR":
		return Comandos.ValidarDatosRMUSR(tokens)
	case "CHGRP":
		return Comandos.ValidarDatosCHGRP(tokens)
	case "MKFILE":
		return Comandos.ValidarDatosMKFILE(tokens)
	case "MKDIR":
		return Comandos.ValidarDatosMKDIR(tokens)
	default:
		return fmt.Sprintf("⚠️ Comando no reconocido: %s", cmd)
	}
}

func main() {
	http.HandleFunc("/api/execute", executeHandler)
	http.HandleFunc("/api/exec-script", executeScriptHandler)

	fmt.Println("🚀 Servidor Go iniciado en http://localhost:8080")
	fmt.Println("📡 Esperando conexiones del frontend React...")
	fmt.Println("📁 Listo para procesar comandos EXT2")
	fmt.Println("==================================================")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
