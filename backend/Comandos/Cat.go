package Comandos

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"godisk-backend/Structs"
	"godisk-backend/Utils"
)

// ValidarDatosCAT valida los parámetros del comando CAT
func ValidarDatosCAT(tokens []string) string {
	if len(tokens) < 1 {
		return Utils.Error("CAT", "Se requiere al menos el parámetro filen")
	}

	var archivos []string
	var idParticion string

	// Parsear tokens para obtener múltiples archivos y partición
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		tk := strings.Split(token, "=")
		if len(tk) != 2 {
			continue
		}

		param := strings.ToLower(tk[0])
		value := strings.ReplaceAll(tk[1], "\"", "")

		switch param {
		case "id":
			idParticion = tk[1] // Mantener case original del ID
		default:
			// Aceptar file1, file2, file3... fileN
			if strings.HasPrefix(param, "file") {
				archivos = append(archivos, value)
			} else {
				return Utils.Error("CAT", "Parámetro no reconocido: "+param)
			}
		}
	}

	// Validaciones
	if len(archivos) == 0 {
		return Utils.Error("CAT", "Se requiere al menos un archivo (filen)")
	}

	return cat(archivos, idParticion)
}

// cat muestra el contenido de uno o más archivos
func cat(archivos []string, idParticion string) string {
	fmt.Printf("🔧 DEBUG: Ejecutando CAT con %d archivos, ID: %s\n", len(archivos), idParticion)

	resultado := "\n📄 CONTENIDO DE ARCHIVOS\n"
	resultado += "══════════════════════════════════════════════════════════════\n"

	for i, archivo := range archivos {
		fmt.Printf("🔧 DEBUG: Leyendo archivo: %s\n", archivo)

		contenido := leerArchivoReal(archivo, idParticion)
		if contenido == "" {
			error := fmt.Sprintf("❌ Error al leer el archivo: %s", archivo)
			fmt.Println(error)
			resultado += error + "\n"
			continue
		}

		// Mostrar en debug
		fmt.Printf("📄 Contenido de %s:\n%s\n", archivo, contenido)

		// Agregar al resultado
		if i > 0 {
			resultado += "\n" // Separador entre archivos
		}
		resultado += fmt.Sprintf("📄 %s:\n%s\n", archivo, contenido)
	}

	resultado += "══════════════════════════════════════════════════════════════\n"
	return resultado
}

// leerArchivoReal lee un archivo real del sistema de archivos EXT2
func leerArchivoReal(rutaArchivo string, idParticion string) string {
	// Determinar qué partición usar
	var idFinal string

	if idParticion != "" {
		idFinal = idParticion
		fmt.Printf("🔧 DEBUG: Usando ID especificado: %s\n", idFinal)
	} else {
		idFinal = obtenerParticionDeSesion()
		if idFinal == "" {
			idFinal = obtenerPrimeraParticionMontada()
			if idFinal == "" {
				fmt.Printf("❌ CAT: No hay particiones montadas\n")
				return ""
			}
		}
		fmt.Printf("🔧 DEBUG: Usando ID automático: %s\n", idFinal)
	}

	fmt.Printf("🔧 DEBUG: Buscando archivo '%s' en partición %s\n", rutaArchivo, idFinal)

	// 1. Obtener la partición montada
	var pathDisco string
	particion := GetMount("CAT", idFinal, &pathDisco)
	if particion == nil {
		fmt.Printf("❌ CAT: Partición %s no está montada\n", idFinal)
		return ""
	}

	fmt.Printf("🔧 DEBUG: Partición encontrada en: %s\n", pathDisco)

	// 2. Abrir archivo del disco
	file, err := os.Open(pathDisco)
	if err != nil {
		fmt.Printf("❌ CAT: Error al abrir disco: %v\n", err)
		return ""
	}
	defer file.Close()

	// 3. Leer superbloque
	file.Seek(particion.Part_start, 0)
	var superbloque Structs.SuperBloque
	if err := binary.Read(file, binary.BigEndian, &superbloque); err != nil {
		fmt.Printf("❌ CAT: Error al leer superbloque: %v\n", err)
		return ""
	}

	fmt.Printf("🔧 DEBUG: SuperBloque leído - FS: %d, Inodos: %d\n",
		superbloque.S_filesystem_type, superbloque.S_inodes_count)

	// 4. Buscar el archivo en el sistema de archivos
	contenido := buscarArchivoEnSistema(file, superbloque, rutaArchivo)

	return contenido
}

