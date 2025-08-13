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

type Transition struct {
	partition int
	start     int
	end       int
	after     int
}

var startValue int

// ValidarDatosFDISK valida los parámetros del comando FDISK
func ValidarDatosFDISK(tokens []string) string {
	if len(tokens) < 3 {
		return Utils.Error("FDISK", "Se requieren al menos 3 parámetros para este comando.")
	}

	size := ""
	path := ""
	name := ""
	unit := ""
	tipo := ""
	fit := ""
	delete := ""
	add := ""

	// Parsear tokens - FIX: Normalizar nombres de parámetros
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		tk := strings.Split(token, "=")
		if len(tk) != 2 {
			continue
		}

		param := strings.ToLower(tk[0])
		value := tk[1]

		switch param {
		case "fit":
			if fit == "" {
				fit = strings.ToUpper(value)
			} else {
				return Utils.Error("FDISK", "parámetro fit repetido")
			}
		case "size":
			if size == "" {
				size = value
			} else {
				return Utils.Error("FDISK", "parámetro size repetido")
			}
		case "unit":
			if unit == "" {
				unit = strings.ToUpper(value)
			} else {
				return Utils.Error("FDISK", "parámetro unit repetido")
			}
		case "path":
			if path == "" {
				path = value
			} else {
				return Utils.Error("FDISK", "parámetro path repetido")
			}
		case "name":
			if name == "" {
				name = value
			} else {
				return Utils.Error("FDISK", "parámetro name repetido")
			}
		case "type":
			if tipo == "" {
				tipo = strings.ToUpper(value)
			} else {
				return Utils.Error("FDISK", "parámetro type repetido")
			}
		case "delete":
			if delete == "" {
				delete = value
			} else {
				return Utils.Error("FDISK", "parámetro delete repetido")
			}
		case "add":
			if add == "" {
				add = value
			} else {
				return Utils.Error("FDISK", "parámetro add repetido")
			}
		default:
			return Utils.Error("FDISK", "parámetro no reconocido: "+param)
		}
	}

	// Valores por defecto
	if tipo == "" {
		tipo = "P"
	}
	if fit == "" {
		fit = "WF"
	}
	if unit == "" {
		unit = "K"
	}

	// Validaciones mejoradas
	if name == "" || path == "" || size == "" {
		return Utils.Error("FDISK", "Los parámetros name, path y size son obligatorios")
	}

	validFits := []string{"BF", "FF", "WF"}
	if !Utils.ValidarParametro(fit, validFits) {
		return Utils.Error("FDISK", "Valores válidos para fit: BF, FF, WF")
	}

	validUnits := []string{"B", "K", "M"}
	if !Utils.ValidarParametro(unit, validUnits) {
		return Utils.Error("FDISK", "Valores válidos para unit: B, K, M")
	}

	validTypes := []string{"P", "E", "L"}
	if !Utils.ValidarParametro(tipo, validTypes) {
		return Utils.Error("FDISK", "Valores válidos para type: P, E, L")
	}

	// Ejecutar comando según el tipo
	if delete != "" {
		return eliminarParticion(path, name)
	} else if add != "" {
		return addParticion(path, name, add, unit)
	} else {
		return FDISK(size, path, name, unit, tipo, fit)
	}
}

