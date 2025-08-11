package Utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Comparar compara dos strings ignorando mayúsculas/minúsculas
func Comparar(a, b string) bool {
	return strings.EqualFold(a, b)
}

// Error muestra un mensaje de error
func Error(comando, mensaje string) string {
	return fmt.Sprintf("❌ [%s] ERROR: %s", comando, mensaje)
}

// Mensaje muestra un mensaje informativo
func Mensaje(comando, mensaje string) string {
	return fmt.Sprintf("✅ [%s] %s", comando, mensaje)
}

// ArchivoExiste verifica si un archivo existe
func ArchivoExiste(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ParseCommand parsea un comando y extrae parámetros
func ParseCommand(command string) map[string]string {
	params := make(map[string]string)

	// Regex para capturar parámetros -param=value
	re := regexp.MustCompile(`-(\w+)=([^\s]+)`)
	matches := re.FindAllStringSubmatch(command, -1)

	for _, match := range matches {
		if len(match) == 3 {
			key := strings.ToLower(match[1])
			value := strings.Trim(match[2], "\"'") // Eliminar comillas
			params[key] = value
		}
	}

	return params
}

func SepararTokens(texto string) []string {
	var tokens []string
	if texto == "" {
		return tokens
	}
	texto += " "
	var token string
	estado := 0
	for i := 0; i < len(texto); i++ {
		c := string(texto[i])
		if estado == 0 && c == "-" {
			estado = 1
		} else if estado == 0 && c == "#" {
			continue
		} else if estado != 0 {
			if estado == 1 {
				if c == "=" {
					estado = 2
				} else if c == " " {
					continue
				}
			} else if estado == 2 {
				if c == " " {
					continue
				}
				if c == "\"" {
					estado = 3
					continue
				} else {
					estado = 4
				}
			} else if estado == 3 {
				if c == "\"" {
					estado = 4
					continue
				}
			} else if estado == 4 && c == "\"" {
				tokens = []string{}
				continue
			} else if estado == 4 && c == " " {
				estado = 0
				tokens = append(tokens, token)
				token = ""
				continue
			}
			token += c
		}
	}
	return tokens
}
