package Comandos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"godisk-backend/Structs"
	"godisk-backend/Utils"
)

// Estructura para manejar sesiones activas
type UsuarioActivo struct {
	User     string
	Password string
	Id       string
	Uid      int
	Gid      int
}

// Variable global para la sesión activa
var Logged UsuarioActivo

// ValidarDatosLOGIN valida los parámetros del comando LOGIN
func ValidarDatosLOGIN(tokens []string) string {
	if len(tokens) < 3 {
		return Utils.Error("LOGIN", "Se requieren los parámetros: -user, -pass, -id")
	}

	var usuario, password, idParticion string

	// Parsear tokens
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		tk := strings.Split(token, "=")
		if len(tk) != 2 {
			continue
		}

		param := strings.ToLower(tk[0])
		value := strings.ReplaceAll(tk[1], "\"", "")

		switch param {
		case "user":
			usuario = value
		case "pass":
			password = value
		case "id":
			idParticion = tk[1] // Mantener case original del ID
		default:
			return Utils.Error("LOGIN", "Parámetro no reconocido: "+param)
		}
	}

	// Validaciones
	if usuario == "" {
		return Utils.Error("LOGIN", "El parámetro -user es obligatorio")
	}
	if password == "" {
		return Utils.Error("LOGIN", "El parámetro -pass es obligatorio")
	}
	if idParticion == "" {
		return Utils.Error("LOGIN", "El parámetro -id es obligatorio")
	}

	// Verificar si ya hay una sesión activa
	if Logged.User != "" {
		return Utils.Error("LOGIN", "Ya hay un usuario logueado. Debe hacer LOGOUT antes de iniciar otra sesión")
	}

	if login(usuario, password, idParticion) {
		return Utils.Mensaje("LOGIN", fmt.Sprintf("Bienvenido %s. Sesión iniciada correctamente", usuario))
	} else {
		return Utils.Error("LOGIN", "Error en las credenciales o partición no encontrada")
	}
}

// login inicia sesión en el sistema
func login(usuario, password, idParticion string) bool {
	fmt.Printf("🔧 DEBUG: Intentando login - Usuario: %s, ID: %s\n", usuario, idParticion)

	// Obtener la partición montada usando la función GetMount de Mount.go
	var pathDisco string
	particion := GetMount("LOGIN", idParticion, &pathDisco)
	if particion == nil {
		fmt.Printf("❌ LOGIN: Partición %s no está montada\n", idParticion)
		return false
	}

	fmt.Printf("🔧 DEBUG: Partición encontrada en: %s\n", pathDisco)

	// Abrir archivo del disco
	file, err := os.Open(strings.ReplaceAll(pathDisco, "\"", ""))
	if err != nil {
		fmt.Printf("❌ LOGIN: No se encontró el disco: %v\n", err)
		return false
	}
	defer file.Close()

	// Leer SuperBloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	data := leerBytes(file, int(unsafe.Sizeof(Structs.SuperBloque{})))
	buffer := bytes.NewBuffer(data)
	err_ := binary.Read(buffer, binary.BigEndian, &super)
	if err_ != nil {
		fmt.Printf("❌ LOGIN: Error al leer superbloque: %v\n", err_)
		return false
	}

	fmt.Printf("🔧 DEBUG: SuperBloque leído - FS: %d\n", super.S_filesystem_type)

	// Leer inodo del archivo users.txt (inodo 1)
	inodo := Structs.NewInodos()
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	data = leerBytes(file, int(unsafe.Sizeof(Structs.Inodos{})))
	buffer = bytes.NewBuffer(data)
	err_ = binary.Read(buffer, binary.BigEndian, &inodo)
	if err_ != nil {
		fmt.Printf("❌ LOGIN: Error al leer inodo users.txt: %v\n", err_)
		return false
	}

	fmt.Printf("🔧 DEBUG: Inodo users.txt - Tipo: %d, Tamaño: %d\n", inodo.I_type, inodo.I_size)

	// Leer contenido del archivo users.txt
	contenidoUsers := leerContenidoUsersArchivo(file, super, inodo)
	if contenidoUsers == "" {
		fmt.Printf("❌ LOGIN: No se pudo leer el archivo users.txt\n")
		return false
	}

	fmt.Printf("🔧 DEBUG: Contenido users.txt:\n%s\n", contenidoUsers)

	// Verificar credenciales
	return verificarCredencialesLogin(usuario, password, contenidoUsers, idParticion)
}

// leerContenidoUsersArchivo lee el contenido completo del archivo users.txt
func leerContenidoUsersArchivo(file *os.File, super Structs.SuperBloque, inodo Structs.Inodos) string {
	var contenido strings.Builder

	// Calcular posición de los bloques de archivos
	mitadBA := (super.S_block_start + int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))) // Después del bloque 0 (directorio)
	tamBA := int64(unsafe.Sizeof(Structs.BloquesArchivos{}))

	// Leer todos los bloques del archivo
	for bloque := 0; bloque < 16; bloque++ {
		if inodo.I_block[bloque] == -1 {
			break
		}

		// Calcular posición del bloque
		posicionBloque := mitadBA + (int64(inodo.I_block[bloque]-1) * tamBA)

		file.Seek(posicionBloque, 0)
		data := leerBytes(file, int(tamBA))
		buffer := bytes.NewBuffer(data)

		var bloqueArchivo Structs.BloquesArchivos
		err := binary.Read(buffer, binary.BigEndian, &bloqueArchivo)
		if err != nil {
			fmt.Printf("❌ Error al leer bloque de archivo: %v\n", err)
			break
		}

		// Extraer contenido del bloque
		for i := 0; i < len(bloqueArchivo.B_content); i++ {
			if bloqueArchivo.B_content[i] != 0 {
				contenido.WriteByte(bloqueArchivo.B_content[i])
			}
		}
	}

	return contenido.String()
}