// buscarArchivoEnSistema busca un archivo en el sistema EXT2 y retorna su contenido
func buscarArchivoEnSistema(file *os.File, sb Structs.SuperBloque, rutaArchivo string) string {
	// Separar la ruta en componentes
	componentes := strings.Split(strings.Trim(rutaArchivo, "/"), "/")
	if len(componentes) == 1 && componentes[0] == "" {
		componentes = []string{} // Ruta raíz
	}

	fmt.Printf("🔧 DEBUG: Buscando componentes: %v\n", componentes)

	// CASO ESPECIAL: Archivo directamente en la raíz (como "users.txt")
	if len(componentes) == 1 && componentes[0] != "" {
		fmt.Printf("🔧 DEBUG: Buscando archivo '%s' en directorio raíz\n", componentes[0])

		// Leer inodo del directorio raíz (inodo 0)
		inodoRaiz, err := leerInodo(file, sb, 0)
		if err != nil {
			fmt.Printf("❌ CAT: Error al leer inodo raíz: %v\n", err)
			return ""
		}

		// Buscar el archivo en el directorio raíz
		inodoArchivo := buscarEnDirectorio(file, sb, inodoRaiz, componentes[0])
		if inodoArchivo == -1 {
			fmt.Printf("❌ CAT: No se encontró '%s' en el directorio raíz\n", componentes[0])
			return ""
		}

		// Leer el inodo del archivo
		inodo, err := leerInodo(file, sb, inodoArchivo)
		if err != nil {
			fmt.Printf("❌ CAT: Error al leer inodo del archivo: %v\n", err)
			return ""
		}

		// Verificar que es un archivo
		if inodo.I_type != 1 {
			fmt.Printf("❌ CAT: '%s' no es un archivo (tipo: %d)\n", componentes[0], inodo.I_type)
			return ""
		}

		// Leer y retornar el contenido
		return leerContenidoArchivo(file, sb, inodo)
	}

	// CASO GENERAL: Navegación por directorios para rutas más complejas
	inodoActual := int64(0)

	// Navegar por cada componente de la ruta
	for i, componente := range componentes {
		fmt.Printf("🔧 DEBUG: Procesando componente '%s' (nivel %d)\n", componente, i)

		// Leer el inodo actual
		inodo, err := leerInodo(file, sb, inodoActual)
		if err != nil {
			fmt.Printf("❌ CAT: Error al leer inodo %d: %v\n", inodoActual, err)
			return ""
		}

		// Si es el último componente y esperamos un archivo
		if i == len(componentes)-1 {
			if inodo.I_type == 1 { // Es archivo
				fmt.Printf("✅ DEBUG: Archivo encontrado en inodo %d\n", inodoActual)
				return leerContenidoArchivo(file, sb, inodo)
			} else {
				fmt.Printf("❌ CAT: '%s' es un directorio, no un archivo\n", componente)
				return ""
			}
		}

		// Si no es el último componente, debe ser un directorio
		if inodo.I_type != 0 {
			fmt.Printf("❌ CAT: '%s' no es un directorio\n", componente)
			return ""
		}

		// Buscar el siguiente componente en el directorio actual
		siguienteInodo := buscarEnDirectorio(file, sb, inodo, componente)
		if siguienteInodo == -1 {
			fmt.Printf("❌ CAT: No se encontró '%s' en el directorio\n", componente)
			return ""
		}

		inodoActual = siguienteInodo
	}

	// Si llegamos aquí, la ruta era solo "/" (directorio raíz)
	fmt.Printf("❌ CAT: No se puede hacer CAT de un directorio\n")
	return ""
}

// leerInodo lee un inodo específico del sistema de archivos
func leerInodo(file *os.File, sb Structs.SuperBloque, numeroInodo int64) (Structs.Inodos, error) {
	var inodo Structs.Inodos

	// Calcular posición del inodo
	inodoSize := int64(unsafe.Sizeof(Structs.Inodos{}))
	posicion := sb.S_inode_start + (numeroInodo * inodoSize)

	fmt.Printf("🔧 DEBUG: Leyendo inodo %d en posición %d\n", numeroInodo, posicion)

	// Leer el inodo
	file.Seek(posicion, 0)
	err := binary.Read(file, binary.BigEndian, &inodo)

	if err == nil {
		fmt.Printf("🔧 DEBUG: Inodo %d - Tipo: %d, Tamaño: %d, Bloque[0]: %d\n",
			numeroInodo, inodo.I_type, inodo.I_size, inodo.I_block[0])
	}

	return inodo, err
}

