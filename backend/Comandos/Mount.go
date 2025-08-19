package Comandos

import (
	"fmt"
	"strings"

	"godisk-backend/Structs"
	"godisk-backend/Utils"
)

// Variables globales para el sistema de montaje
var DiscMont [99]DiscoMontado

type DiscoMontado struct {
	Path        [150]byte
	Estado      byte
	Particiones [26]ParticionMontada
}

type ParticionMontada struct {
	Letra        byte
	Estado       byte
	Nombre       [20]byte
	Id_Particion [10]byte
}

var alfabeto = []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z'}

// ValidarDatosMOUNT valida los parámetros del comando MOUNT
func ValidarDatosMOUNT(tokens []string) string {
	if len(tokens) < 2 {
		return Utils.Error("MOUNT", "Se requieren al menos 2 parámetros para este comando.")
	}

	name := ""
	path := ""

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
		case "name":
			if name == "" {
				name = value
			} else {
				return Utils.Error("MOUNT", "parámetro name repetido")
			}
		case "path":
			if path == "" {
				path = value
			} else {
				return Utils.Error("MOUNT", "parámetro path repetido")
			}
		default:
			return Utils.Error("MOUNT", "parámetro no reconocido: "+param)
		}
	}

	// Validaciones
	if name == "" || path == "" {
		return Utils.Error("MOUNT", "Los parámetros name y path son obligatorios")
	}

	// Ejecutar mount
	resultado := mount(path, name)
	if resultado != "" {
		return resultado // Error
	}

	// Solo retornar el mensaje de éxito, sin el listado
	return Utils.Mensaje("MOUNT", fmt.Sprintf("Partición '%s' montada correctamente", name))
}

// mount monta una partición en el sistema
func mount(diskPath string, partitionName string) string {
	fmt.Printf("🔧 DEBUG: Montando partición - Path: %s, Name: %s\n", diskPath, partitionName)

	// Verificar que el disco existe
	if !Utils.ArchivoExiste(diskPath) {
		return Utils.Error("MOUNT", "El disco no existe en la ruta: "+diskPath)
	}

	// Leer MBR del disco
	mbr := leerDisco(diskPath)
	if mbr == nil {
		return Utils.Error("MOUNT", "Error al leer el MBR del disco")
	}

	// Buscar la partición
	particion := buscarParticiones(*mbr, partitionName, diskPath)
	if particion == nil {
		return Utils.Error("MOUNT", "No se encontró la partición: "+partitionName)
	}

	// Verificar que no sea una partición extendida
	if particion.Part_type == 'E' || particion.Part_type == 'e' {
		return Utils.Error("MOUNT", "No se puede montar una partición extendida")
	}

	// Verificar si la partición ya está montada
	if buscarParticionMontada(diskPath, partitionName) != "" {
		return Utils.Error("MOUNT", "La partición ya está montada")
	}

	// Buscar o crear entrada para el disco
	indiceDisco := buscarOCrearDisco(diskPath)
	if indiceDisco == -1 {
		return Utils.Error("MOUNT", "No hay espacio disponible para montar más discos")
	}

	// Buscar slot libre para la partición
	indiceParticion := buscarSlotLibre(indiceDisco)
	if indiceParticion == -1 {
		return Utils.Error("MOUNT", "No hay espacio disponible para montar más particiones en este disco")
	}

	// Generar ID de partición
	idParticion := generarIdParticion(indiceDisco, indiceParticion)

	// Obtener la letra del disco para la partición
	letraDelDisco := obtenerLetraDelDisco(indiceDisco)

	// Montar la partición
	DiscMont[indiceDisco].Particiones[indiceParticion].Estado = 1
	DiscMont[indiceDisco].Particiones[indiceParticion].Letra = letraDelDisco[0]
	copy(DiscMont[indiceDisco].Particiones[indiceParticion].Nombre[:], partitionName)
	copy(DiscMont[indiceDisco].Particiones[indiceParticion].Id_Particion[:], idParticion)

	fmt.Printf("🔧 DEBUG: Partición montada con ID: %s\n", idParticion)
	return "" // Sin errores
}