// FDISK crea una nueva partición
func FDISK(s, path, name, unit, tipo, fit string) string {
	fmt.Printf("🔧 DEBUG: Creando partición - Size: %s, Path: %s, Name: %s, Unit: %s, Type: %s, Fit: %s\n",
		s, path, name, unit, tipo, fit)

	startValue = 0

	// Convertir tamaño
	i, err := strconv.Atoi(s)
	if err != nil {
		return Utils.Error("FDISK", "Size debe ser un número entero")
	}
	if i <= 0 {
		return Utils.Error("FDISK", "Size debe ser mayor que 0")
	}

	// Calcular tamaño en bytes
	sizeBytes := Utils.ConvertirBytes(i, unit)
	fmt.Printf("🔧 DEBUG: Tamaño en bytes: %d\n", sizeBytes)

	// Verificar que el disco existe
	if !Utils.ArchivoExiste(path) {
		return Utils.Error("FDISK", "El disco no existe en la ruta: "+path)
	}

	// Leer MBR del disco
	mbr := leerDisco(path)
	if mbr == nil {
		return Utils.Error("FDISK", "Error al leer el MBR del disco")
	}

	fmt.Printf("🔧 DEBUG: MBR leído correctamente. Tamaño disco: %d bytes\n", mbr.Mbr_tamano)

	// Verificar tamaño
	if sizeBytes > int(mbr.Mbr_tamano) {
		return Utils.Error("FDISK", fmt.Sprintf("Tamaño de partición (%d) mayor que disco (%d)",
			sizeBytes, mbr.Mbr_tamano))
	}

	// Verificar que no existe partición con el mismo nombre
	if buscarParticiones(*mbr, name, path) != nil {
		return Utils.Error("FDISK", "Ya existe una partición con el nombre: "+name)
	}

	// NUEVA VALIDACIÓN: Verificar restricción de partición extendida
	particiones := getParticiones(*mbr)
	if Utils.Comparar(tipo, "E") {
		if existeParticionExtendida(particiones) {
			return Utils.Error("FDISK", "Solo se puede crear una partición extendida por disco. Ya existe una partición extendida.")
		}
	}

	// NUEVA VALIDACIÓN: Para particiones lógicas, verificar que existe partición extendida
	if Utils.Comparar(tipo, "L") {
		if !existeParticionExtendida(particiones) {
			return Utils.Error("FDISK", "No se puede crear una partición lógica sin una partición extendida. Cree primero una partición extendida (-type=E)")
		}
	}

	// NUEVA VALIDACIÓN: Verificar límite de particiones primarias
	if Utils.Comparar(tipo, "P") {
		numPrimarias := contarParticionesPrimarias(particiones)
		numExtendidas := contarParticionesExtendidas(particiones)

		if numPrimarias+numExtendidas >= 4 {
			return Utils.Error("FDISK", "No se pueden crear más de 4 particiones primarias y extendidas en total")
		}
	}

	espaciosLibres := calcularEspaciosLibres(*mbr, particiones)
	fmt.Printf("🔧 DEBUG: Espacios libres encontrados: %d\n", len(espaciosLibres))

	// Buscar espacio disponible usando estrategia de fit
	posicionInicio := buscarEspacioConFit(espaciosLibres, sizeBytes, fit)
	if posicionInicio == -1 {
		return Utils.Error("FDISK", "No hay espacio suficiente para la partición")
	}

	fmt.Printf("🔧 DEBUG: Posición de inicio encontrada: %d\n", posicionInicio)

	// Crear nueva partición
	nuevaParticion := Structs.NewParticion()
	nuevaParticion.Part_status = '1'
	nuevaParticion.Part_type = tipo[0]
	nuevaParticion.Part_fit = fit[0]
	nuevaParticion.Part_start = int64(posicionInicio)
	nuevaParticion.Part_size = int64(sizeBytes)
	copy(nuevaParticion.Part_name[:], name)

	// Asignar a la primera posición libre en el MBR
	if mbr.Mbr_partition_1.Part_status != '1' {
		mbr.Mbr_partition_1 = nuevaParticion
		fmt.Printf("🔧 DEBUG: Partición asignada a slot 1\n")
	} else if mbr.Mbr_partition_2.Part_status != '1' {
		mbr.Mbr_partition_2 = nuevaParticion
		fmt.Printf("🔧 DEBUG: Partición asignada a slot 2\n")
	} else if mbr.Mbr_partition_3.Part_status != '1' {
		mbr.Mbr_partition_3 = nuevaParticion
		fmt.Printf("🔧 DEBUG: Partición asignada a slot 3\n")
	} else if mbr.Mbr_partition_4.Part_status != '1' {
		mbr.Mbr_partition_4 = nuevaParticion
		fmt.Printf("🔧 DEBUG: Partición asignada a slot 4\n")
	} else {
		return Utils.Error("FDISK", "No hay slots disponibles para más particiones")
	}

	// Escribir MBR actualizado al disco
	if err := escribirMBR(path, *mbr); err != nil {
		return Utils.Error("FDISK", "Error al escribir MBR: "+err.Error())
	}

	// Si es extendida, crear EBR inicial
	if Utils.Comparar(tipo, "E") {
		if err := crearEBRInicial(path, posicionInicio); err != nil {
			return Utils.Error("FDISK", "Error al crear EBR: "+err.Error())
		}
		return Utils.Mensaje("FDISK", fmt.Sprintf("Partición extendida '%s' creada correctamente (%s)",
			name, Utils.FormatearTamaño(int64(sizeBytes))))
	}

	tipoStr := "primaria"
	if Utils.Comparar(tipo, "L") {
		tipoStr = "lógica"
	}

	return Utils.Mensaje("FDISK", fmt.Sprintf("Partición %s '%s' creada correctamente (%s)",
		tipoStr, name, Utils.FormatearTamaño(int64(sizeBytes))))
}

