package Comandos

import (
	"bytes"
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
	var archivos []string
	var idParticion string

	// Parsear tokens
	for _, token := range tokens {
		parts := strings.Split(token, "=")
		if len(parts) != 2 {
			continue
		}

		param := strings.ToLower(parts[0])
		value := strings.Trim(parts[1], "\"") // Remover comillas

		switch param {
		case "id":
			idParticion = value
		default:
			if strings.HasPrefix(param, "file") {
				// Normalizar path
				value = strings.ReplaceAll(value, "\\", "/")
				archivos = append(archivos, value)
			}
		}
	}

	if len(archivos) == 0 {
		return Utils.Error("CAT", "Se requiere al menos un archivo (file1)")
	}

	return cat(archivos, idParticion)
}

// cat muestra el contenido de uno o más archivos
func cat(archivos []string, idParticion string) string {
	resultado := "\n📄 CONTENIDO DE ARCHIVOS\n"
	resultado += "═══════════════════════════════════════\n"

	for _, archivo := range archivos {
		contenido := leerArchivoReal(archivo, idParticion)
		if contenido == "" {
			resultado += fmt.Sprintf("❌ Error al leer: %s\n", archivo)
			continue
		}
		resultado += fmt.Sprintf("📄 %s:\n%s\n\n", archivo, contenido)
	}

	return resultado
}