// verificarCredencialesLogin verifica usuario y contraseña en el contenido de users.txt
func verificarCredencialesLogin(usuario, password, contenidoUsers, idParticion string) bool {
	lineas := strings.Split(strings.TrimSpace(contenidoUsers), "\n")

	fmt.Printf("🔧 DEBUG: Verificando credenciales para usuario '%s'\n", usuario)

	// Buscar usuario
	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" || len(linea) < 3 {
			continue
		}

		// Verificar si es una línea de usuario
		if linea[2] == 'U' || linea[2] == 'u' {
			campos := strings.Split(linea, ",")
			if len(campos) >= 5 && campos[0] != "0" {
				idUsuario := campos[0]
				nombreUsuario := campos[3]
				passwordUsuario := campos[4]
				nombreGrupo := campos[2]

				fmt.Printf("🔧 DEBUG: Usuario encontrado - ID: %s, Nombre: %s, Grupo: %s\n",
					idUsuario, nombreUsuario, nombreGrupo)

				// Verificar credenciales
				if Utils.Comparar(nombreUsuario, usuario) && Utils.Comparar(passwordUsuario, password) {
					// Buscar GID del grupo
					gid := buscarGIDGrupo(nombreGrupo, contenidoUsers)
					if gid == -1 {
						fmt.Printf("❌ LOGIN: No se encontró el grupo '%s'\n", nombreGrupo)
						return false
					}

					// Convertir UID
					uid, err := strconv.Atoi(idUsuario)
					if err != nil {
						fmt.Printf("❌ LOGIN: Error al convertir UID: %v\n", err)
						return false
					}

					// Guardar sesión
					Logged.User = usuario
					Logged.Password = password
					Logged.Id = idParticion
					Logged.Uid = uid
					Logged.Gid = gid

					fmt.Printf("✅ LOGIN: Sesión iniciada - UID: %d, GID: %d\n", uid, gid)
					return true
				}
			}
		}
	}

	fmt.Printf("❌ LOGIN: Usuario '%s' no encontrado o contraseña incorrecta\n", usuario)
	return false
}

// buscarGIDGrupo busca el GID de un grupo en el contenido de users.txt
func buscarGIDGrupo(nombreGrupo, contenidoUsers string) int {
	lineas := strings.Split(strings.TrimSpace(contenidoUsers), "\n")

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" || len(linea) < 3 {
			continue
		}

		// Verificar si es una línea de grupo
		if (linea[2] == 'G' || linea[2] == 'g') && linea[0] != '0' {
			campos := strings.Split(linea, ",")
			if len(campos) >= 3 {
				idGrupo := campos[0]
				nombreGrupoArchivo := campos[2]

				if nombreGrupoArchivo == nombreGrupo {
					gid, err := strconv.Atoi(idGrupo)
					if err != nil {
						return -1
					}
					return gid
				}
			}
		}
	}

	return -1 // No encontrado
}

// LOGOUT cierra la sesión activa
func ValidarDatosLOGOUT(tokens []string) string {
	return logout()
}

func logout() string {
	if Logged.User == "" {
		return Utils.Error("LOGOUT", "No hay ninguna sesión activa")
	}

	usuarioAnterior := Logged.User
	Logged = UsuarioActivo{} // Limpiar sesión

	fmt.Printf("✅ LOGOUT: Sesión cerrada para usuario: %s\n", usuarioAnterior)
	return Utils.Mensaje("LOGOUT", fmt.Sprintf("¡Hasta luego, %s!", usuarioAnterior))
}

// Funciones auxiliares para otros comandos

// ObtenerSesionActiva retorna la sesión actual
func ObtenerSesionActiva() UsuarioActivo {
	return Logged
}

// EstaLogueado verifica si hay una sesión activa
func EstaLogueado() bool {
	return Logged.User != ""
}

// EsUsuarioRoot verifica si el usuario actual es root
func EsUsuarioRoot() bool {
	return EstaLogueado() && Utils.Comparar(Logged.User, "root")
}

// ObtenerIDParticionActual retorna el ID de la partición de la sesión activa
func ObtenerIDParticionActual() string {
	if EstaLogueado() {
		return Logged.Id
	}
	return ""
}

// MostrarInfoSesion muestra información de la sesión activa
func MostrarInfoSesion() string {
	if !EstaLogueado() {
		return "No hay sesión activa"
	}

	info := fmt.Sprintf("Sesión activa:\n")
	info += fmt.Sprintf("- Usuario: %s\n", Logged.User)
	info += fmt.Sprintf("- Tipo: %s\n", map[bool]string{true: "root", false: "usuario"}[EsUsuarioRoot()])
	info += fmt.Sprintf("- UID: %d, GID: %d\n", Logged.Uid, Logged.Gid)
	info += fmt.Sprintf("- Partición: %s\n", Logged.Id)

	return info
}

// leerBytes función auxiliar para leer bytes del archivo
func leerBytes(file *os.File, size int) []byte {
	bytes := make([]byte, size)
	file.Read(bytes)
	return bytes
}