// NUEVAS FUNCIONES DE VALIDACIÓN

// existeParticionExtendida verifica si ya existe una partición extendida
func existeParticionExtendida(particiones []Structs.Particion) bool {
	for _, particion := range particiones {
		if particion.Part_status == '1' && (particion.Part_type == 'E' || particion.Part_type == 'e') {
			return true
		}
	}
	return false
}

// contarParticionesPrimarias cuenta las particiones primarias activas
func contarParticionesPrimarias(particiones []Structs.Particion) int {
	count := 0
	for _, particion := range particiones {
		if particion.Part_status == '1' && (particion.Part_type == 'P' || particion.Part_type == 'p') {
			count++
		}
	}
	return count
}

// contarParticionesExtendidas cuenta las particiones extendidas activas
func contarParticionesExtendidas(particiones []Structs.Particion) int {
	count := 0
	for _, particion := range particiones {
		if particion.Part_status == '1' && (particion.Part_type == 'E' || particion.Part_type == 'e') {
			count++
		}
	}
	return count
}

// obtenerParticionExtendida obtiene la partición extendida si existe
func obtenerParticionExtendida(particiones []Structs.Particion) *Structs.Particion {
	for _, particion := range particiones {
		if particion.Part_status == '1' && (particion.Part_type == 'E' || particion.Part_type == 'e') {
			return &particion
		}
	}
	return nil
}

// validarLimitesParticiones verifica todos los límites de particiones
func validarLimitesParticiones(particiones []Structs.Particion, nuevoTipo string) string {
	numPrimarias := contarParticionesPrimarias(particiones)
	numExtendidas := contarParticionesExtendidas(particiones)
	totalActivas := 0

	for _, p := range particiones {
		if p.Part_status == '1' {
			totalActivas++
		}
	}

	switch strings.ToUpper(nuevoTipo) {
	case "P":
		if numPrimarias+numExtendidas >= 4 {
			return "No se pueden crear más de 4 particiones primarias y extendidas en total"
		}
	case "E":
		if numExtendidas >= 1 {
			return "Solo se puede crear una partición extendida por disco"
		}
		if numPrimarias+numExtendidas >= 4 {
			return "No se pueden crear más de 4 particiones primarias y extendidas en total"
		}
	case "L":
		if numExtendidas == 0 {
			return "No se puede crear una partición lógica sin una partición extendida"
		}
	}

	return "" // Sin errores
}

// Estructura para representar espacios libres
type EspacioLibre struct {
	inicio int
	tamaño int
}