// buscarEnDirectorio busca una entrada en un directorio y retorna el número de inodo
func buscarEnDirectorio(file *os.File, sb Structs.SuperBloque, inodoDir Structs.Inodos, nombreBuscado string) int64 {
	// Leer el bloque del directorio
	bloqueSize := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	posicionBloque := sb.S_block_start + (inodoDir.I_block[0] * bloqueSize)

	fmt.Printf("🔧 DEBUG: Buscando '%s' en directorio, bloque en posición %d\n", nombreBuscado, posicionBloque)

	file.Seek(posicionBloque, 0)
	var bloque Structs.BloquesCarpetas
	if err := binary.Read(file, binary.BigEndian, &bloque); err != nil {
		fmt.Printf("❌ CAT: Error al leer bloque de directorio: %v\n", err)
		return -1
	}

	// Buscar en las entradas del directorio
	for i, entrada := range bloque.B_content {
		if entrada.B_inodo == -1 { // Entrada vacía
			continue
		}

		// Convertir nombre de la entrada a string
		nombreEntrada := ""
		for _, b := range entrada.B_name {
			if b != 0 {
				nombreEntrada += string(b)
			} else {
				break
			}
		}

		fmt.Printf("🔧 DEBUG: Entrada[%d]: '%s' -> inodo %d\n", i, nombreEntrada, entrada.B_inodo)

		if nombreEntrada == nombreBuscado {
			fmt.Printf("✅ DEBUG: Encontrado '%s' -> inodo %d\n", nombreBuscado, entrada.B_inodo)
			return entrada.B_inodo
		}
	}

	return -1 // No encontrado
}

func leerContenidoArchivo(file *os.File, sb Structs.SuperBloque, inodo Structs.Inodos) string {
	if inodo.I_type != 1 {
		return "" // No es un archivo
	}

	fmt.Printf("🔧 DEBUG: Iniciando lectura de contenido de archivo\n")
	fmt.Printf("🔧 DEBUG: Inodo - Tipo: %d, Tamaño: %d bytes, Bloque[0]: %d\n",
		inodo.I_type, inodo.I_size, inodo.I_block[0])

	mitadBA := sb.S_block_start + int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	TamBA := int64(unsafe.Sizeof(Structs.BloquesArchivos{}))
	var contenido strings.Builder

	for bloque := 0; bloque < 16; bloque++ {
		if inodo.I_block[bloque] == -1 {
			break
		}

		PunteroBA := mitadBA + (int64(inodo.I_block[bloque]-1) * TamBA)

		fmt.Printf("🔧 DEBUG: Leyendo bloque %d en posición %d (Nº%d)\n",
			inodo.I_block[bloque], PunteroBA, bloque)

		// Leer el bloque
		file.Seek(PunteroBA, 0)
		var fb Structs.BloquesArchivos
		err := binary.Read(file, binary.BigEndian, &fb)
		if err != nil {
			fmt.Printf("❌ CAT: Error leyendo bloque: %v\n", err)
			continue
		}

		for i := 0; i < len(fb.B_content); i++ {
			if fb.B_content[i] != 0 {
				contenido.WriteByte(fb.B_content[i])
			}
		}
	}

	resultado := contenido.String()
	fmt.Printf("✅ DEBUG: Contenido leído (%d bytes): %q\n", len(resultado), resultado)
	return resultado
}

// obtenerParticionDeSesion obtiene el ID de partición de la sesión activa
func obtenerParticionDeSesion() string {
	// ✅ INTEGRACIÓN CON LOGIN: Usar la sesión activa de Login.go
	if EstaLogueado() {
		return ObtenerIDParticionActual()
	}
	return ""
}

// obtenerPrimeraParticionMontada obtiene la primera partición montada disponible
func obtenerPrimeraParticionMontada() string {
	fmt.Printf("🔧 DEBUG: Buscando primera partición montada\n")

	for i := 0; i < 99; i++ {
		for j := 0; j < 26; j++ {
			if DiscMont[i].Particiones[j].Estado == 1 {
				idEncontrado := convertirAString10(DiscMont[i].Particiones[j].Id_Particion)
				fmt.Printf("🔧 DEBUG: Primera partición encontrada: %s\n", idEncontrado)
				return idEncontrado
			}
		}
	}

	return ""
}

// listarParticionesMontadas muestra todas las particiones disponibles
func listarParticionesMontadas() []string {
	var particiones []string

	for i := 0; i < 99; i++ {
		for j := 0; j < 26; j++ {
			if DiscMont[i].Particiones[j].Estado == 1 {
				id := convertirAString10(DiscMont[i].Particiones[j].Id_Particion)
				nombre := convertirAString20(DiscMont[i].Particiones[j].Nombre)
				path := convertirAString150(DiscMont[i].Path)

				info := fmt.Sprintf("%s (%s en %s)", id, nombre, path)
				particiones = append(particiones, info)
			}
		}
	}

	return particiones
}

// leerArchivoReal (versión original para compatibilidad)
func leerArchivoOriginal(rutaArchivo string) string {
	fmt.Printf("⚠️ DEBUG: Usando función obsoleta leerArchivo, migrando a leerArchivoReal\n")
	return leerArchivoReal(rutaArchivo, "")
}