// GetMount obtiene información de una partición montada
func GetMount(comando string, id string, path *string) *Structs.Particion {
	fmt.Printf("🔧 DEBUG: Buscando partición con ID: %s\n", id)

	// DEBUG: Mostrar todas las particiones montadas
	fmt.Printf("🔧 DEBUG: Listando todas las particiones montadas:\n")
	for i := 0; i < 99; i++ {
		for j := 0; j < 26; j++ {
			if DiscMont[i].Particiones[j].Estado == 1 {
				idMontado := convertirAString10(DiscMont[i].Particiones[j].Id_Particion)
				fmt.Printf("   - Disco %d, Slot %d: ID='%s'\n", i, j, idMontado)
			}
		}
	}

	// Buscar en todos los discos la partición con este ID exacto
	for i := 0; i < 99; i++ {
		for j := 0; j < 26; j++ {
			if DiscMont[i].Particiones[j].Estado == 1 {
				idMontado := convertirAString10(DiscMont[i].Particiones[j].Id_Particion)

				fmt.Printf("🔧 DEBUG: Comparando - ID montado: '%s' vs buscado: '%s'\n", idMontado, id)

				if idMontado == id {
					fmt.Printf("✅ DEBUG: Partición encontrada en disco %d, slot %d\n", i, j)

					// Obtener path del disco
					diskPath := convertirAString150(DiscMont[i].Path)
					*path = diskPath

					// Leer MBR y buscar partición
					mbr := leerDisco(diskPath)
					if mbr == nil {
						fmt.Printf("❌ [%s] Error al leer MBR\n", comando)
						return nil
					}

					nombreParticion := convertirAString20(DiscMont[i].Particiones[j].Nombre)
					fmt.Printf("🔧 DEBUG: Buscando partición '%s' en MBR\n", nombreParticion)
					return buscarParticiones(*mbr, nombreParticion, diskPath)
				}
			}
		}
	}

	fmt.Printf("❌ [%s] Partición no encontrada con ID: %s\n", comando, id)
	return nil
}

// listaMount muestra todas las particiones montadas
func listaMount() string {
	fmt.Println("\n📋 LISTADO DE PARTICIONES MONTADAS")
	fmt.Println("══════════════════════════════════════════════════════════════")

	resultado := "\n📋 LISTADO DE PARTICIONES MONTADAS\n"
	resultado += "══════════════════════════════════════════════════════════════\n"

	hayMontajes := false
	for i := 0; i < 99; i++ {
		for j := 0; j < 26; j++ {
			if DiscMont[i].Particiones[j].Estado == 1 {
				hayMontajes = true

				nombre := convertirAString20(DiscMont[i].Particiones[j].Nombre)
				idParticion := convertirAString10(DiscMont[i].Particiones[j].Id_Particion)
				path := convertirAString150(DiscMont[i].Path)
				letra := string(DiscMont[i].Particiones[j].Letra)

				// Mostrar en debug
				fmt.Printf("🔹 ID: %s | Path: %s | Nombre: %s | Letra: %s\n",
					idParticion, path, nombre, letra)

				// Agregar al resultado
				linea := fmt.Sprintf("🔹 ID: %s | Path: %s | Nombre: %s | Letra: %s\n",
					idParticion, path, nombre, letra)
				resultado += linea
			}
		}
	}

	if !hayMontajes {
		mensaje := "❌ No hay particiones montadas actualmente"
		fmt.Println(mensaje)
		resultado += mensaje + "\n"
	}

	fmt.Println("══════════════════════════════════════════════════════════════")
	resultado += "══════════════════════════════════════════════════════════════\n"
	return resultado
}

// Funciones auxiliares

// buscarParticionMontada verifica si una partición ya está montada
func buscarParticionMontada(diskPath, partitionName string) string {
	for i := 0; i < 99; i++ {
		pathMontado := convertirAString150(DiscMont[i].Path)
		if pathMontado == diskPath {
			for j := 0; j < 26; j++ {
				if DiscMont[i].Particiones[j].Estado == 1 {
					nombreMontado := convertirAString20(DiscMont[i].Particiones[j].Nombre)
					if nombreMontado == partitionName {
						return convertirAString10(DiscMont[i].Particiones[j].Id_Particion)
					}
				}
			}
		}
	}
	return ""
}

// buscarOCrearDisco busca un disco existente o crea una nueva entrada
func buscarOCrearDisco(diskPath string) int {
	// PASO 1: Buscar si el disco ya existe
	for i := 0; i < 99; i++ {
		if DiscMont[i].Estado == 1 {
			pathMontado := convertirAString150(DiscMont[i].Path)
			if pathMontado == diskPath {
				fmt.Printf("🔧 DEBUG: Disco existente encontrado en índice %d\n", i)
				return i
			}
		}
	}

	// PASO 2: Si no existe, crear nueva entrada EN EL PRIMER SLOT LIBRE
	for i := 0; i < 99; i++ {
		if DiscMont[i].Estado == 0 {
			DiscMont[i].Estado = 1
			copy(DiscMont[i].Path[:], diskPath)
			fmt.Printf("🔧 DEBUG: Nuevo disco creado en índice %d\n", i)
			return i
		}
	}

	return -1 // No hay espacio
}