// calcularEspaciosLibres encuentra todos los espacios disponibles
func calcularEspaciosLibres(mbr Structs.MBR, particiones []Structs.Particion) []EspacioLibre {
	var espacios []EspacioLibre
	var ocupados []struct{ inicio, fin int }

	// Agregar el MBR como espacio ocupado
	mbrSize := int(unsafe.Sizeof(mbr))
	ocupados = append(ocupados, struct{ inicio, fin int }{0, mbrSize})

	// Agregar particiones activas como espacios ocupados
	for _, p := range particiones {
		if p.Part_status == '1' {
			ocupados = append(ocupados, struct{ inicio, fin int }{
				int(p.Part_start),
				int(p.Part_start + p.Part_size),
			})
		}
	}

	// Ordenar por posición de inicio (bubble sort simple)
	for i := 0; i < len(ocupados)-1; i++ {
		for j := 0; j < len(ocupados)-i-1; j++ {
			if ocupados[j].inicio > ocupados[j+1].inicio {
				ocupados[j], ocupados[j+1] = ocupados[j+1], ocupados[j]
			}
		}
	}

	// Encontrar espacios libres entre particiones
	ultimoFin := 0
	for _, ocu := range ocupados {
		if ocu.inicio > ultimoFin {
			// Hay espacio libre
			espacios = append(espacios, EspacioLibre{
				inicio: ultimoFin,
				tamaño: ocu.inicio - ultimoFin,
			})
		}
		if ocu.fin > ultimoFin {
			ultimoFin = ocu.fin
		}
	}

	// Espacio libre al final del disco
	if ultimoFin < int(mbr.Mbr_tamano) {
		espacios = append(espacios, EspacioLibre{
			inicio: ultimoFin,
			tamaño: int(mbr.Mbr_tamano) - ultimoFin,
		})
	}

	return espacios
}

// buscarEspacioConFit encuentra la mejor posición según la estrategia
func buscarEspacioConFit(espacios []EspacioLibre, tamaño int, fit string) int {
	var mejorEspacio *EspacioLibre
	var mejorIndice int = -1

	for i, espacio := range espacios {
		if espacio.tamaño >= tamaño {
			if mejorEspacio == nil {
				mejorEspacio = &espacio
				mejorIndice = i
				if Utils.Comparar(fit, "FF") {
					break // First Fit toma el primero
				}
			} else {
				switch fit {
				case "BF": // Best Fit - el más pequeño que sirva
					if espacio.tamaño < mejorEspacio.tamaño {
						mejorEspacio = &espacio
						mejorIndice = i
					}
				case "WF": // Worst Fit - el más grande
					if espacio.tamaño > mejorEspacio.tamaño {
						mejorEspacio = &espacio
						mejorIndice = i
					}
				}
			}
		}
	}

	if mejorIndice != -1 {
		return espacios[mejorIndice].inicio
	}
	return -1
}

// escribirMBR escribe el MBR al disco
func escribirMBR(path string, mbr Structs.MBR) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir archivo: %v", err)
	}
	defer file.Close()

	file.Seek(0, 0)

	// FIX: Usar BigEndian para mantener compatibilidad
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.BigEndian, &mbr); err != nil {
		return fmt.Errorf("error al serializar MBR: %v", err)
	}

	if _, err := file.Write(buffer.Bytes()); err != nil {
		return fmt.Errorf("error al escribir MBR: %v", err)
	}

	fmt.Printf("🔧 DEBUG: MBR escrito correctamente (%d bytes)\n", buffer.Len())
	return nil
}

// crearEBRInicial crea el EBR inicial para particiones extendidas
func crearEBRInicial(path string, start int) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	ebr := Structs.NewEBR()
	ebr.Part_start = int64(start)

	file.Seek(int64(start), 0)
	return binary.Write(file, binary.BigEndian, &ebr)
}

// Funciones auxiliares existentes (sin cambios críticos)
func leerDisco(path string) *Structs.MBR {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("❌ Error al abrir archivo: %v\n", err)
		return nil
	}
	defer file.Close()

	var mbr Structs.MBR
	file.Seek(0, 0)
	// FIX: Usar BigEndian para leer también
	err = binary.Read(file, binary.BigEndian, &mbr)
	if err != nil {
		fmt.Printf("❌ Error al leer MBR: %v\n", err)
		return nil
	}
	return &mbr
}