func leerArchivoReal(rutaArchivo string, idParticion string) string {
	fmt.Printf("🔧 DEBUG: Iniciando lectura de archivo '%s'\n", rutaArchivo)

	// Determinar qué partición usar
	var idFinal string
	if idParticion != "" {
		idFinal = idParticion
	} else {
		idFinal = obtenerParticionDeSesion()
		if idFinal == "" {
			idFinal = obtenerPrimeraParticionMontada()
			if idFinal == "" {
				return Utils.Error("CAT", "No hay particiones montadas")
			}
		}
	}

	// Obtener la partición montada
	var pathDisco string
	particion := GetMount("CAT", idFinal, &pathDisco)
	if particion == nil {
		return Utils.Error("CAT", "Partición no encontrada: "+idFinal)
	}

	// Abrir archivo del disco
	file, err := os.OpenFile(pathDisco, os.O_RDWR, 0644)
	if err != nil {
		return Utils.Error("CAT", "Error abriendo disco: "+err.Error())
	}
	defer file.Close()

	// Leer superbloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	if err := binary.Read(file, binary.LittleEndian, &super); err != nil {
		return Utils.Error("CAT", "Error leyendo superbloque: "+err.Error())
	}

	// Preparar tamaños
	tamInodo := int64(unsafe.Sizeof(Structs.Inodos{}))
	tamBloqueCarp := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	// tamBloqueArch not needed; readers use fileBlockOffset helper
	// Caso especial: users.txt en la raíz
	if rutaArchivo == "/users.txt" || rutaArchivo == "users.txt" {
		fmt.Printf("🔧 DEBUG: Detectado users.txt\n")
		contenido := leerUsersFile(file, super)
		if contenido == "" {
			return Utils.Error("CAT", "Error leyendo users.txt")
		}
		return contenido
	}

	// Para otros archivos, proceder con la navegación normal
	components := strings.Split(strings.TrimSpace(rutaArchivo), "/")
	components = components[1:] // Remover elemento vacío inicial
	if len(components) == 0 {
		return Utils.Error("CAT", "Ruta inválida")
	}

	// Leer inodo raíz
	var currentInode Structs.Inodos
	currentInodePos := super.S_inode_start
	file.Seek(currentInodePos, 0)
	if err := binary.Read(file, binary.LittleEndian, &currentInode); err != nil {
		return Utils.Error("CAT", "Error leyendo inodo raíz")
	}

	// Navegar la ruta
	for i, component := range components {
		if component == "" {
			continue
		}

		// Si es el último componente, buscar archivo
		if i == len(components)-1 {
			found := false

			for j := 0; j < 12 && !found && currentInode.I_block[j] != -1; j++ {
				var dirBlock Structs.BloquesCarpetas
				blockPos := super.S_block_start + (currentInode.I_block[j] * tamBloqueCarp)

				file.Seek(blockPos, 0)
				if err := binary.Read(file, binary.LittleEndian, &dirBlock); err != nil {
					continue
				}

				for _, entry := range dirBlock.B_content {
					if entry.B_inodo == -1 {
						continue
					}

					name := strings.Trim(string(entry.B_name[:]), "\x00")
					if name == component {
						// Leer inodo del archivo
						var fileInode Structs.Inodos
						file.Seek(super.S_inode_start+(entry.B_inodo*tamInodo), 0)
						if err := binary.Read(file, binary.LittleEndian, &fileInode); err != nil {
							return Utils.Error("CAT", "Error leyendo inodo del archivo")
						}

						if fileInode.I_type != 1 {
							return Utils.Error("CAT", fmt.Sprintf("'%s' no es un archivo", component))
						}

						// Leer contenido
						var content strings.Builder
						for k := 0; k < 12 && fileInode.I_block[k] != -1; k++ {
							dataBlock, err := readFileBlock(file, super, fileInode.I_block[k])
							if err != nil {
								continue
							}

							bytesRestantes := fileInode.I_size - int64(content.Len())
							if bytesRestantes > int64(len(dataBlock)) {
								bytesRestantes = int64(len(dataBlock))
							}
							content.Write(dataBlock[:bytesRestantes])
						}

						return content.String()
					}
				}
			}

			if !found {
				return Utils.Error("CAT", fmt.Sprintf("Archivo no encontrado: %s", component))
			}
		}

		// Navegar al siguiente directorio
		found := false
		for j := 0; j < 12 && !found && currentInode.I_block[j] != -1; j++ {
			var dirBlock Structs.BloquesCarpetas
			blockPos := super.S_block_start + (currentInode.I_block[j] * tamBloqueCarp)

			file.Seek(blockPos, 0)
			if err := binary.Read(file, binary.LittleEndian, &dirBlock); err != nil {
				continue
			}

			for _, entry := range dirBlock.B_content {
				name := strings.Trim(string(entry.B_name[:]), "\x00")
				if name == component {
					file.Seek(super.S_inode_start+(entry.B_inodo*tamInodo), 0)
					if err := binary.Read(file, binary.LittleEndian, &currentInode); err != nil {
						return Utils.Error("CAT", "Error leyendo inodo")
					}

					if currentInode.I_type != 0 {
						return Utils.Error("CAT", fmt.Sprintf("'%s' no es un directorio", component))
					}

					found = true
					break
				}
			}
		}

		if !found {
			return Utils.Error("CAT", fmt.Sprintf("No existe el directorio '%s'", component))
		}
	}

	return Utils.Error("CAT", "Error inesperado leyendo archivo")
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
	err := binary.Read(file, binary.LittleEndian, &inodo)

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
	if err := binary.Read(file, binary.LittleEndian, &bloque); err != nil {
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
		return ""
	}

	var contenido strings.Builder

	// Leer bloques directos
	var bytesLeidos int64
	for i := 0; i < 12 && inodo.I_block[i] != -1 && bytesLeidos < inodo.I_size; i++ {
		// Usar la función auxiliar para leer cada bloque
		bloqueContenido := leerBloqueArchivo(file, sb, inodo.I_block[i])

		// Verificar si aún tenemos espacio en el archivo
		if bytesLeidos+int64(len(bloqueContenido)) > inodo.I_size {
			// Truncar el contenido para no exceder el tamaño del archivo
			contenido.WriteString(bloqueContenido[:inodo.I_size-bytesLeidos])
			bytesLeidos = inodo.I_size
		} else {
			// Agregar todo el contenido del bloque
			contenido.WriteString(bloqueContenido)
			bytesLeidos += int64(len(bloqueContenido))
		}
	}

	return contenido.String()
}