// buscarSlotLibre busca un slot libre para montar una partición
func buscarSlotLibre(indiceDisco int) int {
	for j := 0; j < 26; j++ {
		if DiscMont[indiceDisco].Particiones[j].Estado == 0 {
			return j
		}
	}
	return -1 // No hay espacio
}

// generarIdParticion genera un ID único para la partición
func generarIdParticion(indiceDisco, indiceParticion int) string {
	carnet := "75" // Últimos 2 dígitos del carnet 202307775

	// PASO 1: Determinar la letra del disco basado en cuántos discos únicos hay montados
	letraDelDisco := obtenerLetraDelDisco(indiceDisco)

	// PASO 2: Contar cuántas particiones ya están montadas EN ESTE DISCO ESPECÍFICO
	numeroParticion := contarParticionesMontadasEnDisco(indiceDisco) + 1

	return fmt.Sprintf("%s%d%s", carnet, numeroParticion, letraDelDisco)
}

// obtenerLetraDelDisco obtiene la letra correspondiente al disco basado en el orden de montaje
func obtenerLetraDelDisco(indiceDisco int) string {
	// Contar cuántos discos diferentes ya están montados ANTES de este disco
	discosUnicos := 0

	for i := 0; i < indiceDisco; i++ {
		if DiscMont[i].Estado == 1 {
			discosUnicos++
		}
	}

	// La letra corresponde al orden de los discos montados (A, B, C, ...)
	return string(alfabeto[discosUnicos])
}

// contarParticionesMontadasEnDisco cuenta las particiones ya montadas en un disco específico
func contarParticionesMontadasEnDisco(indiceDisco int) int {
	contador := 0
	for j := 0; j < 26; j++ {
		if DiscMont[indiceDisco].Particiones[j].Estado == 1 {
			contador++
		}
	}
	return contador
}

// Funciones auxiliares para conversión de tipos específicos

// convertirAString20 convierte [20]byte a string
func convertirAString20(bytes [20]byte) string {
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

// convertirAString150 convierte [150]byte a string
func convertirAString150(bytes [150]byte) string {
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

// convertirAString10 convierte [10]byte a string
func convertirAString10(bytes [10]byte) string {
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

func ValidarDatosMOUNTED(tokens []string) string {
	// El comando MOUNTED no requiere parámetros
	// Solo muestra todas las particiones montadas
	return mounted()
}

// mounted muestra todas las particiones montadas en el sistema
func mounted() string {
	fmt.Println("\n📋 MOUNTED - PARTICIONES MONTADAS EN EL SISTEMA")
	fmt.Println("══════════════════════════════════════════════════════════════════════")

	resultado := "\n📋 MOUNTED - PARTICIONES MONTADAS EN EL SISTEMA\n"
	resultado += "══════════════════════════════════════════════════════════════════════\n"

	hayMontajes := false

	for i := 0; i < 99; i++ {
		for j := 0; j < 26; j++ {
			if DiscMont[i].Particiones[j].Estado == 1 {
				hayMontajes = true

				// Extraer información de la partición montada
				nombre := convertirAString20(DiscMont[i].Particiones[j].Nombre)
				idParticion := convertirAString10(DiscMont[i].Particiones[j].Id_Particion)
				path := convertirAString150(DiscMont[i].Path)
				letra := string(DiscMont[i].Particiones[j].Letra)

				// Mostrar información formateada (debug)
				fmt.Printf("🔹 ID: %-10s | Path: %-30s | Nombre: %-15s | Letra: %s\n",
					idParticion, path, nombre, letra)

				// Agregar al resultado para la aplicación web
				linea := fmt.Sprintf("🔹 ID: %-10s | Path: %-30s | Nombre: %-15s | Letra: %s\n",
					idParticion, path, nombre, letra)
				resultado += linea
			}
		}
	}

	if !hayMontajes {
		mensaje := "❌ No hay particiones montadas actualmente\n   Use el comando MOUNT para montar una partición"
		fmt.Println("❌ No hay particiones montadas actualmente")
		fmt.Println("   Use el comando MOUNT para montar una partición")
		resultado += mensaje + "\n"
		resultado += "══════════════════════════════════════════════════════════════════════\n"
		return resultado
	}

	fmt.Println("══════════════════════════════════════════════════════════════════════")
	resultado += "══════════════════════════════════════════════════════════════════════\n"
	return resultado
}