func getParticiones(disco Structs.MBR) []Structs.Particion {
	return []Structs.Particion{
		disco.Mbr_partition_1,
		disco.Mbr_partition_2,
		disco.Mbr_partition_3,
		disco.Mbr_partition_4,
	}
}

func buscarParticiones(mbr Structs.MBR, name string, path string) *Structs.Particion {
	particiones := getParticiones(mbr)

	for _, particion := range particiones {
		if particion.Part_status == '1' {
			nombre := Utils.ConvertirAString(particion.Part_name)
			if Utils.Comparar(nombre, name) {
				return &particion
			}
		}
	}
	return nil
}

// buscarParticionPorNombre busca una partición por nombre
func buscarParticionPorNombre(particiones []Structs.Particion, name string) *Structs.Particion {
	for _, particion := range particiones {
		if particion.Part_status == '1' {
			nombre := Utils.ConvertirAString(particion.Part_name)
			if Utils.Comparar(nombre, name) {
				return &particion
			}
		}
	}
	return nil
}

// FUNCIONES AUXILIARES PARA MANEJAR TIPOS ESPECÍFICOS

// convertirFitAString convierte [2]byte a string para Dsk_fit
func convertirFitAString(fit [2]byte) string {
	resultado := ""
	for _, b := range fit {
		if b != 0 {
			resultado += string(b)
		} else {
			break
		}
	}
	if resultado == "" {
		return "WF" // Por defecto Worst Fit
	}
	return resultado
}

// convertirFechaAString convierte [16]byte a string para fecha
func convertirFechaAString(fecha [16]byte) string {
	resultado := ""
	for _, b := range fecha {
		if b != 0 {
			resultado += string(b)
		} else {
			break
		}
	}
	if resultado == "" {
		return "No definida"
	}
	return resultado
}

// ListarParticiones muestra información de todas las particiones de un disco
func ListarParticiones(path string) string {
	if !Utils.ArchivoExiste(path) {
		return Utils.Error("FDISK", "El disco no existe en la ruta especificada")
	}

	// Leer MBR
	mbr := leerDisco(path)
	if mbr == nil {
		return Utils.Error("FDISK", "Error al leer el disco")
	}

	resultado := "\n📋 INFORMACIÓN DEL DISCO: " + path + "\n"
	resultado += "═════════════════════════════════════════════════════\n"
	resultado += fmt.Sprintf("🔧 Tamaño del disco: %s\n", Utils.FormatearTamaño(mbr.Mbr_tamano))
	resultado += fmt.Sprintf("🎯 Estrategia Fit: %s\n", convertirFitAString(mbr.Dsk_fit))
	resultado += fmt.Sprintf("📅 Fecha creación: %s\n", convertirFechaAString(mbr.Mbr_fecha_creacion))
	resultado += fmt.Sprintf("🔑 Signature: %d\n\n", mbr.Mbr_dsk_signature)

	// Obtener particiones
	particiones := getParticiones(*mbr)

	resultado += "📁 PARTICIONES:\n"
	resultado += "─────────────────────────────────────────────────────\n"

	espacioUsado := int64(unsafe.Sizeof(Structs.MBR{}))
	hayParticiones := false

	for i, particion := range particiones {
		if particion.Part_status == '1' {
			hayParticiones = true
			nombre := Utils.ConvertirAString(particion.Part_name)
			tipo := obtenerTipoParticion(particion.Part_type)
			fit := obtenerEstrategiaFit(particion.Part_fit)

			resultado += fmt.Sprintf("🔹 Partición %d:\n", i+1)
			resultado += fmt.Sprintf("   📛 Nombre: %s\n", nombre)
			resultado += fmt.Sprintf("   📊 Tipo: %s\n", tipo)
			resultado += fmt.Sprintf("   📏 Tamaño: %s\n", Utils.FormatearTamaño(particion.Part_size))
			resultado += fmt.Sprintf("   📍 Inicio: %d\n", particion.Part_start)
			resultado += fmt.Sprintf("   🎯 Fit: %s\n", fit)
			resultado += "   ✅ Estado: Activa\n"

			// Si es extendida, mostrar particiones lógicas
			if particion.Part_type == 'E' || particion.Part_type == 'e' {
				logicas := getLogicas(particion, path)
				if len(logicas) > 0 {
					resultado += "   📁 Particiones Lógicas:\n"
					for j, ebr := range logicas {
						if ebr.Part_status == '1' {
							nombreLogica := Utils.ConvertirAString(ebr.Part_name)
							resultado += fmt.Sprintf("      🔸 Lógica %d: %s (%s)\n",
								j+1, nombreLogica, Utils.FormatearTamaño(ebr.Part_size))
						}
					}
				}
			}
			resultado += "\n"
			espacioUsado += particion.Part_size
		}
	}

	if !hayParticiones {
		resultado += "❌ No hay particiones creadas en este disco.\n\n"
	}

	// Mostrar espacio libre
	espacioLibre := mbr.Mbr_tamano - espacioUsado
	porcentajeUsado := Utils.CalcularPorcentaje(int(espacioUsado), int(mbr.Mbr_tamano))

	resultado += "📊 RESUMEN DE ESPACIO:\n"
	resultado += "─────────────────────────────────────────────────────\n"
	resultado += fmt.Sprintf("💾 Espacio usado: %s (%.1f%%)\n",
		Utils.FormatearTamaño(espacioUsado), porcentajeUsado)
	resultado += fmt.Sprintf("🆓 Espacio libre: %s (%.1f%%)\n",
		Utils.FormatearTamaño(espacioLibre), 100-porcentajeUsado)

	return resultado
}

