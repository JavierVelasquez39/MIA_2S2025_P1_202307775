package Comandos

import (
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

	// CORREGIDO: Usar LittleEndian en vez de LittleEndian
	if err := binary.Read(file, binary.LittleEndian, &super); err != nil {
		fmt.Printf("❌ LOGIN: Error al leer superbloque: %v\n", err)
		return false
	}

	fmt.Printf("🔧 DEBUG: SuperBloque leído - FS: %d, Inodos: %d\n",
		super.S_filesystem_type, super.S_inodes_count)

	// Leer inodo del archivo users.txt (inodo 1)
	inodo := Structs.NewInodos()
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)

	// CORREGIDO: Usar LittleEndian en vez de LittleEndian
	if err := binary.Read(file, binary.LittleEndian, &inodo); err != nil {
		fmt.Printf("❌ LOGIN: Error al leer inodo users.txt: %v\n", err)
		return false
	}

	// Verificar que sea un archivo
	if inodo.I_type != 1 {
		fmt.Printf("❌ LOGIN: El inodo users.txt no es un archivo (tipo=%d)\n", inodo.I_type)
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
// leerContenidoUsersArchivo lee el contenido completo del archivo users.txt
func leerContenidoUsersArchivo(file *os.File, super Structs.SuperBloque, inodo Structs.Inodos) string {
	// Verificamos que el inodo tenga al menos un bloque asignado
	if inodo.I_block[0] == -1 {
		fmt.Println("❌ LOGIN: El archivo users.txt no tiene bloques asignados")
		return ""
	}

	// Calcular el tamaño real del archivo
	tamanoArchivo := inodo.I_size
	fmt.Printf("🔧 DEBUG: Tamaño del archivo users.txt: %d bytes\n", tamanoArchivo)

	// CORRECCIÓN CRÍTICA: Usar el mismo cálculo exacto que en formatearEXT2
	// El problema está en cómo se calcula la posición del bloque
	bloquePos := fileBlockOffset(super, inodo.I_block[0])

	fmt.Printf("🔧 DEBUG: Leyendo bloque %d en posición %d\n", inodo.I_block[0], bloquePos)

	// Posicionar el puntero de archivo
	file.Seek(bloquePos, 0)

	// Leer la ranura del bloque de manera segura (evita problemas de alineamiento)
	data, err := readFileBlock(file, super, inodo.I_block[0])
	if err != nil {
		fmt.Printf("❌ LOGIN: Error al leer bloque de archivo: %v\n", err)
		return ""
	}

	// Extraer solo hasta el tamaño real del archivo
	if int64(len(data)) > tamanoArchivo {
		data = data[:tamanoArchivo]
	}
	contenido := string(data)
	fmt.Printf("🔧 DEBUG: Contenido completo (%d bytes):\n%s\n", len(contenido), contenido)

	return contenido
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
		partes := strings.Split(linea, ",")
		if len(partes) < 5 {
			continue
		}

		if partes[1] == "U" || partes[1] == "u" {
			if partes[0] != "0" {
				idUsuario := partes[0]
				nombreGrupo := partes[2]     // GRUPO está en posición 2
				nombreUsuario := partes[3]   // NOMBRE está en posición 3
				passwordUsuario := partes[4] // PASSWORD está en posición 4

				fmt.Printf("🔧 DEBUG: Usuario encontrado - ID: %s, Nombre: %s, Grupo: %s, Password: %s\n",
					idUsuario, nombreUsuario, nombreGrupo, passwordUsuario)

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
				} else {
					// Agregar depuración adicional para ver qué falla
					fmt.Printf("🔧 DEBUG: Comparación fallida - Usuario DB: '%s' vs Input: '%s', Pass DB: '%s' vs Input: '%s'\n",
						nombreUsuario, usuario, passwordUsuario, password)
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

	fmt.Printf("🔧 DEBUG: Buscando grupo '%s' en contenido users.txt\n", nombreGrupo)

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" || len(linea) < 3 {
			continue
		}

		// Verificar si es una línea de grupo
		partes := strings.Split(linea, ",")
		if len(partes) < 3 {
			continue
		}

		if (partes[1] == "G" || partes[1] == "g") && partes[0] != "0" {
			idGrupo := partes[0]
			nombreGrupoArchivo := partes[2]

			fmt.Printf("🔧 DEBUG: Comparando grupo '%s' con '%s'\n", nombreGrupoArchivo, nombreGrupo)

			if Utils.Comparar(nombreGrupoArchivo, nombreGrupo) {
				gid, err := strconv.Atoi(idGrupo)
				if err != nil {
					return -1
				}
				fmt.Printf("✅ DEBUG: Grupo encontrado - ID: %d\n", gid)
				return gid
			}
		}
	}

	fmt.Printf("❌ DEBUG: Grupo '%s' no encontrado\n", nombreGrupo)
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
