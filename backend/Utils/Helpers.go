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

// ============ FUNCIONES AUXILIARES PARA FDISK ============

// Funciones matemáticas para FDISK
func MayorQue(a, b int) bool {
	return a > b
}

func MenorQue(a, b int) bool {
	return a < b
}

func Suma(a, b int) int {
	return a + b
}

func Resta(a, b int) int {
	return a - b
}

// EscribirBytes escribe bytes a un archivo
func EscribirBytes(file *os.File, data []byte) error {
	_, err := file.Write(data)
	return err
}

// LeerBytes lee bytes de un archivo
func LeerBytes(file *os.File, size int) []byte {
	data := make([]byte, size)
	file.Read(data)
	return data
}

// ConvertirBytes convierte un tamaño según la unidad especificada
func ConvertirBytes(size int, unit string) int {
	switch strings.ToUpper(unit) {
	case "K":
		return size * 1024
	case "M":
		return size * 1024 * 1024
	case "B":
		return size
	default:
		return size
	}
}

// ValidarParametro verifica si un parámetro está en una lista de valores válidos
func ValidarParametro(valor string, valoresValidos []string) bool {
	for _, valido := range valoresValidos {
		if Comparar(valor, valido) {
			return true
		}
	}
	return false
}

// LimpiarPath elimina comillas del path
func LimpiarPath(path string) string {
	return strings.ReplaceAll(path, "\"", "")
}

// ConvertirAString convierte un array de bytes a string eliminando caracteres nulos
func ConvertirAString(bytes [16]byte) string {
	nombre := ""
	for _, b := range bytes {
		if b != 0 {
			nombre += string(b)
		} else {
			break
		}
	}
	return nombre
}

// CopiarString copia un string a un array de bytes de tamaño fijo
func CopiarString(destino []byte, origen string) {
	copy(destino, origen)
}

// CalcularEspacioLibre calcula el espacio libre entre particiones
func CalcularEspacioLibre(inicio, fin, tamanoDisco int) int {
	if fin > tamanoDisco {
		return 0
	}
	return tamanoDisco - fin
}

// ValidarTamaño verifica que el tamaño sea válido
func ValidarTamaño(size string) (int, error) {
	return 0, nil // Implementar validación específica si es necesaria
}

// CrearDirectorio crea un directorio si no existe
func CrearDirectorio(path string) error {
	dir := strings.Replace(path, "\\", "/", -1)
	lastSlash := strings.LastIndex(dir, "/")
	if lastSlash > 0 {
		dirPath := dir[:lastSlash]
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

// ValidarExtension verifica que el archivo tenga la extensión correcta
func ValidarExtension(path, extension string) bool {
	return strings.HasSuffix(strings.ToLower(path), strings.ToLower(extension))
}

// FormatearPath formatea el path para el sistema operativo
func FormatearPath(path string) string {
	// Reemplazar barras según el SO
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ReplaceAll(path, "\"", "")
	return path
}

// VerificarPermisos verifica si se tienen permisos de escritura en un directorio
func VerificarPermisos(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// ObtenerTamanoArchivo obtiene el tamaño de un archivo
func ObtenerTamanoArchivo(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// EsDirectorio verifica si una ruta es un directorio
func EsDirectorio(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// LimpiarNombre limpia un nombre de caracteres especiales
func LimpiarNombre(nombre string) string {
	// Eliminar caracteres no válidos para nombres de partición
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		nombre = strings.ReplaceAll(nombre, char, "")
	}
	return strings.TrimSpace(nombre)
}

// ValidarNombre verifica que el nombre sea válido
func ValidarNombre(nombre string) bool {
	if len(nombre) == 0 || len(nombre) > 16 {
		return false
	}
	// Verificar caracteres válidos
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(nombre, char) {
			return false
		}
	}
	return true
}

// RedondearTamaño redondea un tamaño al múltiplo más cercano
func RedondearTamaño(size, multiple int) int {
	return ((size + multiple - 1) / multiple) * multiple
}

// CalcularPorcentaje calcula el porcentaje de uso
func CalcularPorcentaje(usado, total int) float64 {
	if total == 0 {
		return 0
	}
	return (float64(usado) / float64(total)) * 100
}

// FormatearTamaño formatea un tamaño en bytes a una representación legible
func FormatearTamaño(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ConvertirAString20 convierte [20]byte a string
func ConvertirAString20(bytes [20]byte) string {
	resultado := ""
	for _, b := range bytes {
		if b != 0 {
			resultado += string(b)
		} else {
			break
		}
	}
	return resultado
}

// ConvertirAString150 convierte [150]byte a string
func ConvertirAString150(bytes [150]byte) string {
	resultado := ""
	for _, b := range bytes {
		if b != 0 {
			resultado += string(b)
		} else {
			break
		}
	}
	return resultado
}

// ConvertirAString10 convierte [10]byte a string
func ConvertirAString10(bytes [10]byte) string {
	resultado := ""
	for _, b := range bytes {
		if b != 0 {
			resultado += string(b)
		} else {
			break
		}
	}
	return resultado
}