// Funciones auxiliares para formateo
func obtenerTipoParticion(tipo byte) string {
	switch tipo {
	case 'P', 'p':
		return "Primaria"
	case 'E', 'e':
		return "Extendida"
	case 'L', 'l':
		return "Lógica"
	default:
		return "Desconocido"
	}
}

func obtenerEstrategiaFit(fit byte) string {
	switch fit {
	case 'F', 'f':
		return "First Fit"
	case 'B', 'b':
		return "Best Fit"
	case 'W', 'w':
		return "Worst Fit"
	default:
		return "Desconocido"
	}
}

// formatearFecha - mantener la función original para compatibilidad
func formatearFecha(timestamp int64) string {
	if timestamp == 0 {
		return "No definida"
	}
	return fmt.Sprintf("%d", timestamp)
}

// DebugMBR muestra información básica del MBR para debugging
func DebugMBR(path string) string {
	mbr := leerDisco(path)
	if mbr == nil {
		return "❌ Error al leer MBR"
	}

	resultado := "🔍 DEBUG MBR:\n"
	resultado += fmt.Sprintf("Tamaño: %d bytes\n", mbr.Mbr_tamano)
	resultado += fmt.Sprintf("Signature: %d\n", mbr.Mbr_dsk_signature)

	particiones := getParticiones(*mbr)
	for i, p := range particiones {
		if p.Part_status == '1' {
			nombre := Utils.ConvertirAString(p.Part_name)
			resultado += fmt.Sprintf("P%d: %s (%d bytes, inicio: %d)\n",
				i+1, nombre, p.Part_size, p.Part_start)
		}
	}

	return resultado
}

// Funciones pendientes de implementar
func eliminarParticion(path, name string) string {
	return Utils.Mensaje("FDISK", "Funcionalidad de eliminación en desarrollo")
}

func addParticion(path, name, valor, unit string) string {
	return Utils.Mensaje("FDISK", "Funcionalidad de modificación en desarrollo")
}

func crearLogica(particion Structs.Particion, extended Structs.Particion, path string) string {
	return Utils.Mensaje("FDISK", "Funcionalidad de particiones lógicas en desarrollo")
}

func getLogicas(particion Structs.Particion, path string) []Structs.EBR {
	return []Structs.EBR{}
}