// procesarBloqueIndirecto procesa un bloque de apuntadores y lee los bloques referenciados
func procesarBloqueIndirecto(file *os.File, sb Structs.SuperBloque,
	numeroBloque int64, tamanoArchivo int64, bytesLeidos int64,
	contenido *strings.Builder) int64 {

	// Leer bloque de apuntadores usando la ranura canónica
	var bloqueApuntadores Structs.BloquesApuntadores
	if _, err := file.Seek(fileBlockOffset(sb, numeroBloque), 0); err != nil {
		fmt.Printf("❌ CAT: Error posicionando bloque de apuntadores: %v\n", err)
		return bytesLeidos
	}
	if err := binary.Read(file, binary.LittleEndian, &bloqueApuntadores); err != nil {
		fmt.Printf("❌ CAT: Error leyendo bloque de apuntadores: %v\n", err)
		return bytesLeidos
	}

	for _, ptr := range bloqueApuntadores.B_pointers {
		if ptr == -1 || bytesLeidos >= tamanoArchivo {
			break
		}

		data, err := readFileBlock(file, sb, ptr)
		if err != nil {
			continue
		}

		if len(data) > 0 {
			if bytesLeidos+int64(len(data)) > tamanoArchivo {
				contenido.WriteString(string(data[:tamanoArchivo-bytesLeidos]))
				bytesLeidos = tamanoArchivo
			} else {
				contenido.WriteString(string(data))
				bytesLeidos += int64(len(data))
			}
		}
	}

	return bytesLeidos
}

// Funciones auxiliares existentes
func obtenerParticionDeSesion() string {
	if EstaLogueado() {
		return ObtenerIDParticionActual()
	}
	return ""
}

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

func leerUsersFile(file *os.File, sb Structs.SuperBloque) string {
	// Leer inodo de users.txt (inodo 1)
	tamInodo := int64(unsafe.Sizeof(Structs.Inodos{}))
	var inodoUsers Structs.Inodos
	file.Seek(sb.S_inode_start+tamInodo, 0)
	if err := binary.Read(file, binary.LittleEndian, &inodoUsers); err != nil {
		fmt.Printf("❌ Error leyendo inodo users.txt: %v\n", err)
		return ""
	}

	fmt.Printf("🔧 DEBUG: Leyendo users.txt - Tamaño: %d bytes\n", inodoUsers.I_size)

	if inodoUsers.I_type != 1 || inodoUsers.I_size == 0 {
		fmt.Printf("❌ Error: users.txt no es un archivo válido\n")
		return ""
	}

	var contenido strings.Builder
	bytesRestantes := inodoUsers.I_size

	for i := 0; i < 12 && inodoUsers.I_block[i] != -1 && bytesRestantes > 0; i++ {
		data, err := readFileBlock(file, sb, inodoUsers.I_block[i])
		if err != nil {
			fmt.Printf("❌ Error leyendo bloque %d: %v\n", i, err)
			continue
		}

		bytesAProcesar := int64(len(data))
		if bytesRestantes < bytesAProcesar {
			bytesAProcesar = bytesRestantes
		}

		if bytesAProcesar > 0 {
			contenido.WriteString(string(data[:bytesAProcesar]))
			bytesRestantes -= bytesAProcesar
		}
	}

	resultado := contenido.String()
	fmt.Printf("🔧 DEBUG: Contenido leído (%d bytes):\n%s\n", len(resultado), resultado)
	return resultado
}

// leerBloqueArchivo lee correctamente un bloque de archivo y devuelve su contenido como string
func leerBloqueArchivo(file *os.File, sb Structs.SuperBloque, bloqueIdx int64) string {
	if bloqueIdx == -1 {
		return ""
	}

	tamBloque := int64(unsafe.Sizeof(Structs.BloquesArchivos{}))
	posBloque := sb.S_block_start + (bloqueIdx * tamBloque)

	file.Seek(posBloque, 0)
	var bloqueArchivo Structs.BloquesArchivos
	if err := binary.Read(file, binary.LittleEndian, &bloqueArchivo); err != nil {
		fmt.Printf("❌ Error leyendo bloque %d: %v\n", bloqueIdx, err)
		return ""
	}

	// Eliminamos bytes nulos y convertimos a string
	contenidoBytes := bytes.TrimRight(bloqueArchivo.B_content[:], "\x00")
	contenido := string(contenidoBytes)

	fmt.Printf("🔧 DEBUG: Bloque %d leído, contenido: %q\n", bloqueIdx, contenido)
	return contenido
}
