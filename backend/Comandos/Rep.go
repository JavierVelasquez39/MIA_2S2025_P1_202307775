package Comandos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"godisk-backend/Structs"
	"godisk-backend/Utils"

	"github.com/awalterschulze/gographviz"
)

// Apuntadores stores pointers for generating reports
type Apuntadores struct {
	Inodos    []string
	Bloques   []string
	Direccion []string
}

// Ap is a global variable to store pointers
var Ap = Apuntadores{}

// ValidarDatosReporte validates data for reports and calls corresponding function
func ValidarDatosReporte(context []string) string {
	name := ""
	path := ""
	id := ""
	path_file_ls := ""

	// Extract parameters from context
	for i := 0; i < len(context); i++ {
		current := context[i]
		comando := strings.Split(current, "=")
		if len(comando) >= 2 {
			if Comparar(comando[0], "name") {
				name = comando[1]
			} else if Comparar(comando[0], "path") {
				path = strings.ReplaceAll(comando[1], "\"", "")
			} else if Comparar(comando[0], "id") {
				id = comando[1]
			} else if Comparar(comando[0], "path_file_ls") {
				path_file_ls = strings.ReplaceAll(comando[1], "\"", "")
			}
		}
	}

	// Check mandatory parameters
	if path == "" || name == "" || id == "" {
		return Error("REP", "Faltan parámetros obligatorios (name, path, id)")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return Error("REP", "No se pudo crear el directorio: "+err.Error())
		}
	}

	// Check report type and call corresponding function
	switch {
	case Comparar(name, "mbr"):
		MBR_R(path, id)
	case Comparar(name, "disk"):
		DISK_R_Simple(path, id)
	case Comparar(name, "inode"):
		Report_Inode(path, id)
	case Comparar(name, "block"):
		Report_Block(path, id)
	case Comparar(name, "bm_inode"):
		BitMap_inodo(path, id)
	case Comparar(name, "bm_block"):
		BitMap_block(path, id)
	case Comparar(name, "tree"):
		ReporteTree(path, id)
	case Comparar(name, "sb"):
		SB_Reporte(path, id)
	case Comparar(name, "file"):
		if path_file_ls == "" {
			return Error("REP", "Parámetro path_file_ls requerido para reporte file")
		}
		return File_Reporte(path, id, path_file_ls)
	case Comparar(name, "ls"):
		if path_file_ls == "" {
			return Error("REP", "Parámetro path_file_ls requerido para reporte ls")
		}
		LS_Reporte(path, id, path_file_ls)
	default:
		return Error("REP", "Nombre de reporte no válido")
	}

	// Return success message
	return Mensaje("REP", "Reporte generado correctamente: "+name)
}

// MBR_R generates a report for the MBR of a disk
func MBR_R(path string, id string) {
	// Find the disk path from the mounted partition list
	fmt.Println("Buscando disco para ID:", id)

	diskPath, found := GetDiskPathFromID(id)
	if !found {
		fmt.Println(Utils.Error("REP", "No se encontró el disco para la ID: "+id))
		return
	}

	fmt.Println("Disco encontrado en:", diskPath)

	// Read the MBR from the disk
	fmt.Println("Leyendo MBR desde:", diskPath)
	mbr := LeerDisco(diskPath)
	if mbr == nil {
		return
	}

	// Create a new Graphviz graph
	graphAst, _ := gographviz.ParseString(`digraph G {}`)
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		fmt.Println(Utils.Error("REP", "Error al crear el grafo: "+err.Error()))
		return
	}

	// Format MBR data for display
	mbrTamano := strconv.FormatInt(mbr.Mbr_tamano, 10)
	mbrFechaCreacion := string(bytes.Trim(mbr.Mbr_fecha_creacion[:], "\x00"))
	mbrDiskSignature := strconv.FormatInt(mbr.Mbr_dsk_signature, 10)

	// Create HTML table for MBR data
	Codigo_HTML := fmt.Sprintf(`<<TABLE>
    <TR style="background-color: #4B0082; color: white;">
        <TD BGCOLOR="#4B0082">
            <FONT COLOR="white">Reporte MBR</FONT>
        </TD>
        <TD BGCOLOR="#4B0082">
            <FONT COLOR="white">Datos</FONT>
        </TD>
    </TR>
    <TR>
        <TD>Mbr_tamano</TD><TD>%s</TD>
    </TR>
    <TR>
        <TD>Mbr_fecha_creacion</TD><TD>%s</TD>
    </TR>
    <TR>
        <TD>Mbr_disk_signature</TD><TD>%s</TD>
    </TR>
    `, mbrTamano, mbrFechaCreacion, mbrDiskSignature)

	// Add partition information
	particiones := GetParticiones(*mbr)
	for i := 0; i < len(particiones); i++ {
		particion := particiones[i]
		Codigo_HTML += fmt.Sprintf(`
        <TR style="background-color: #4B0082; color: white;">
            <TD BGCOLOR="#4B0082">
                <FONT COLOR="white">Particion</FONT>
            </TD>
            <TD BGCOLOR="#4B0082">
                
            </TD>
        </TR>
        `)
		if particion.Part_type != 'E' {
			TipoParticion := string(particion.Part_type)
			PartStatus := string(particion.Part_status)
			PartStart := strconv.FormatInt(particion.Part_start, 10)
			PartSize := strconv.FormatInt(particion.Part_size, 10)
			PartFit := string(particion.Part_fit)

			PartName := string(bytes.Trim(particion.Part_name[:], "\x00"))

			Codigo_HTML += fmt.Sprintf(`
            <TR>
                <TD>Part_status</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_type</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_fit</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_start</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_size</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_name</TD><TD>%s</TD>
            </TR>
        `, PartStatus, TipoParticion, PartFit, PartStart, PartSize, PartName)
		} else {
			TipoParticion := string(particion.Part_type)
			PartStatus := string(particion.Part_status)
			PartStart := strconv.FormatInt(particion.Part_start, 10)
			PartSize := strconv.FormatInt(particion.Part_size, 10)
			PartFit := string(particion.Part_fit)

			PartName := string(bytes.Trim(particion.Part_name[:], "\x00"))

			Codigo_HTML += fmt.Sprintf(`
            <TR>
                <TD>Part_status</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_type</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_fit</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_start</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_size</TD><TD>%s</TD>
            </TR>
            <TR>
                <TD>Part_name</TD><TD>%s</TD>
            </TR>
            `, PartStatus, TipoParticion, PartFit, PartStart, PartSize, PartName)

			// Get logical partitions within this extended partition
			ebrs := GetLogicas(particion, diskPath)
			for j := 0; j < len(ebrs); j++ {
				ebr := ebrs[j]
				Codigo_HTML += fmt.Sprintf(`
                <TR style="background-color: #FA8072; color: white;">
                    <TD BGCOLOR="#FA8072">
                        <FONT COLOR="white">Partición logica</FONT>
                    </TD>
                    <TD BGCOLOR="#FA8072">
                        
                    </TD>
                </TR>
                `)

				TipoParticion_Logica := "L"
				PartStatus_Logica := string(ebr.Part_status)
				PartStart_Logica := strconv.FormatInt(ebr.Part_start, 10)
				PartSize_Logica := strconv.FormatInt(ebr.Part_size, 10)
				PartFit_Logica := string(ebr.Part_fit)

				PartName_Logica := string(bytes.Trim(ebr.Part_name[:], "\x00"))

				Codigo_HTML += fmt.Sprintf(`
                <TR>
                    <TD>Part_status</TD><TD>%s</TD>
                </TR>
                <TR>
                    <TD>Part_type</TD><TD>%s</TD>
                </TR>
                <TR>
                    <TD>Part_fit</TD><TD>%s</TD>
                </TR>
                <TR>
                    <TD>Part_start</TD><TD>%s</TD>
                </TR>
                <TR>
                    <TD>Part_size</TD><TD>%s</TD>
                </TR>
                <TR>
                    <TD>Part_name</TD><TD>%s</TD>
                </TR>
                `, PartStatus_Logica, TipoParticion_Logica, PartFit_Logica, PartStart_Logica, PartSize_Logica, PartName_Logica)
			}
		}
	}

	// Close the HTML table
	Codigo_HTML += fmt.Sprintf(`</TABLE>>`)

	// Add the node to the graph
	graph.AddNode("G", "a", map[string]string{"label": Codigo_HTML, "shape": "plaintext"})

	// Create .dot file
	dotPath := path + ".dot"
	fmt.Println("Generando archivo DOT:", dotPath)
	err := os.WriteFile(dotPath, []byte(graph.String()), 0644)
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error al escribir archivo .dot: "+err.Error()))
		return
	}

	// Generate the PNG file using Graphviz
	pngPath := path
	fmt.Println("Generando imagen PNG:", pngPath)
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", pngPath)
	fmt.Println("Ejecutando comando:", "dot", "-Tpng", dotPath, "-o", pngPath)
	err = cmd.Run()
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error al generar imagen PNG: "+err.Error()))
		return
	}

	// Remove temporary .dot file
	os.Remove(dotPath)
}

// GetDiskPathFromID finds the disk path for a given partition ID by checking the mounted disks
func GetDiskPathFromID(id string) (string, bool) {
	// Get all mounted disks and partitions
	mounted := GetMountedPartitions()

	// Check if this ID is in the mounted partitions
	for _, mount := range mounted {
		if mount.Id == id {
			return mount.Path, true
		}
	}

	return "", false
}

// MountedPartition represents a mounted partition with its path and ID
type MountedPartition struct {
	Id   string
	Path string
}

// GetMountedPartitions returns all currently mounted partitions
func GetMountedPartitions() []MountedPartition {
	fmt.Println("🔍 Buscando en particiones montadas")

	var result []MountedPartition

	// Iterate through all disks in DiscMont
	for i := 0; i < 99; i++ {
		if DiscMont[i].Estado == 1 { // Only check active disks
			diskPath := string(bytes.Trim(DiscMont[i].Path[:], "\x00"))

			// Check all partitions in this disk
			for j := 0; j < 26; j++ {
				if DiscMont[i].Particiones[j].Estado == 1 { // Only check active partitions
					idParticion := string(bytes.Trim(DiscMont[i].Particiones[j].Id_Particion[:], "\x00"))

					fmt.Printf("   - Disco %d, Slot %d: ID='%s', Path='%s'\n",
						i, j, idParticion, diskPath)

					result = append(result, MountedPartition{
						Id:   idParticion,
						Path: diskPath,
					})
				}
			}
		}
	}

	return result
}

func LeerDisco(path string) *Structs.MBR {

	file, err := os.Open(path)
	if err != nil {
		fmt.Println(Utils.Error("REP", "No se puede abrir el archivo del disco: "+err.Error()))
		return nil
	}
	defer file.Close()

	mbr := Structs.MBR{}
	file.Seek(0, 0)
	err = binary.Read(file, binary.LittleEndian, &mbr)
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error al leer el MBR: "+err.Error()))
		return nil
	}

	return &mbr
}

func GetParticiones(mbr Structs.MBR) []Structs.Particion {
	particiones := make([]Structs.Particion, 0)

	// Check each partition
	if mbr.Mbr_partition_1.Part_status == '1' {
		particiones = append(particiones, mbr.Mbr_partition_1)
	}
	if mbr.Mbr_partition_2.Part_status == '1' {
		particiones = append(particiones, mbr.Mbr_partition_2)
	}
	if mbr.Mbr_partition_3.Part_status == '1' {
		particiones = append(particiones, mbr.Mbr_partition_3)
	}
	if mbr.Mbr_partition_4.Part_status == '1' {
		particiones = append(particiones, mbr.Mbr_partition_4)
	}

	return particiones
}

func GetLogicas(extendida Structs.Particion, diskPath string) []Structs.EBR {
	ebrs := make([]Structs.EBR, 0)

	file, err := os.Open(diskPath)
	if err != nil {
		fmt.Println(Utils.Error("REP", "No se puede abrir el archivo del disco: "+err.Error()))
		return ebrs
	}
	defer file.Close()

	currentPos := extendida.Part_start

	for currentPos >= 0 {
		ebr := Structs.EBR{}
		file.Seek(currentPos, 0)
		err := binary.Read(file, binary.LittleEndian, &ebr)
		if err != nil {
			break
		}

		if ebr.Part_status == '1' {
			ebrs = append(ebrs, ebr)
		}

		// Move to the next EBR, if any
		if ebr.Part_next <= 0 {
			break
		}
		currentPos = ebr.Part_next
	}

	return ebrs
}

func Comparar(a, b string) bool {
	return Utils.Comparar(a, b)
}

func Error(comando, mensaje string) string {
	return Utils.Error(comando, mensaje)
}

func Mensaje(comando, mensaje string) string {
	return Utils.Mensaje(comando, mensaje)
}

// DISK_R genera un reporte gráfico de la estructura del disco
func DISK_R(path string, id string) {
	// Obtener la ruta del disco a partir del ID de la partición montada
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		fmt.Println(Utils.Error("REP", "No se encontró el disco para la ID: "+id))
		return
	}

	fmt.Println("Generando reporte DISK para:", diskPath)

	// Leer el MBR del disco
	mbr := LeerDisco(diskPath)
	if mbr == nil {
		return
	}

	// Crear un nuevo grafo de Graphviz
	graphAst, _ := gographviz.ParseString(`digraph G {
        rankdir=LR;
        node [shape=record, style=filled];
        bgcolor="#FFFFFF";
    }`)
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		fmt.Println(Utils.Error("REP", "Error al crear el grafo: "+err.Error()))
		return
	}

	// Obtener información del disco
	tamanoTotal := mbr.Mbr_tamano
	tamanoMBR := int64(binary.Size(mbr))

	// Obtener todas las particiones activas
	particiones := GetParticiones(*mbr)

	// Crear una lista de todas las regiones (MBR, particiones, espacios libres)
	type Region struct {
		Tipo       string // "MBR", "Primaria", "Extendida", "Lógica", "EBR", "Libre"
		Nombre     string
		Inicio     int64
		Tamano     int64
		Porcentaje float64
		Color      string
		Padre      *Region // Para particiones lógicas dentro de extendidas
	}

	var regiones []Region

	// Añadir MBR como primera región
	regiones = append(regiones, Region{
		Tipo:       "MBR",
		Nombre:     "MBR",
		Inicio:     0,
		Tamano:     tamanoMBR,
		Porcentaje: float64(tamanoMBR) * 100.0 / float64(tamanoTotal),
		Color:      "#E6E6FA", // Lavender
	})

	// Crear un mapa para rastrear el espacio ocupado
	espacioOcupado := make(map[int64]int64)
	espacioOcupado[0] = tamanoMBR // MBR ocupa desde 0 hasta tamanoMBR

	// Añadir particiones
	var regionesExtendidas []*Region

	for _, p := range particiones {
		if p.Part_status == '1' {
			nombre := string(bytes.Trim(p.Part_name[:], "\x00"))
			inicio := p.Part_start
			fin := inicio + p.Part_size

			var tipo, color string

			switch p.Part_type {
			case 'P', 'p':
				tipo = "Primaria"
				color = "#90EE90" // LightGreen
			case 'E', 'e':
				tipo = "Extendida"
				color = "#ADD8E6" // LightBlue
			case 'L', 'l':
				tipo = "Lógica"
				color = "#FFB6C1" // LightPink
			}

			porcentaje := float64(p.Part_size) * 100.0 / float64(tamanoTotal)

			region := Region{
				Tipo:       tipo,
				Nombre:     nombre,
				Inicio:     inicio,
				Tamano:     p.Part_size,
				Porcentaje: porcentaje,
				Color:      color,
			}

			regiones = append(regiones, region)
			espacioOcupado[inicio] = fin

			// Si es extendida, guardarla para procesar sus particiones lógicas después
			if p.Part_type == 'E' || p.Part_type == 'e' {
				regionIndex := len(regiones) - 1
				regionesExtendidas = append(regionesExtendidas, &regiones[regionIndex])
			}
		}
	}

	// Procesar particiones lógicas dentro de extendidas
	for _, extendidaPtr := range regionesExtendidas {
		extendida := *extendidaPtr

		// Buscar la partición extendida en el mbr
		var partExt Structs.Particion
		for _, p := range particiones {
			if p.Part_status == '1' && (p.Part_type == 'E' || p.Part_type == 'e') && p.Part_start == extendida.Inicio {
				partExt = p
				break
			}
		}

		// Leer todas las particiones lógicas
		logicas := GetLogicas(partExt, diskPath)

		// Si hay particiones lógicas
		if len(logicas) > 0 {
			var posicionActual int64 = extendida.Inicio

			for _, ebr := range logicas {
				if ebr.Part_status == '1' {
					nombreLogica := string(bytes.Trim(ebr.Part_name[:], "\x00"))

					// Añadir EBR
					tamanoEBR := int64(binary.Size(Structs.EBR{}))
					ebrPorcentaje := float64(tamanoEBR) * 100.0 / float64(tamanoTotal)

					regiones = append(regiones, Region{
						Tipo:       "EBR",
						Nombre:     "EBR",
						Inicio:     ebr.Part_start - tamanoEBR,
						Tamano:     tamanoEBR,
						Porcentaje: ebrPorcentaje,
						Color:      "#FFD700", // Gold
						Padre:      extendidaPtr,
					})

					// Añadir partición lógica
					logicaPorcentaje := float64(ebr.Part_size) * 100.0 / float64(tamanoTotal)

					regiones = append(regiones, Region{
						Tipo:       "Lógica",
						Nombre:     nombreLogica,
						Inicio:     ebr.Part_start,
						Tamano:     ebr.Part_size,
						Porcentaje: logicaPorcentaje,
						Color:      "#FFB6C1", // LightPink
						Padre:      extendidaPtr,
					})

					posicionActual = ebr.Part_start + ebr.Part_size
				}
			}

			// Verificar si hay espacio libre al final de la extendida
			finExtendida := extendida.Inicio + extendida.Tamano
			if posicionActual < finExtendida {
				espacioLibre := finExtendida - posicionActual
				librePorcentaje := float64(espacioLibre) * 100.0 / float64(tamanoTotal)

				regiones = append(regiones, Region{
					Tipo:       "Libre en Extendida",
					Nombre:     "Libre",
					Inicio:     posicionActual,
					Tamano:     espacioLibre,
					Porcentaje: librePorcentaje,
					Color:      "#F0E68C", // Khaki
					Padre:      extendidaPtr,
				})
			}
		} else {
			// No hay particiones lógicas, toda la extendida está libre
			librePorcentaje := extendida.Porcentaje

			regiones = append(regiones, Region{
				Tipo:       "Libre en Extendida",
				Nombre:     "Libre",
				Inicio:     extendida.Inicio,
				Tamano:     extendida.Tamano,
				Porcentaje: librePorcentaje,
				Color:      "#F0E68C", // Khaki
				Padre:      extendidaPtr,
			})
		}
	}

	// Ordenar regiones por posición de inicio
	sort.Slice(regiones, func(i, j int) bool {
		return regiones[i].Inicio < regiones[j].Inicio
	})

	// Identificar espacios libres entre particiones
	var regionesFinales []Region
	var ultimaPos int64 = 0

	for _, r := range regiones {
		// Si hay un hueco entre la última posición y el inicio de esta región
		if r.Inicio > ultimaPos {
			espacioLibre := r.Inicio - ultimaPos
			librePorcentaje := float64(espacioLibre) * 100.0 / float64(tamanoTotal)

			// Añadir región de espacio libre
			regionesFinales = append(regionesFinales, Region{
				Tipo:       "Libre",
				Nombre:     "Libre",
				Inicio:     ultimaPos,
				Tamano:     espacioLibre,
				Porcentaje: librePorcentaje,
				Color:      "#DCDCDC", // Gainsboro (gris claro)
			})
		}

		// Añadir la región actual
		regionesFinales = append(regionesFinales, r)
		ultimaPos = r.Inicio + r.Tamano
	}

	// Verificar si hay espacio libre al final del disco
	if ultimaPos < tamanoTotal {
		espacioLibre := tamanoTotal - ultimaPos
		librePorcentaje := float64(espacioLibre) * 100.0 / float64(tamanoTotal)

		regionesFinales = append(regionesFinales, Region{
			Tipo:       "Libre",
			Nombre:     "Libre",
			Inicio:     ultimaPos,
			Tamano:     espacioLibre,
			Porcentaje: librePorcentaje,
			Color:      "#DCDCDC", // Gainsboro
		})
	}

	// Crear la representación visual
	discoLabel := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">
    <TR>
        <TD COLSPAN="%d" BGCOLOR="#4B0082"><FONT COLOR="white">Reporte DISK: %s</FONT></TD>
    </TR>
    <TR>`, len(regionesFinales), filepath.Base(diskPath))

	// Añadir cada región al reporte - ESTA ES LA CORRECCIÓN CLAVE
	for _, r := range regionesFinales {
		ancho := int(math.Max(1, math.Round(r.Porcentaje*2))) // Escalar el ancho según el porcentaje

		// Usamos "p" como prefijo para el PORT para evitar conflictos con valores numéricos
		discoLabel += fmt.Sprintf(`<TD WIDTH="%d" BGCOLOR="%s" PORT="p%d">
            <TABLE BORDER="0" CELLBORDER="0" CELLSPACING="0">
                <TR><TD><FONT POINT-SIZE="10"><B>%s</B></FONT></TD></TR>
                <TR><TD><FONT POINT-SIZE="9">%s</FONT></TD></TR>
                <TR><TD><FONT POINT-SIZE="8">%.1f%%</FONT></TD></TR>
            </TABLE>
        </TD>`,
			ancho,
			r.Color,
			int(r.Inicio), // Prefijamos con 'p' para evitar problemas con valores numéricos
			r.Tipo,
			r.Nombre,
			r.Porcentaje)
	}

	discoLabel += "</TR></TABLE>>"

	// Añadir el nodo disco al gráfico
	graph.AddNode("G", "disco", map[string]string{
		"shape":     "plaintext",
		"label":     discoLabel,
		"fontname":  "Arial",
		"fontsize":  "12",
		"fontcolor": "#333333",
	})

	// Añadir leyenda
	leyendaLabel := `<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">
        <TR><TD COLSPAN="2" BGCOLOR="#4B0082"><FONT COLOR="white">Leyenda</FONT></TD></TR>
        <TR><TD BGCOLOR="#E6E6FA">MBR</TD><TD>Master Boot Record</TD></TR>
        <TR><TD BGCOLOR="#90EE90">Primaria</TD><TD>Partición Primaria</TD></TR>
        <TR><TD BGCOLOR="#ADD8E6">Extendida</TD><TD>Partición Extendida</TD></TR>
        <TR><TD BGCOLOR="#FFD700">EBR</TD><TD>Extended Boot Record</TD></TR>
        <TR><TD BGCOLOR="#FFB6C1">Lógica</TD><TD>Partición Lógica</TD></TR>
        <TR><TD BGCOLOR="#DCDCDC">Libre</TD><TD>Espacio sin asignar</TD></TR>
        <TR><TD BGCOLOR="#F0E68C">Libre en Ext.</TD><TD>Espacio libre en extendida</TD></TR>
    </TABLE>>`

	graph.AddNode("G", "leyenda", map[string]string{
		"shape":     "plaintext",
		"label":     leyendaLabel,
		"fontname":  "Arial",
		"fontsize":  "10",
		"fontcolor": "#333333",
	})

	// Crear archivo .dot
	dotPath := path + ".dot"
	err := os.WriteFile(dotPath, []byte(graph.String()), 0644)
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error al escribir archivo .dot: "+err.Error()))
		return
	}

	// Generar la imagen PNG con mejor manejo de errores
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error al generar imagen: "+err.Error()))
		fmt.Println("Output de Graphviz:", string(output))
		fmt.Println("Verificando instalación de Graphviz...")

		// Verificar si dot está instalado
		checkCmd := exec.Command("which", "dot")
		if checkErr := checkCmd.Run(); checkErr != nil {
			fmt.Println(Utils.Error("REP", "Graphviz no está instalado. Por favor instálelo con: sudo apt-get install graphviz"))
		} else {
			fmt.Println("Graphviz está instalado, pero hubo un error al generar la imagen.")
			fmt.Println("El archivo .dot fue generado correctamente en:", dotPath)
			fmt.Println("Puede intentar generar la imagen manualmente con: dot -Tpng", dotPath, "-o", path)
		}

		// No borrar el archivo .dot en caso de error para permitir diagnóstico
		return
	}

	// Eliminar archivo .dot temporal
	os.Remove(dotPath)

	fmt.Println(Utils.Mensaje("REP", "Reporte DISK generado correctamente"))
}

// DISK_R_Simple genera un reporte de disco más simple y compatible
func DISK_R_Simple(path string, id string) {
	// Obtener la ruta del disco
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		fmt.Println(Utils.Error("REP", "No se encontró el disco para la ID: "+id))
		return
	}

	fmt.Println("Generando reporte DISK (versión simple) para:", diskPath)

	// Leer el MBR del disco
	mbr := LeerDisco(diskPath)
	if mbr == nil {
		return
	}

	// Obtener información del disco
	tamanoTotal := mbr.Mbr_tamano

	// Crear contenido DOT manualmente como string para máximo control y evitar errores
	dotContent := "digraph G {\n"
	dotContent += "  rankdir=LR;\n" // Asegurar disposición horizontal
	dotContent += "  bgcolor=\"#FFFFFF\";\n"
	dotContent += "  node [shape=box style=filled];\n"

	// Título
	dotContent += "  titulo [shape=plaintext label=\"Reporte DISK: " + filepath.Base(diskPath) + "\"];\n\n"

	// Crear subgrafo para la estructura del disco
	dotContent += "  subgraph cluster_disk {\n"
	dotContent += "    label=\"Estructura del Disco\";\n"
	dotContent += "    style=filled;\n"
	dotContent += "    color=\"#F0F0F0\";\n"
	dotContent += "    rank=same;\n" // Forzar que todos los nodos principales estén en la misma fila

	// MBR al inicio
	dotContent += "    mbr [label=\"MBR\\n0.1%\" fillcolor=\"#E6E6FA\" height=0.7 width=0.5];\n"

	// Obtener todas las particiones
	particiones := GetParticiones(*mbr)

	// Variables para nodos y conexiones
	var mainNodes []string = []string{"mbr"}
	var extendidaID string = ""
	var extendida Structs.Particion
	var hayExtendida bool = false

	// Generar nodos para particiones primarias y extendidas
	for i, p := range particiones {
		if p.Part_status != '1' {
			continue
		}

		nombre := string(bytes.Trim(p.Part_name[:], "\x00"))
		porcentaje := float64(p.Part_size) * 100.0 / float64(tamanoTotal)

		// Calcular ancho proporcional al tamaño
		ancho := math.Max(1.0, porcentaje/10.0)

		nodeID := fmt.Sprintf("part_%d", i)

		switch p.Part_type {
		case 'P', 'p':
			// Partición primaria
			dotContent += fmt.Sprintf("    %s [label=\"Primaria\\n%s\\n%.1f%%\" fillcolor=\"#90EE90\" width=%.1f];\n",
				nodeID, nombre, porcentaje, ancho)
			mainNodes = append(mainNodes, nodeID)

		case 'E', 'e':
			// Partición extendida - Se procesará después para incluir particiones lógicas
			extendidaID = nodeID
			extendida = p
			hayExtendida = true

			// Solo creamos el nodo de la extendida, las lógicas las añadiremos después
			dotContent += fmt.Sprintf("    %s [label=\"Extendida\\n%s\\n%.1f%%\" fillcolor=\"#ADD8E6\" width=%.1f];\n",
				nodeID, nombre, porcentaje, ancho)
			mainNodes = append(mainNodes, nodeID)
		}
	}

	// Espacio libre al final si existe
	var espacioUsado int64 = int64(binary.Size(*mbr))
	for _, p := range particiones {
		if p.Part_status == '1' {
			espacioUsado += p.Part_size
		}
	}

	var espacioLibre int64 = tamanoTotal - espacioUsado
	if espacioLibre > 0 {
		porcentajeLibre := float64(espacioLibre) * 100.0 / float64(tamanoTotal)
		anchoLibre := math.Max(1.0, porcentajeLibre/10.0)

		dotContent += fmt.Sprintf("    libre [label=\"Libre\\n%.1f%%\" fillcolor=\"#DCDCDC\" width=%.1f];\n",
			porcentajeLibre, anchoLibre)
		mainNodes = append(mainNodes, "libre")
	}

	// Crear conexiones invisibles entre todos los nodos principales para mantenerlos en orden
	for i := 0; i < len(mainNodes)-1; i++ {
		dotContent += fmt.Sprintf("    %s -> %s [style=invis];\n", mainNodes[i], mainNodes[i+1])
	}

	// Cerrar el subgraph principal
	dotContent += "  }\n\n"

	// Si hay una partición extendida, crear un subgrafo especial para sus particiones lógicas
	if hayExtendida && extendidaID != "" {
		logicas := GetLogicas(extendida, diskPath)

		if len(logicas) > 0 {
			// Crear subgrafo para particiones lógicas, conectado visualmente a la extendida
			dotContent += "  subgraph cluster_logicas {\n"
			dotContent += "    margin=10;\n"
			dotContent += "    color=\"#ADD8E6\";\n"
			dotContent += "    style=filled;\n"
			dotContent += "    rank=same;\n"
			dotContent += fmt.Sprintf("    label=\"Particiones Lógicas en %s\";\n",
				string(bytes.Trim(extendida.Part_name[:], "\x00")))

			var nodosLogicos []string
			var posActual int64 = extendida.Part_start

			// Procesar todas las particiones lógicas
			for j, ebr := range logicas {
				if ebr.Part_status == '1' {
					// Añadir EBR
					ebrID := fmt.Sprintf("ebr_%d", j)
					dotContent += fmt.Sprintf("    %s [label=\"EBR\" fillcolor=\"#FFD700\" width=0.5 height=0.5];\n", ebrID)
					nodosLogicos = append(nodosLogicos, ebrID)

					// Añadir partición lógica
					nombreLogica := string(bytes.Trim(ebr.Part_name[:], "\x00"))
					porcentajeLogica := float64(ebr.Part_size) * 100.0 / float64(tamanoTotal)
					anchoLogica := math.Max(1.0, porcentajeLogica/10.0)

					logicaID := fmt.Sprintf("logica_%d", j)
					dotContent += fmt.Sprintf("    %s [label=\"Lógica\\n%s\\n%.1f%%\" fillcolor=\"#FFB6C1\" width=%.1f];\n",
						logicaID, nombreLogica, porcentajeLogica, anchoLogica)
					nodosLogicos = append(nodosLogicos, logicaID)

					// ELIMINADO: Conectar EBR con su partición lógica
					// dotContent += fmt.Sprintf("    %s -> %s;\n", ebrID, logicaID)

					// Actualizar posición actual
					posActual = ebr.Part_start + ebr.Part_size
				}
			}

			// Espacio libre dentro de extendida si existe
			var espacioLibreExt = (extendida.Part_start + extendida.Part_size) - posActual
			if espacioLibreExt > 0 {
				porcentajeLibreExt := float64(espacioLibreExt) * 100.0 / float64(tamanoTotal)
				anchoLibreExt := math.Max(1.0, porcentajeLibreExt/10.0)

				dotContent += fmt.Sprintf("    libre_ext [label=\"Libre\\n%.1f%%\" fillcolor=\"#F0E68C\" width=%.1f];\n",
					porcentajeLibreExt, anchoLibreExt)

				// ELIMINADO: Conectar el último nodo lógico con el espacio libre
				// if len(nodosLogicos) > 0 {
				//    dotContent += fmt.Sprintf("    %s -> libre_ext [style=dashed];\n",
				//        nodosLogicos[len(nodosLogicos)-1])
				// }
			}

			// Para mantener la disposición horizontal dentro del subgrafo de lógicas
			// Crear conexiones invisibles entre nodos para mantener el orden
			for i := 0; i < len(nodosLogicos)-1; i++ {
				dotContent += fmt.Sprintf("    %s -> %s [style=invis];\n", nodosLogicos[i], nodosLogicos[i+1])
			}

			// Si hay espacio libre en la extendida, añadirlo a las conexiones invisibles
			if espacioLibreExt > 0 && len(nodosLogicos) > 0 {
				dotContent += fmt.Sprintf("    %s -> libre_ext [style=invis];\n",
					nodosLogicos[len(nodosLogicos)-1])
			}

			// Cerrar el subgrafo de particiones lógicas
			dotContent += "  }\n\n"

			// Conectar la partición extendida con la primera lógica/EBR si existe
			if len(nodosLogicos) > 0 {
				dotContent += fmt.Sprintf("  %s -> %s [ltail=cluster_disk lhead=cluster_logicas style=dashed];\n",
					extendidaID, nodosLogicos[0])
			}
		}
	}

	// Cerrar el grafo
	dotContent += "}\n"

	// Escribir el archivo DOT
	dotPath := path + ".dot"
	err := os.WriteFile(dotPath, []byte(dotContent), 0644)
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error al escribir archivo .dot: "+err.Error()))
		return
	}

	// Ejecutar dot para generar la imagen
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error al generar imagen: "+err.Error()))
		fmt.Println("Output:", string(output))

		// Intentar generar SVG como alternativa
		svgPath := path + ".svg"
		cmdSVG := exec.Command("dot", "-Tsvg", dotPath, "-o", svgPath)
		if err := cmdSVG.Run(); err == nil {
			fmt.Println(Utils.Mensaje("REP", "Reporte generado en formato SVG: "+svgPath))
		} else {
			fmt.Println(Utils.Error("REP", "También falló la generación SVG"))
			fmt.Println("Puede intentar generar la imagen manualmente con:")
			fmt.Println("dot -Tpng", dotPath, "-o", path)
		}
		return
	}

	fmt.Println(Utils.Mensaje("REP", "Reporte DISK generado correctamente"))
	os.Remove(dotPath) // Eliminar archivo DOT temporal
}

// ReporteTree genera un reporte gráfico de la estructura jerárquica del sistema de archivos
func ReporteTree(path string, id string) string {
	fmt.Println("Generando reporte tree para la partición", id)

	// Obtener la ruta del disco
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		return Utils.Error("REP", "No se encontró el disco para la ID: "+id)
	}

	// Abrir el archivo del disco
	file, err := os.Open(diskPath)
	if err != nil {
		return Utils.Error("REP", "Error abriendo disco: "+err.Error())
	}
	defer file.Close()

	// Obtener la partición montada
	particion := GetMount("REP", id, &diskPath)
	if particion == nil {
		return Utils.Error("REP", "No se encontró la partición montada")
	}

	// Leer el superbloque
	var sb Structs.SuperBloque
	file.Seek(particion.Part_start, 0)
	if err := binary.Read(file, binary.LittleEndian, &sb); err != nil {
		return Utils.Error("REP", "Error leyendo superbloque: "+err.Error())
	}

	// Crear contenido DOT
	var dot strings.Builder
	dot.WriteString("digraph G {\n")
	// Cambiar de TB (top-bottom) a LR (left-right) para disposición horizontal
	dot.WriteString("  graph [rankdir=LR, nodesep=0.7, ranksep=0.9];\n")
	dot.WriteString("  node [shape=record];\n")
	dot.WriteString("  bgcolor=\"#FFFFFF\";\n")
	dot.WriteString("  edge [color=\"#1976D2\"];\n\n")

	// Título del reporte
	dot.WriteString("  label=\"Reporte Tree - Sistema de Archivos EXT2\";\n")
	dot.WriteString("  labelloc=\"t\";\n")
	dot.WriteString("  fontsize=20;\n\n")

	// Registro de inodos procesados para evitar ciclos
	procesados := make(map[int64]bool)

	// Mapa de bloques para evitar duplicados
	bloquesProcesados := make(map[int64]bool)

	// Comenzar desde el inodo raíz (0)
	dot.WriteString(procesarInodo(file, sb, 0, &procesados, &bloquesProcesados))

	dot.WriteString("}\n")

	// Escribir archivo DOT
	dotPath := path + ".dot"
	if err := os.WriteFile(dotPath, []byte(dot.String()), 0644); err != nil {
		return Utils.Error("REP", "Error escribiendo archivo DOT: "+err.Error())
	}

	// Generar PNG usando Graphviz
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("Error generando imagen:", err)
		fmt.Println("Output:", string(output))
		return Utils.Error("REP", "Error generando imagen PNG: "+err.Error())
	}

	// Eliminar archivo DOT temporal si no hay errores
	os.Remove(dotPath)

	return Utils.Mensaje("REP", "Reporte Tree generado exitosamente en: "+path)
}

// procesarInodo procesa recursivamente un inodo y sus bloques asociados
func procesarInodo(file *os.File, sb Structs.SuperBloque, inodeIdx int64, procesados *map[int64]bool, bloquesProcesados *map[int64]bool) string {
	// Evitar procesar inodos ya procesados
	if (*procesados)[inodeIdx] {
		return ""
	}

	// Marcar inodo como procesado
	(*procesados)[inodeIdx] = true

	// Leer inodo
	var inodo Structs.Inodos
	file.Seek(sb.S_inode_start+inodeIdx*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &inodo); err != nil {
		fmt.Println("Error leyendo inodo:", err)
		return ""
	}

	// Verificar si el inodo está en uso
	if inodo.I_size == -1 {
		return "" // Inodo no asignado
	}

	var result strings.Builder

	// Determinar el tipo de inodo (directorio o archivo)
	tipoInodo := "Archivo"
	if inodo.I_type == 0 {
		tipoInodo = "Directorio"
	}

	// Crear tabla HTML para inodo con puertos específicos para cada valor numérico
	result.WriteString(fmt.Sprintf("  inode_%d [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n", inodeIdx))
	result.WriteString(fmt.Sprintf("    <TR><TD BGCOLOR=\"#5D4037\" COLSPAN=\"2\"><FONT COLOR=\"white\">INODO %d - %s</FONT></TD></TR>\n", inodeIdx, tipoInodo))
	result.WriteString(fmt.Sprintf("    <TR><TD>UID</TD><TD>%d</TD></TR>\n", inodo.I_uid))
	result.WriteString(fmt.Sprintf("    <TR><TD>GID</TD><TD>%d</TD></TR>\n", inodo.I_gid))
	result.WriteString(fmt.Sprintf("    <TR><TD>SIZE</TD><TD>%d</TD></TR>\n", inodo.I_size))
	result.WriteString(fmt.Sprintf("    <TR><TD>TIPO</TD><TD>%d</TD></TR>\n", inodo.I_type))
	result.WriteString(fmt.Sprintf("    <TR><TD>PERMISOS</TD><TD>%d</TD></TR>\n", inodo.I_perm))

	// Procesamiento: Mostrar todos los apuntadores con puertos específicos en los VALORES
	for i := 0; i < 15; i++ {
		blockLabel := fmt.Sprintf("BLOCK %d", i)
		if i >= 12 {
			switch i {
			case 12:
				blockLabel = "BLOCK 12 (I1)"
			case 13:
				blockLabel = "BLOCK 13 (I2)"
			case 14:
				blockLabel = "BLOCK 14 (I3)"
			}
		} else {
			blockLabel = fmt.Sprintf("BLOCK %d (D)", i)
		}

		// CLAVE: El puerto va SOLO en la celda del valor numérico
		result.WriteString(fmt.Sprintf("    <TR><TD>%s</TD><TD PORT=\"block_%d\">%d</TD></TR>\n",
			blockLabel, i, inodo.I_block[i]))
	}
	result.WriteString("  </TABLE>>];\n\n")

	// Procesamiento: Usar los puertos correctos en las conexiones
	// Procesar bloques directos (0-11)
	for i := 0; i < 12; i++ {
		if inodo.I_block[i] != -1 {
			// La conexión sale del puerto específico del valor numérico
			if inodo.I_type == 0 { // Directorio
				result.WriteString(fmt.Sprintf("  inode_%d:block_%d -> block_dir_%d;\n", inodeIdx, i, inodo.I_block[i]))
				contenido := procesarBloqueCarpeta(file, sb, inodo.I_block[i], inodeIdx, i, procesados, bloquesProcesados)
				result.WriteString(contenido)
			} else { // Archivo
				result.WriteString(fmt.Sprintf("  inode_%d:block_%d -> block_file_%d;\n", inodeIdx, i, inodo.I_block[i]))
				contenido := procesarBloqueArchivo(file, sb, inodo.I_block[i], inodeIdx, i, bloquesProcesados)
				result.WriteString(contenido)
			}
		}
	}

	// Procesar apuntadores indirectos con puertos correctos
	if inodo.I_block[12] != -1 {
		result.WriteString(fmt.Sprintf("  inode_%d:block_12 -> block_ptr_%d;\n", inodeIdx, inodo.I_block[12]))
		contenido := procesarBloqueApuntadorTree(file, sb, inodo.I_block[12], inodeIdx, 1, inodo.I_type, procesados, bloquesProcesados)
		result.WriteString(contenido)
	}

	if inodo.I_block[13] != -1 {
		result.WriteString(fmt.Sprintf("  inode_%d:block_13 -> block_ptr_%d;\n", inodeIdx, inodo.I_block[13]))
		contenido := procesarBloqueApuntadorTree(file, sb, inodo.I_block[13], inodeIdx, 2, inodo.I_type, procesados, bloquesProcesados)
		result.WriteString(contenido)
	}

	if inodo.I_block[14] != -1 {
		result.WriteString(fmt.Sprintf("  inode_%d:block_14 -> block_ptr_%d;\n", inodeIdx, inodo.I_block[14]))
		contenido := procesarBloqueApuntadorTree(file, sb, inodo.I_block[14], inodeIdx, 3, inodo.I_type, procesados, bloquesProcesados)
		result.WriteString(contenido)
	}

	return result.String()
}

// procesarBloqueCarpeta con conexiones precisas desde cada entrada
func procesarBloqueCarpeta(file *os.File, sb Structs.SuperBloque, bloqueIdx int64, inodeIdx int64, apuntadorIdx int, procesados *map[int64]bool, bloquesProcesados *map[int64]bool) string {
	// Evitar procesar bloques ya procesados
	if (*bloquesProcesados)[bloqueIdx] {
		return ""
	}

	// Marcar bloque como procesado
	(*bloquesProcesados)[bloqueIdx] = true

	// Leer bloque de carpeta
	var bloqueCarpetas Structs.BloquesCarpetas
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueCarpetas); err != nil {
		fmt.Println("Error leyendo bloque carpeta:", err)
		return ""
	}

	var result strings.Builder

	// Crear tabla HTML con puertos específicos SOLO en los valores numéricos
	result.WriteString(fmt.Sprintf("  block_dir_%d [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n", bloqueIdx))
	result.WriteString(fmt.Sprintf("    <TR><TD BGCOLOR=\"#1565C0\" COLSPAN=\"2\"><FONT COLOR=\"white\">BLOQUE CARPETA %d</FONT></TD></TR>\n", bloqueIdx))

	// Procesamiento: Puerto solo en la celda del valor del inodo
	for i, entrada := range bloqueCarpetas.B_content {
		if entrada.B_inodo != -1 {
			nombre := strings.Trim(string(entrada.B_name[:]), "\x00")
			// El puerto va SOLO en la celda del valor numérico
			result.WriteString(fmt.Sprintf("    <TR><TD>%s</TD><TD PORT=\"entry_%d\">%d</TD></TR>\n",
				nombre, i, entrada.B_inodo))
		} else {
			result.WriteString(fmt.Sprintf("    <TR><TD>-</TD><TD>-1</TD></TR>\n"))
		}
	}
	result.WriteString("  </TABLE>>];\n\n")

	// Procesamiento: Conexiones precisas desde el puerto del valor
	for i, entrada := range bloqueCarpetas.B_content {
		if entrada.B_inodo != -1 {
			nombre := strings.Trim(string(entrada.B_name[:]), "\x00")
			if nombre != "." && nombre != ".." {
				// La flecha sale exactamente del puerto del valor numérico
				result.WriteString(fmt.Sprintf("  block_dir_%d:entry_%d -> inode_%d;\n",
					bloqueIdx, i, entrada.B_inodo))

				// Procesar inodo recursivamente
				if !(*procesados)[entrada.B_inodo] {
					result.WriteString(procesarInodo(file, sb, entrada.B_inodo, procesados, bloquesProcesados))
				}
			}
		}
	}

	return result.String()
}

// procesarBloqueArchivo mejorado
func procesarBloqueArchivo(file *os.File, sb Structs.SuperBloque, bloqueIdx int64, inodeIdx int64, apuntadorIdx int, bloquesProcesados *map[int64]bool) string {
	// Evitar procesar bloques ya procesados
	if (*bloquesProcesados)[bloqueIdx] {
		return ""
	}

	// Marcar bloque como procesado
	(*bloquesProcesados)[bloqueIdx] = true

	// Leer bloque de archivo
	var bloquesArchivos Structs.BloquesArchivos
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloquesArchivos); err != nil {
		fmt.Println("Error leyendo bloque archivo:", err)
		return ""
	}

	var result strings.Builder

	// Obtener vista previa del contenido
	contenido := string(bytes.Trim(bloquesArchivos.B_content[:], "\x00"))
	if len(contenido) > 15 {
		contenido = contenido[:12] + "..."
	}

	// Escapar caracteres especiales para DOT
	contenido = strings.ReplaceAll(contenido, "\"", "\\\"")
	contenido = strings.ReplaceAll(contenido, "\n", "\\n")

	// Crear nodo para bloque de archivo
	result.WriteString(fmt.Sprintf("  block_file_%d [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n", bloqueIdx))
	result.WriteString(fmt.Sprintf("    <TR><TD BGCOLOR=\"#2E7D32\" COLSPAN=\"2\"><FONT COLOR=\"white\">BLOQUE ARCHIVO %d</FONT></TD></TR>\n", bloqueIdx))
	result.WriteString(fmt.Sprintf("    <TR><TD>%s</TD></TR>\n", contenido))
	result.WriteString("  </TABLE>>];\n\n")

	return result.String()
}

// procesarBloqueApuntadorTree con conexiones precisas desde cada puntero
func procesarBloqueApuntadorTree(file *os.File, sb Structs.SuperBloque, bloqueIdx int64, inodeIdx int64, nivel int, tipoInodo int64, procesados *map[int64]bool, bloquesProcesados *map[int64]bool) string {
	// Evitar procesar bloques ya procesados
	if (*bloquesProcesados)[bloqueIdx] {
		return ""
	}

	// Marcar bloque como procesado
	(*bloquesProcesados)[bloqueIdx] = true

	// Leer bloque de apuntadores
	var bloqueApuntadores Structs.BloquesApuntadores
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueApuntadores); err != nil {
		fmt.Println("Error leyendo bloque apuntador:", err)
		return ""
	}

	var result strings.Builder

	// Determinar tipo de apuntador
	var tipoStr string
	var color string
	switch nivel {
	case 1:
		tipoStr = "SIMPLE"
		color = "#FF9800"
	case 2:
		tipoStr = "DOBLE"
		color = "#F57C00"
	case 3:
		tipoStr = "TRIPLE"
		color = "#E65100"
	}

	// Procesamiento: Puerto específico SOLO en los valores numéricos
	result.WriteString(fmt.Sprintf("  block_ptr_%d [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n", bloqueIdx))
	result.WriteString(fmt.Sprintf("    <TR><TD BGCOLOR=\"%s\" COLSPAN=\"2\"><FONT COLOR=\"white\">BLOQUE APUNTADOR %s %d</FONT></TD></TR>\n", color, tipoStr, bloqueIdx))

	// Mostrar punteros con puerto SOLO en el valor numérico
	for i, ptr := range bloqueApuntadores.B_pointers {
		result.WriteString(fmt.Sprintf("    <TR><TD>PTR %d</TD><TD PORT=\"ptr_%d\">%d</TD></TR>\n",
			i, i, ptr))
	}
	result.WriteString("  </TABLE>>];\n\n")

	// Procesamiento: Conexiones precisas desde el puerto del valor
	for i, ptr := range bloqueApuntadores.B_pointers {
		if ptr != -1 {
			if nivel == 1 {
				// Es un apuntador simple, apunta a bloques de datos
				if tipoInodo == 0 { // Directorio
					result.WriteString(fmt.Sprintf("  block_ptr_%d:ptr_%d -> block_dir_%d;\n", bloqueIdx, i, ptr))
					contenido := procesarBloqueCarpeta(file, sb, ptr, inodeIdx, i, procesados, bloquesProcesados)
					result.WriteString(contenido)
				} else { // Archivo
					result.WriteString(fmt.Sprintf("  block_ptr_%d:ptr_%d -> block_file_%d;\n", bloqueIdx, i, ptr))
					contenido := procesarBloqueArchivo(file, sb, ptr, inodeIdx, i, bloquesProcesados)
					result.WriteString(contenido)
				}
			} else {
				// Es un apuntador de nivel superior (doble o triple)
				result.WriteString(fmt.Sprintf("  block_ptr_%d:ptr_%d -> block_ptr_%d;\n", bloqueIdx, i, ptr))
				contenido := procesarBloqueApuntadorTree(file, sb, ptr, inodeIdx, nivel-1, tipoInodo, procesados, bloquesProcesados)
				result.WriteString(contenido)
			}
		}
	}

	return result.String()
}

func BitMap_inodo(path string, id string) string {
	// Obtener la ruta del disco
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		return Utils.Error("REP", "No se encontró el disco para la ID: "+id)
	}

	// Abrir el archivo del disco
	file, err := os.Open(diskPath)
	if err != nil {
		return Utils.Error("REP", "Error al abrir el disco: "+err.Error())
	}
	defer file.Close()

	// Obtener la partición montada
	particion := GetMount("REP", id, &diskPath)
	if particion == nil {
		return Utils.Error("REP", "No se encontró la partición montada")
	}

	// Leer el superbloque
	var sb Structs.SuperBloque
	file.Seek(particion.Part_start, 0)
	if err := binary.Read(file, binary.LittleEndian, &sb); err != nil {
		return Utils.Error("REP", "Error al leer el superbloque: "+err.Error())
	}

	// Leer el bitmap de inodos
	count := int(sb.S_inodes_count)
	bitmap := make([]byte, count)
	file.Seek(sb.S_bm_inode_start, 0)
	if _, err := file.Read(bitmap); err != nil {
		return Utils.Error("REP", "Error al leer el bitmap de inodos: "+err.Error())
	}

	// Tamaño de la visualización
	const filas = 21
	const cols = 21

	var b bytes.Buffer
	b.WriteString("REPORTE BITMAP INODOS\n")
	b.WriteString(fmt.Sprintf("Partición: %s\n", diskPath))
	b.WriteString(fmt.Sprintf("Total inodos: %d\n\n", sb.S_inodes_count))

	// Encabezado
	b.WriteString("     ")
	for c := 0; c < cols; c++ {
		b.WriteString(fmt.Sprintf(" %2d", c))
	}
	b.WriteString("\n")

	// Filas
	for r := 0; r < filas; r++ {
		start := r * cols
		if start >= count {
			break
		}
		b.WriteString(fmt.Sprintf("%3d:", r+1))
		for c := 0; c < cols && (start+c) < count; c++ {
			bit := '0'
			if bitmap[start+c] == '1' {
				bit = '1'
			}
			b.WriteString(fmt.Sprintf("  %c", bit))
		}
		b.WriteString("\n")
	}

	if count > filas*cols {
		b.WriteString(fmt.Sprintf("\n... %d inodos más ...\n", count-(filas*cols)))
	}

	if err := os.WriteFile(path, b.Bytes(), 0644); err != nil {
		return Utils.Error("REP", "Error escribiendo archivo TXT: "+err.Error())
	}

	return Utils.Mensaje("REP", "Reporte de bitmap de inodos generado exitosamente: "+path)
}

func BitMap_block(path string, id string) string {
	// Obtener la ruta del disco
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		return Utils.Error("REP", "No se encontró el disco para la ID: "+id)
	}

	// Abrir el archivo del disco
	file, err := os.Open(diskPath)
	if err != nil {
		return Utils.Error("REP", "Error al abrir el disco: "+err.Error())
	}
	defer file.Close()

	// Obtener la partición montada
	particion := GetMount("REP", id, &diskPath)
	if particion == nil {
		return Utils.Error("REP", "No se encontró la partición montada")
	}

	// Leer el superbloque
	var sb Structs.SuperBloque
	file.Seek(particion.Part_start, 0)
	if err := binary.Read(file, binary.LittleEndian, &sb); err != nil {
		return Utils.Error("REP", "Error al leer el superbloque: "+err.Error())
	}

	// Leer el bitmap de bloques
	count := int(sb.S_blocks_count)
	bitmap := make([]byte, count)
	file.Seek(sb.S_bm_block_start, 0)
	if _, err := file.Read(bitmap); err != nil {
		return Utils.Error("REP", "Error al leer el bitmap de bloques: "+err.Error())
	}

	// Tamaño de la visualización
	const filas = 21
	const cols = 21

	var b bytes.Buffer
	b.WriteString("REPORTE BITMAP BLOQUES\n")
	b.WriteString(fmt.Sprintf("Partición: %s\n", diskPath))
	b.WriteString(fmt.Sprintf("Total bloques: %d\n\n", sb.S_blocks_count))

	// Encabezado
	b.WriteString("     ")
	for c := 0; c < cols; c++ {
		b.WriteString(fmt.Sprintf(" %2d", c))
	}
	b.WriteString("\n")

	// Filas
	for r := 0; r < filas; r++ {
		start := r * cols
		if start >= count {
			break
		}
		b.WriteString(fmt.Sprintf("%3d:", r+1))
		for c := 0; c < cols && (start+c) < count; c++ {
			bit := '0'
			if bitmap[start+c] == '1' {
				bit = '1'
			}
			b.WriteString(fmt.Sprintf("  %c", bit))
		}
		b.WriteString("\n")
	}

	if count > filas*cols {
		b.WriteString(fmt.Sprintf("\n... %d bloques más ...\n", count-(filas*cols)))
	}

	if err := os.WriteFile(path, b.Bytes(), 0644); err != nil {
		return Utils.Error("REP", "Error escribiendo archivo TXT: "+err.Error())
	}

	return Utils.Mensaje("REP", "Reporte de bitmap de bloques generado exitosamente: "+path)
}

func Report_Block(path string, id string) string {
	// Obtener la ruta del disco
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		return Utils.Error("REP", "No se encontró el disco para la ID: "+id)
	}

	// Abrir el archivo del disco
	file, err := os.OpenFile(diskPath, os.O_RDONLY, 0644)
	if err != nil {
		return Utils.Error("REP", "Error abriendo disco: "+err.Error())
	}
	defer file.Close()

	// Obtener la partición montada
	particion := GetMount("REP", id, &diskPath)
	if particion == nil {
		return Utils.Error("REP", "No se encontró la partición montada")
	}

	// Leer el superbloque
	var sb Structs.SuperBloque
	file.Seek(particion.Part_start, 0)
	if err := binary.Read(file, binary.LittleEndian, &sb); err != nil {
		return Utils.Error("REP", "Error leyendo superbloque: "+err.Error())
	}

	fmt.Printf("🔧 DEBUG: Generando reporte de bloques para la partición en: %s\n", diskPath)

	// Crear contenido DOT
	var dot strings.Builder
	dot.WriteString("digraph G {\n")
	dot.WriteString("  graph [rankdir=LR, splines=ortho, nodesep=0.8];\n") // Disposición horizontal (left-right)
	dot.WriteString("  node [shape=record];\n")
	dot.WriteString("  bgcolor=\"#FFFFFF\";\n\n")

	// Mapas para rastrear bloques
	bloquesCarpeta := make(map[int64]Structs.BloquesCarpetas)
	bloquesArchivo := make(map[int64][]byte)
	bloquesApuntadores := make(map[int64]Structs.BloquesApuntadores)

	// Mapa para rastrear conexiones
	conexiones := make(map[string]bool) // Usar mapa para evitar duplicados

	// Procesar inodos para encontrar bloques
	fmt.Printf("🔍 Procesando inodos para identificar bloques...\n")
	for i := int64(0); i < sb.S_inodes_count; i++ {
		var inodo Structs.Inodos
		file.Seek(sb.S_inode_start+i*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
		if err := binary.Read(file, binary.LittleEndian, &inodo); err != nil {
			continue
		}

		// Verificar si el inodo está en uso
		if inodo.I_size == -1 {
			continue
		}

		fmt.Printf("  📁 Inodo %d encontrado (tipo: %d)\n", i, inodo.I_type)

		// Determinar si es directorio o archivo
		esCarpeta := inodo.I_type == 0

		// Leer bloques directos (0-11)
		for j := 0; j < 12; j++ {
			if inodo.I_block[j] == -1 {
				continue
			}

			fmt.Printf("    💾 Bloque directo[%d] = %d\n", j, inodo.I_block[j])

			// Leer el bloque según su tipo
			if esCarpeta {
				var bloqueCarpeta Structs.BloquesCarpetas
				file.Seek(sb.S_block_start+inodo.I_block[j]*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
				if err := binary.Read(file, binary.LittleEndian, &bloqueCarpeta); err != nil {
					continue
				}
				bloquesCarpeta[inodo.I_block[j]] = bloqueCarpeta
			} else {
				// Leer bloque de archivo
				var bloqueArchivo Structs.BloquesArchivos
				file.Seek(sb.S_block_start+inodo.I_block[j]*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
				if err := binary.Read(file, binary.LittleEndian, &bloqueArchivo); err != nil {
					continue
				}
				// Extraer solo los bytes de contenido del bloque
				bloquesArchivo[inodo.I_block[j]] = bloqueArchivo.B_content[:]
			}
		}

		// Leer bloque de apuntadores simple (índice 12)
		if inodo.I_block[12] != -1 {
			fmt.Printf("    📌 Bloque apuntador simple: %d\n", inodo.I_block[12])
			leerBloqueApuntador(file, sb, inodo.I_block[12], 1, esCarpeta,
				bloquesCarpeta, bloquesArchivo, bloquesApuntadores, conexiones)
		}

		// Leer bloque de apuntadores doble (índice 13)
		if inodo.I_block[13] != -1 {
			fmt.Printf("    📌 Bloque apuntador doble: %d\n", inodo.I_block[13])
			leerBloqueApuntador(file, sb, inodo.I_block[13], 2, esCarpeta,
				bloquesCarpeta, bloquesArchivo, bloquesApuntadores, conexiones)
		}

		// Leer bloque de apuntadores triple (índice 14)
		if inodo.I_block[14] != -1 {
			fmt.Printf("    📌 Bloque apuntador triple: %d\n", inodo.I_block[14])
			leerBloqueApuntador(file, sb, inodo.I_block[14], 3, esCarpeta,
				bloquesCarpeta, bloquesArchivo, bloquesApuntadores, conexiones)
		}
	}

	// Agregar nodos de bloques de carpetas
	if len(bloquesCarpeta) > 0 {
		dot.WriteString("  subgraph cluster_carpetas {\n")
		dot.WriteString("    label=\"Bloques de Carpetas\";\n")
		dot.WriteString("    bgcolor=\"#E3F2FD\";\n")

		for idx, bloque := range bloquesCarpeta {
			dot.WriteString(fmt.Sprintf("    bc_%d [shape=record, style=filled, fillcolor=\"#90CAF9\", label=\"{Bloque %d|{", idx, idx))

			// Mostrar contenido del bloque de carpeta
			entradas := []string{}
			for _, entrada := range bloque.B_content {
				if entrada.B_inodo != -1 {
					nombre := strings.Trim(string(entrada.B_name[:]), "\x00")
					entradas = append(entradas, fmt.Sprintf("%s:%d", nombre, entrada.B_inodo))
				}
			}

			if len(entradas) > 0 {
				dot.WriteString(strings.Join(entradas, "|"))
			} else {
				dot.WriteString("vacío")
			}

			dot.WriteString("}}\"]; // bloque carpeta\n")
		}
		dot.WriteString("  }\n\n")
	}

	// Agregar nodos de bloques de archivos
	if len(bloquesArchivo) > 0 {
		dot.WriteString("  subgraph cluster_archivos {\n")
		dot.WriteString("    label=\"Bloques de Archivos\";\n")
		dot.WriteString("    bgcolor=\"#E8F5E9\";\n")

		for idx, contenido := range bloquesArchivo {
			// Preparar vista previa del contenido limpiando bytes nulos y caracteres no imprimibles
			cleanContent := []byte{}
			for _, b := range contenido {
				// Solo incluir caracteres imprimibles y espacios
				if (b >= 32 && b <= 126) || b == 10 || b == 13 || b == 9 { // ASCII imprimibles + nueva línea + retorno + tab
					cleanContent = append(cleanContent, b)
				}
			}

			preview := string(cleanContent)
			// Recortar espacios en blanco iniciales y finales
			preview = strings.TrimSpace(preview)

			// Limitar longitud de la vista previa
			if len(preview) > 15 {
				preview = preview[:12] + "..."
			} else if preview == "" {
				preview = "(vacío)"
			}

			// Escapar caracteres especiales para DOT
			preview = strings.ReplaceAll(preview, "\"", "\\\"")
			preview = strings.ReplaceAll(preview, "\n", "\\n")

			dot.WriteString(fmt.Sprintf("    ba_%d [shape=record, style=filled, fillcolor=\"#A5D6A7\", label=\"{Bloque %d|%s}\"];\n",
				idx, idx, preview))
		}
		dot.WriteString("  }\n\n")
	}

	// Agregar nodos de bloques de apuntadores
	if len(bloquesApuntadores) > 0 {
		dot.WriteString("  subgraph cluster_apuntadores {\n")
		dot.WriteString("    label=\"Bloques de Apuntadores\";\n")
		dot.WriteString("    bgcolor=\"#FFF8E1\";\n")

		for idx, bloque := range bloquesApuntadores {
			// Determinar tipo de apuntador
			tipoApuntador := ""
			if _, ok := conexiones[fmt.Sprintf("ap_%d_nivel_1", idx)]; ok {
				tipoApuntador = "Simple"
			} else if _, ok := conexiones[fmt.Sprintf("ap_%d_nivel_2", idx)]; ok {
				tipoApuntador = "Doble"
			} else if _, ok := conexiones[fmt.Sprintf("ap_%d_nivel_3", idx)]; ok {
				tipoApuntador = "Triple"
			} else {
				tipoApuntador = "???"
			}

			dot.WriteString(fmt.Sprintf("    ap_%d [shape=record, style=filled, fillcolor=\"#FFECB3\", label=\"{Bloque Ap. %s %d|{",
				idx, tipoApuntador, idx))

			// Mostrar punteros válidos
			punteros := []string{}
			for _, ptr := range bloque.B_pointers {
				if ptr != -1 {
					punteros = append(punteros, fmt.Sprintf("%d", ptr))
				}
			}

			if len(punteros) > 0 {
				dot.WriteString(strings.Join(punteros, "|"))
			} else {
				dot.WriteString("vacío")
			}

			dot.WriteString("}}\"]; // bloque apuntador\n")
		}
		dot.WriteString("  }\n\n")
	}

	// Agregar todas las conexiones
	for conexion := range conexiones {
		dot.WriteString("  " + conexion + " [color=\"#1976D2\"];\n")
	}

	// Cerrar el gráfico
	dot.WriteString("}\n")

	// Escribir archivo DOT
	dotPath := path + ".dot"
	if err := os.WriteFile(dotPath, []byte(dot.String()), 0644); err != nil {
		return Utils.Error("REP", "Error escribiendo archivo DOT: "+err.Error())
	}

	// Generar PNG usando Graphviz
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("Error generando imagen:", err)
		fmt.Println("Output:", string(output))
		return Utils.Error("REP", "Error generando imagen PNG: "+err.Error())
	}

	// Eliminar archivo DOT temporal si no hay errores
	os.Remove(dotPath)

	return Utils.Mensaje("REP", "Reporte de bloques generado exitosamente en: "+path)
}

// leerBloqueApuntador lee recursivamente un bloque de apuntadores y registra sus conexiones
func leerBloqueApuntador(
	file *os.File,
	sb Structs.SuperBloque,
	bloqueIdx int64,
	nivel int,
	esCarpeta bool,
	bloquesCarpeta map[int64]Structs.BloquesCarpetas,
	bloquesArchivo map[int64][]byte,
	bloquesApuntadores map[int64]Structs.BloquesApuntadores,
	conexiones map[string]bool) {

	// Evitar recursión infinita
	if nivel <= 0 || bloqueIdx < 0 {
		return
	}

	// Guardar nivel de este apuntador para identificación posterior
	conexiones[fmt.Sprintf("ap_%d_nivel_%d", bloqueIdx, nivel)] = true

	// Leer el bloque de apuntadores
	var bloqueAp Structs.BloquesApuntadores
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueAp); err != nil {
		fmt.Printf("⚠️ Error leyendo bloque apuntador %d: %v\n", bloqueIdx, err)
		return
	}

	// Registrar el bloque de apuntadores
	bloquesApuntadores[bloqueIdx] = bloqueAp

	// Procesar cada puntero del bloque
	for _, ptr := range bloqueAp.B_pointers {
		if ptr == -1 {
			continue
		}

		// Agregar conexión entre este bloque de apuntador y el bloque apuntado
		if nivel == 1 {
			// Bloque de apuntador simple apunta a bloques de datos
			if esCarpeta {
				// Leer el bloque de carpeta apuntado
				var bloqueCarpeta Structs.BloquesCarpetas
				file.Seek(sb.S_block_start+ptr*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
				if err := binary.Read(file, binary.LittleEndian, &bloqueCarpeta); err == nil {
					bloquesCarpeta[ptr] = bloqueCarpeta
					// Crear conexión
					conexiones[fmt.Sprintf("ap_%d -> bc_%d", bloqueIdx, ptr)] = true
				}
			} else {
				// Leer bloque de archivo
				data := make([]byte, unsafe.Sizeof(Structs.BloquesArchivos{}))
				file.Seek(sb.S_block_start+ptr*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
				if _, err := file.Read(data); err == nil {
					bloquesArchivo[ptr] = data
					// Crear conexión
					conexiones[fmt.Sprintf("ap_%d -> ba_%d", bloqueIdx, ptr)] = true
				}
			}
		} else {
			// Bloque apuntador de nivel superior apunta a otro bloque de apuntadores
			leerBloqueApuntador(file, sb, ptr, nivel-1, esCarpeta,
				bloquesCarpeta, bloquesArchivo, bloquesApuntadores, conexiones)

			// Crear conexión entre bloques de apuntadores
			conexiones[fmt.Sprintf("ap_%d -> ap_%d", bloqueIdx, ptr)] = true
		}
	}
}

// procesarBloqueApuntador lee recursivamente un bloque de apuntadores y registra sus conexiones
func procesarBloqueApuntador(
	file *os.File,
	sb Structs.SuperBloque,
	bloqueIdx int64,
	nivel int,
	bloquesCarpeta map[int64]bool,
	bloquesArchivo map[int64]bool,
	bloquesApuntadores map[int64]int,
	conexiones *[]string,
	esCarpeta bool) {

	// Evitar recursión infinita
	if nivel <= 0 {
		return
	}

	// Leer el bloque de apuntadores
	var bloque Structs.BloquesApuntadores
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloque); err != nil {
		fmt.Printf("⚠️ Error leyendo bloque apuntador %d: %v\n", bloqueIdx, err)
		return
	}

	// Procesar cada puntero del bloque
	for _, ptr := range bloque.B_pointers {
		if ptr == -1 {
			continue
		}

		// Agregar conexión
		*conexiones = append(*conexiones, fmt.Sprintf("bloque_%d -> bloque_%d", bloqueIdx, ptr))

		if nivel == 1 {
			// Bloque apuntador simple apunta a bloques de datos
			if esCarpeta {
				bloquesCarpeta[ptr] = true
			} else {
				bloquesArchivo[ptr] = true
			}
		} else {
			// Bloque apuntador múltiple apunta a otros bloques de apuntadores
			bloquesApuntadores[ptr] = nivel - 1
			procesarBloqueApuntador(file, sb, ptr, nivel-1, bloquesCarpeta, bloquesArchivo, bloquesApuntadores, conexiones, esCarpeta)
		}
	}
}

// contarBloquesDirectosUsados cuenta cuántos bloques directos están utilizados en un inodo
func contarBloquesDirectosUsados(inodo Structs.Inodos) int {
	count := 0
	for i := 0; i < 12; i++ { // Solo bloques directos
		if inodo.I_block[i] != -1 {
			count++
		}
	}
	return count
}

// procesarBloqueContenido procesa un bloque y determina su tipo (carpeta o archivo)
func procesarBloqueContenido(file *os.File, sb Structs.SuperBloque, bloqueIdx int64, dotContent *string,
	bloquesVistos map[int64]bool, conexiones map[int64][]int64) {

	// Intentar leer primero como bloque carpeta
	var bloqueCarpeta Structs.BloquesCarpetas
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueCarpeta); err != nil {
		fmt.Printf("⚠️ Warning: Error leyendo bloque %d: %v\n", bloqueIdx, err)
		return
	}

	// Verificar si parece un bloque de carpeta válido
	esDirectorio := false
	for _, entry := range bloqueCarpeta.B_content {
		if entry.B_inodo != -1 && strings.Trim(string(entry.B_name[:]), "\x00") != "" {
			esDirectorio = true
			break
		}
	}

	if esDirectorio {
		// Es un bloque de directorio
		*dotContent += fmt.Sprintf("  bloque_%d [label=\"{Bloque Carpeta %d|{", bloqueIdx, bloqueIdx)

		first := true
		for _, entry := range bloqueCarpeta.B_content {
			if entry.B_inodo != -1 {
				name := strings.Trim(string(entry.B_name[:]), "\x00")
				if name != "" {
					if !first {
						*dotContent += "|"
					}
					*dotContent += fmt.Sprintf("%s → i:%d", name, entry.B_inodo)
					first = false

					// Registrar conexión con el inodo
					conexiones[bloqueIdx] = append(conexiones[bloqueIdx], entry.B_inodo)
				}
			}
		}

		if first { // No se encontraron entradas
			*dotContent += "(vacío)"
		}

		*dotContent += "}}\", fillcolor=\"#e1f5fe\"];\n"
	} else {
		// Leer como bloque de archivo
		data, err := readFileBlock(file, sb, bloqueIdx)
		if err != nil {
			fmt.Printf("⚠️ Warning: Error leyendo bloque archivo %d: %v\n", bloqueIdx, err)
			return
		}

		contenido := string(data)
		if strings.TrimSpace(contenido) == "" {
			contenido = "(vacío)"
		} else if len(contenido) > 20 {
			contenido = strings.ReplaceAll(contenido[:17], "\n", "\\n") + "..."
		} else {
			contenido = strings.ReplaceAll(contenido, "\n", "\\n")
		}

		contenido = strings.ReplaceAll(contenido, "\"", "\\\"")

		*dotContent += fmt.Sprintf("  bloque_%d [label=\"{Bloque Archivo %d|%s}\", fillcolor=\"#e8f5e9\"];\n",
			bloqueIdx, bloqueIdx, contenido)
	}

	bloquesVistos[bloqueIdx] = true
}

// procesarBloqueApuntadorSimple procesa un bloque de apuntadores simple
func procesarBloqueApuntadorSimple(file *os.File, sb Structs.SuperBloque, bloqueIdx int64, dotContent *string,
	bloquesVistos map[int64]bool, conexiones map[int64][]int64) {

	if bloquesVistos[bloqueIdx] {
		return
	}

	var bloqueAp Structs.BloquesApuntadores
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueAp); err != nil {
		fmt.Printf("⚠️ Warning: Error leyendo bloque apuntadores %d: %v\n", bloqueIdx, err)
		return
	}

	*dotContent += fmt.Sprintf("  bloque_%d [label=\"{Bloque Ap. Simple %d|{", bloqueIdx, bloqueIdx)

	punterosValidos := []int64{}
	for _, ptr := range bloqueAp.B_pointers {
		if ptr != -1 {
			punterosValidos = append(punterosValidos, ptr)
			conexiones[bloqueIdx] = append(conexiones[bloqueIdx], ptr)
		}
	}

	for i, ptr := range punterosValidos {
		if i > 0 {
			*dotContent += "|"
		}
		*dotContent += fmt.Sprintf("%d", ptr)
	}

	if len(punterosValidos) == 0 {
		*dotContent += "(vacío)"
	}

	*dotContent += "}}\", fillcolor=\"#fff8e1\"];\n"

	// Procesar cada bloque apuntado
	for _, ptr := range punterosValidos {
		if !bloquesVistos[ptr] {
			procesarBloqueContenido(file, sb, ptr, dotContent, bloquesVistos, conexiones)
		}
	}

	bloquesVistos[bloqueIdx] = true
}

// procesarBloqueApuntadorDoble procesa un bloque de apuntadores doble
func procesarBloqueApuntadorDoble(file *os.File, sb Structs.SuperBloque, bloqueIdx int64, dotContent *string,
	bloquesVistos map[int64]bool, conexiones map[int64][]int64) {

	if bloquesVistos[bloqueIdx] {
		return
	}

	var bloqueAp Structs.BloquesApuntadores
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueAp); err != nil {
		fmt.Printf("⚠️ Warning: Error leyendo bloque apuntadores doble %d: %v\n", bloqueIdx, err)
		return
	}

	*dotContent += fmt.Sprintf("  bloque_%d [label=\"{Bloque Ap. Doble %d|{", bloqueIdx, bloqueIdx)

	punterosValidos := []int64{}
	for _, ptr := range bloqueAp.B_pointers {
		if ptr != -1 {
			punterosValidos = append(punterosValidos, ptr)
			conexiones[bloqueIdx] = append(conexiones[bloqueIdx], ptr)
		}
	}

	for i, ptr := range punterosValidos {
		if i > 0 {
			*dotContent += "|"
		}
		*dotContent += fmt.Sprintf("%d", ptr)
	}

	if len(punterosValidos) == 0 {
		*dotContent += "(vacío)"
	}

	*dotContent += "}}\", fillcolor=\"#fce4ec\"];\n"

	// Procesar cada bloque de apuntadores simple referenciado
	for _, ptr := range punterosValidos {
		if !bloquesVistos[ptr] {
			procesarBloqueApuntadorSimple(file, sb, ptr, dotContent, bloquesVistos, conexiones)
		}
	}

	bloquesVistos[bloqueIdx] = true
}

// Report_Inode generates a report of inodes and writes a PNG via dot
func Report_Inode(path string, id string) string {
	// Find disk path
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		return Utils.Error("REP", "No se encontró el disco para la ID: "+id)
	}

	f, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
	if err != nil {
		return Utils.Error("REP", "Error abriendo disco: "+err.Error())
	}
	defer f.Close()

	part := GetMount("REP", id, &diskPath)
	if part == nil {
		return Utils.Error("REP", "No se encontró la partición montada")
	}

	var sb Structs.SuperBloque
	f.Seek(part.Part_start, 0)
	if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
		return Utils.Error("REP", "Error leyendo superbloque: "+err.Error())
	}

	// Read inode bitmap
	bitmap := make([]byte, sb.S_inodes_count)
	f.Seek(sb.S_bm_inode_start, 0)
	if _, err := f.Read(bitmap); err != nil {
		return Utils.Error("REP", "Error leyendo bitmap de inodos: "+err.Error())
	}

	// Build DOT table
	var b bytes.Buffer
	b.WriteString("digraph G {\n")
	b.WriteString("  node [shape=plaintext];\n")
	b.WriteString("  tbl [label=<\n")
	b.WriteString("  <table border=1 cellborder=1 cellspacing=0 cellpadding=4>\n")
	b.WriteString("    <tr><td><b>Index</b></td><td><b>Type</b></td><td><b>Size</b></td><td><b>Blocks</b></td><td><b>ATime</b></td><td><b>CTime</b></td><td><b>MTime</b></td></tr>\n")

	for i := int64(0); i < sb.S_inodes_count; i++ {
		var inode Structs.Inodos
		f.Seek(sb.S_inode_start+(i*int64(unsafe.Sizeof(Structs.Inodos{}))), 0)
		if err := binary.Read(f, binary.LittleEndian, &inode); err != nil {
			// skip on read error
			continue
		}

		// Only show allocated inodes
		if i < int64(len(bitmap)) && bitmap[i] != '1' {
			continue
		}

		t := "DIR"
		if inode.I_type == 1 {
			t = "FILE"
		}

		// Collect block list
		var blocks []string
		for _, bidx := range inode.I_block {
			if bidx != -1 {
				blocks = append(blocks, fmt.Sprintf("%d", bidx))
			}
		}

		aTime := strings.TrimRight(string(inode.I_atime[:]), "\x00")
		cTime := strings.TrimRight(string(inode.I_ctime[:]), "\x00")
		mTime := strings.TrimRight(string(inode.I_mtime[:]), "\x00")

		b.WriteString(fmt.Sprintf("    <tr><td>%d</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			i, t, inode.I_size, strings.Join(blocks, ","), aTime, cTime, mTime))
	}

	b.WriteString("  </table>\n")
	b.WriteString(">];\n")
	b.WriteString("}\n")

	// Write temporary dot file
	tmpDot := path + ".dot"
	if err := os.WriteFile(tmpDot, b.Bytes(), 0644); err != nil {
		return Utils.Error("REP", "Error escribiendo DOT temporal: "+err.Error())
	}

	// Run dot to generate PNG
	cmd := exec.Command("dot", "-Tpng", tmpDot, "-o", path)
	if err := cmd.Run(); err != nil {
		// If dot fails, still write the DOT as fallback
		_ = os.WriteFile(path+".txt", b.Bytes(), 0644)
		return Utils.Error("REP", "Error ejecutando dot para generar PNG: "+err.Error())
	}

	// Clean up dot file
	_ = os.Remove(tmpDot)

	return Utils.Mensaje("REP", "Reporte de inodos generado: "+path)
}

// procesarBloqueApuntadorDetallado lee recursivamente un bloque de apuntadores y registra sus conexiones
func procesarBloqueApuntadorDetallado(
	file *os.File,
	sb Structs.SuperBloque,
	bloqueIdx int64,
	nivel int,
	bloquesCarpeta map[int64]bool,
	bloquesArchivo map[int64]bool,
	bloquesApuntadores map[int64]int,
	conexiones *[]string,
	esCarpeta bool) {

	// Evitar recursión infinita
	if nivel <= 0 {
		return
	}

	// Leer el bloque de apuntadores
	var bloque Structs.BloquesApuntadores
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloque); err != nil {
		fmt.Printf("⚠️ Error leyendo bloque apuntador %d: %v\n", bloqueIdx, err)
		return
	}

	// Procesar cada puntero del bloque
	for _, ptr := range bloque.B_pointers {
		if ptr == -1 {
			continue
		}

		// Agregar conexión
		*conexiones = append(*conexiones, fmt.Sprintf("bloque_%d -> bloque_%d", bloqueIdx, ptr))

		if nivel == 1 {
			// Bloque apuntador simple apunta a bloques de datos
			if esCarpeta {
				bloquesCarpeta[ptr] = true
			} else {
				bloquesArchivo[ptr] = true
			}
		} else {
			// Bloque apuntador múltiple apunta a otros bloques de apuntadores
			bloquesApuntadores[ptr] = nivel - 1
			procesarBloqueApuntadorDetallado(file, sb, ptr, nivel-1, bloquesCarpeta, bloquesArchivo, bloquesApuntadores, conexiones, esCarpeta)
		}
	}
}

func SB_Reporte(path string, id string) string {
	// Obtener la ruta del disco
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		return Utils.Error("REP", "No se encontró el disco para la ID: "+id)
	}

	// Abrir el archivo del disco
	file, err := os.Open(diskPath)
	if err != nil {
		return Utils.Error("REP", "Error al abrir el disco: "+err.Error())
	}
	defer file.Close()

	// Obtener la partición montada
	particion := GetMount("REP", id, &diskPath)
	if particion == nil {
		return Utils.Error("REP", "No se encontró la partición montada")
	}

	// Leer el superbloque
	var sb Structs.SuperBloque
	file.Seek(particion.Part_start, 0)
	if err := binary.Read(file, binary.LittleEndian, &sb); err != nil {
		return Utils.Error("REP", "Error al leer el superbloque: "+err.Error())
	}

	// Obtener nombre del disco
	diskName := filepath.Base(diskPath)

	// Crear contenido DOT para Graphviz
	dotContent := "digraph G {\n"
	dotContent += "  node [shape=plaintext];\n"
	dotContent += "  rankdir=TB;\n\n"

	// Crear tabla HTML para el superbloque
	tableContent := `<<TABLE BORDER="1" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">
    <TR>
        <TD COLSPAN="2" BGCOLOR="#4CAF50"><FONT COLOR="white">REPORTE SUPERBLOQUE</FONT></TD>
    </TR>
    <TR>
        <TD BGCOLOR="#f2f2f2">Nombre</TD>
        <TD BGCOLOR="#f2f2f2">` + diskName + `</TD>
    </TR>`

	// Actualizar s_umtime a la fecha actual
	currentTime := time.Now().Format("2006-01-02 15:04")
	copy(sb.S_umtime[:], currentTime)

	// Definir los campos en el orden específico mostrado en la imagen
	rows := []struct {
		nombre string
		valor  interface{}
	}{
		{"s_inodes_count", sb.S_inodes_count},
		{"s_blocks_count", sb.S_blocks_count},
		{"s_free_blocks_count", sb.S_free_blocks_count},
		{"s_free_inodes_count", sb.S_free_inodes_count},
		{"s_mtime", string(bytes.Trim(sb.S_mtime[:], "\x00"))},
		{"s_umtime", string(bytes.Trim(sb.S_umtime[:], "\x00"))},
		{"s_mnt_count", sb.S_mnt_count},
		{"s_magic", fmt.Sprintf("0x%X", sb.S_magic)},
		{"s_inode_size", sb.S_inode_size},
		{"s_block_size", sb.S_block_size},
		{"s_first_ino", sb.S_firts_ino},
		{"s_first_blo", sb.S_first_blo},
		{"s_bm_inode_start", sb.S_bm_inode_start},
		{"s_bm_block_start", sb.S_bm_block_start},
		{"s_inode_start", sb.S_inode_start},
		{"s_block_start", sb.S_block_start},
	}

	// Agregar cada fila a la tabla
	for i, row := range rows {
		bgColor := "#ffffff"
		if i%2 == 0 {
			bgColor = "#f9f9f9"
		}
		tableContent += fmt.Sprintf(`
        <TR BGCOLOR="%s">
            <TD>%s</TD>
            <TD>%v</TD>
        </TR>`, bgColor, row.nombre, row.valor)
	}

	tableContent += "</TABLE>>"

	// Agregar el nodo con la tabla al grafo
	dotContent += fmt.Sprintf("  sb [label=%s];\n", tableContent)
	dotContent += "}\n"

	// Escribir archivo DOT
	dotPath := path + ".dot"
	if err := os.WriteFile(dotPath, []byte(dotContent), 0644); err != nil {
		return Utils.Error("REP", "Error escribiendo archivo DOT: "+err.Error())
	}

	// Generar imagen PNG usando Graphviz
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", path)
	if err := cmd.Run(); err != nil {
		return Utils.Error("REP", "Error generando imagen: "+err.Error())
	}

	// Eliminar archivo DOT temporal
	os.Remove(dotPath)

	return Utils.Mensaje("REP", "Reporte de SuperBloque generado exitosamente")
}

func File_Reporte(path string, id string, pathFile string) string {
	fmt.Printf("🔧 DEBUG: Generando reporte FILE (TXT) para archivo '%s' -> destino: %s\n", pathFile, path)

	// Crear directorio destino
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Utils.Error("REP", "Error creando directorio para reporte: "+err.Error())
	}

	// Buscar disco y partición
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		return Utils.Error("REP", "No se encontró el disco para la ID: "+id)
	}

	f, err := os.Open(diskPath)
	if err != nil {
		return Utils.Error("REP", "Error abriendo disco: "+err.Error())
	}
	defer f.Close()

	particion := GetMount("REP", id, &diskPath)
	if particion == nil {
		return Utils.Error("REP", "No se encontró la partición montada")
	}

	var sb Structs.SuperBloque
	f.Seek(particion.Part_start, 0)
	if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
		return Utils.Error("REP", "Error leyendo superbloque: "+err.Error())
	}

	// Validar ruta del archivo
	pathFile = strings.TrimSpace(pathFile)
	if !strings.HasPrefix(pathFile, "/") {
		return Utils.Error("REP", "La ruta del archivo debe ser absoluta")
	}

	components := strings.Split(pathFile, "/")[1:]

	// Navegar la ruta para obtener el inodo
	currentInodeIndex := int64(0)
	var currentInode Structs.Inodos
	for i, comp := range components {
		if comp == "" {
			continue
		}
		// leer inodo actual
		f.Seek(sb.S_inode_start+currentInodeIndex*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
		if err := binary.Read(f, binary.LittleEndian, &currentInode); err != nil {
			return Utils.Error("REP", "Error leyendo inodo: "+err.Error())
		}

		if i < len(components)-1 && currentInode.I_type != 0 {
			return Utils.Error("REP", "Un componente de la ruta no es un directorio")
		}

		found := false
		for b := 0; b < 12 && !found && currentInode.I_block[b] != -1; b++ {
			var bloqueCarpeta Structs.BloquesCarpetas
			f.Seek(sb.S_block_start+currentInode.I_block[b]*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
			if err := binary.Read(f, binary.LittleEndian, &bloqueCarpeta); err != nil {
				continue
			}
			for _, entry := range bloqueCarpeta.B_content {
				nombre := strings.Trim(string(entry.B_name[:]), "\x00")
				if nombre == comp && entry.B_inodo != -1 {
					currentInodeIndex = entry.B_inodo
					found = true
					break
				}
			}
		}
		if !found {
			return Utils.Error("REP", "No se encontró el archivo o directorio: "+comp)
		}
	}

	// Leer inodo final
	f.Seek(sb.S_inode_start+currentInodeIndex*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	if err := binary.Read(f, binary.LittleEndian, &currentInode); err != nil {
		return Utils.Error("REP", "Error leyendo inodo del archivo: "+err.Error())
	}
	if currentInode.I_type != 1 {
		return Utils.Error("REP", "La ruta no corresponde a un archivo")
	}

	// Leer contenido raw respetando I_size
	var buf bytes.Buffer
	var leidos int64 = 0
	for i := 0; i < 12 && currentInode.I_block[i] != -1 && leidos < currentInode.I_size; i++ {
		data, err := readFileBlock(f, sb, currentInode.I_block[i])
		if err != nil {
			continue
		}
		restante := currentInode.I_size - leidos
		take := int64(len(data))
		if restante < take {
			take = restante
		}
		if take > 0 {
			buf.Write(data[:take])
			leidos += take
		}
	}

	// Preparar contenido de salida
	nombre := components[len(components)-1]
	salida := fmt.Sprintf("NOMBRE: %s\n\n%s", nombre, buf.String())

	// Escribir archivo txt local
	if err := os.WriteFile(path, []byte(salida), 0644); err != nil {
		return Utils.Error("REP", "Error escribiendo archivo de reporte: "+err.Error())
	}

	fmt.Printf("✅ Reporte FILE escrito en: %s\n", path)
	return Utils.Mensaje("REP", "Reporte FILE generado exitosamente: "+path)
}

// Función auxiliar mejorada para obtener el contenido del archivo
func obtenerContenidoArchivo(file *os.File, sb Structs.SuperBloque, inodo Structs.Inodos) string {
	var buf bytes.Buffer

	// Leer respetando el tamaño real del archivo (I_size)
	var leidos int64 = 0
	for i := 0; i < 12 && inodo.I_block[i] != -1 && leidos < inodo.I_size; i++ {
		data, err := readFileBlock(file, sb, inodo.I_block[i])
		if err != nil {
			fmt.Printf("⚠️ Warning: Error leyendo bloque %d: %v\n", i, err)
			continue
		}

		restante := inodo.I_size - leidos
		take := int64(len(data))
		if restante < take {
			take = restante
		}

		if take > 0 {
			buf.Write(data[:take])
			leidos += take
		}
	}

	// Escapar caracteres especiales para DOT y convertir saltos de línea a <BR/>
	resultado := strings.ReplaceAll(buf.String(), "&", "&amp;")
	resultado = strings.ReplaceAll(resultado, "<", "&lt;")
	resultado = strings.ReplaceAll(resultado, ">", "&gt;")
	resultado = strings.ReplaceAll(resultado, "\"", "&quot;")
	resultado = strings.ReplaceAll(resultado, "\n", "<BR/>")

	if resultado == "" {
		resultado = "(archivo vacío)"
	}

	return resultado
}

// obtenerInodosEnUso lee el bitmap de inodos y devuelve un mapa con los inodos en uso
func obtenerInodosEnUso(file *os.File, superBloque Structs.SuperBloque) map[int64]bool {
	// Crear mapa de inodos en uso
	inodosEnUso := make(map[int64]bool)

	// Leer bitmap de inodos
	file.Seek(superBloque.S_bm_inode_start, 0)
	bitmapInodos := make([]byte, superBloque.S_inodes_count)
	if _, err := file.Read(bitmapInodos); err != nil {
		fmt.Println(Utils.Error("REP", "Error al leer bitmap de inodos: "+err.Error()))
		return inodosEnUso
	}

	// Marcar los inodos en uso
	for i := int64(0); i < superBloque.S_inodes_count; i++ {
		if i < int64(len(bitmapInodos)) && bitmapInodos[i] == '1' {
			inodosEnUso[i] = true
		}
	}

	return inodosEnUso
}

// generarConexionesBloqueCarpeta genera conexiones desde bloques carpeta a otros bloques
func generarConexionesBloqueCarpeta(file *os.File, sb Structs.SuperBloque, bloqueIdx int64) string {
	var bloqueCarpeta Structs.BloquesCarpetas
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueCarpeta); err != nil {
		return ""
	}

	dotContent := ""

	// Buscar entradas que apunten a inodos (que a su vez apuntan a bloques)
	for i, entry := range bloqueCarpeta.B_content {
		// Ignorar entradas inválidas, vacías o las especiales "." y ".."
		nombre := strings.Trim(string(entry.B_name[:]), "\x00")
		if entry.B_inodo <= 0 || nombre == "" || nombre == "." || nombre == ".." {
			continue
		}

		// Leer el inodo referenciado para encontrar su bloque principal
		var inodo Structs.Inodos
		file.Seek(sb.S_inode_start+entry.B_inodo*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
		if err := binary.Read(file, binary.LittleEndian, &inodo); err != nil {
			continue
		}

		// Si el inodo tiene un bloque principal, conectar con él
		if inodo.I_block[0] >= 0 {
			dotContent += fmt.Sprintf("  bloque_%d:e%d -> bloque_%d [label=\"%s\"];\n",
				bloqueIdx, i, inodo.I_block[0], nombre)
		}
	}

	return dotContent
}

// generarConexionesApuntador genera conexiones desde bloques apuntadores a otros bloques

// LS_Reporte genera un reporte que muestra información de archivos y carpetas con permisos, propietario, grupo, etc.
func LS_Reporte(path string, id string, pathDir string) {
	// Obtener la ruta del disco
	diskPath, found := GetDiskPathFromID(id)
	if !found {
		fmt.Println(Utils.Error("REP", "No se encontró el disco para la ID: "+id))
		return
	}

	// Abrir el archivo del disco
	f, err := os.OpenFile(diskPath, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println(Utils.Error("REP", "Error abriendo disco: "+err.Error()))
		return
	}
	defer f.Close()

	// Obtener la partición montada
	part := GetMount("REP", id, &diskPath)
	if part == nil {
		fmt.Println(Utils.Error("REP", "No se encontró la partición montada"))
		return
	}

	// Leer superbloque
	var sb Structs.SuperBloque
	f.Seek(part.Part_start, 0)
	if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
		fmt.Println(Utils.Error("REP", "Error leyendo superbloque: "+err.Error()))
		return
	}

	// Validar ruta
	pathDir = strings.TrimSpace(pathDir)
	if !strings.HasPrefix(pathDir, "/") {
		fmt.Println(Utils.Error("REP", "La ruta debe ser absoluta"))
		return
	}

	// Navegar la ruta para obtener el inodo del directorio
	comps := strings.Split(pathDir, "/")[1:]
	currentIdx := int64(0)
	var currentInode Structs.Inodos
	for _, comp := range comps {
		if comp == "" {
			continue
		}
		// leer inodo actual
		f.Seek(sb.S_inode_start+currentIdx*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
		if err := binary.Read(f, binary.LittleEndian, &currentInode); err != nil {
			fmt.Println(Utils.Error("REP", "Error leyendo inodo: "+err.Error()))
			return
		}

		if currentInode.I_type != 0 {
			fmt.Println(Utils.Error("REP", "Un componente de la ruta no es un directorio"))
			return
		}

		foundEntry := false
		for b := 0; b < 12 && !foundEntry && currentInode.I_block[b] != -1; b++ {
			var block Structs.BloquesCarpetas
			f.Seek(sb.S_block_start+currentInode.I_block[b]*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
			if err := binary.Read(f, binary.LittleEndian, &block); err != nil {
				continue
			}
			for _, entry := range block.B_content {
				name := strings.Trim(string(entry.B_name[:]), "\x00")
				if name == comp && entry.B_inodo != -1 {
					currentIdx = entry.B_inodo
					foundEntry = true
					break
				}
			}
		}
		if !foundEntry {
			fmt.Println(Utils.Error("REP", "No se encontró la ruta: "+comp))
			return
		}
	}

	// Leer inodo objetivo
	f.Seek(sb.S_inode_start+currentIdx*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	if err := binary.Read(f, binary.LittleEndian, &currentInode); err != nil {
		fmt.Println(Utils.Error("REP", "Error leyendo inodo final: "+err.Error()))
		return
	}
	if currentInode.I_type != 0 {
		fmt.Println(Utils.Error("REP", "La ruta no corresponde a un directorio"))
		return
	}

	// Leer users.txt para mapear UIDs/GIDs a nombres
	// Se asume que users.txt está en el inodo 1
	var inodeUsers Structs.Inodos
	f.Seek(sb.S_inode_start+1*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	if err := binary.Read(f, binary.LittleEndian, &inodeUsers); err != nil {
		fmt.Println(Utils.Error("REP", "Error leyendo inodo de users.txt: "+err.Error()))
		inodeUsers = Structs.NewInodos()
	}

	usersContent := leerContenidoUsersArchivo(f, sb, inodeUsers)

	// Mapear UIDs y GIDs a nombres
	uidToUser := make(map[string]string)
	gidToGroup := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(usersContent), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		id := parts[0]
		kind := parts[1]
		if kind == "G" && len(parts) >= 3 {
			gidToGroup[id] = parts[2]
		} else if kind == "U" && len(parts) >= 4 {
			uidToUser[id] = parts[3]
		}
	}

	// Helper para convertir permisos numéricos a rwx (notación tipo 750)
	permToStr := func(p int64, isDir bool) string {
		prefix := "-"
		if isDir {
			prefix = "d"
		}
		// interpretar p como decimal con tres cifras owner/group/other
		owner := (p / 100) % 10
		group := (p / 10) % 10
		other := p % 10
		part := func(d int64) string {
			s := ""
			if d&4 == 4 {
				s += "r"
			} else {
				s += "-"
			}
			if d&2 == 2 {
				s += "w"
			} else {
				s += "-"
			}
			if d&1 == 1 {
				s += "x"
			} else {
				s += "-"
			}
			return s
		}
		return prefix + part(owner) + part(group) + part(other)
	}

	// Construir tabla DOT con estilo mejorado
	dot := "digraph G {\n"
	dot += "  node [shape=plaintext];\n"
	dot += "  rankdir=TB;\n\n"

	table := `<<TABLE BORDER="1" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">`
	table += "<TR><TD COLSPAN=\"8\" BGCOLOR=\"#4CAF50\"><FONT COLOR=\"white\">Reporte LS: " + pathDir + "</FONT></TD></TR>"
	table += "<TR BGCOLOR=\"#f2f2f2\"><TD>Permisos</TD><TD>Owner</TD><TD>Grupo</TD><TD>Size (Bytes)</TD><TD>Fecha</TD><TD>Hora</TD><TD>Tipo</TD><TD>Name</TD></TR>"

	// Recorrer entradas del directorio usando todos los bloques directos
	entriesFound := false

	for b := 0; b < 12 && currentInode.I_block[b] != -1; b++ {
		var block Structs.BloquesCarpetas
		f.Seek(sb.S_block_start+currentInode.I_block[b]*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
		if err := binary.Read(f, binary.LittleEndian, &block); err != nil {
			fmt.Println(Utils.Error("REP", "Error leyendo bloque de carpeta: "+err.Error()))
			continue
		}

		for _, entry := range block.B_content {
			// Ignorar entradas inválidas o vacías
			if entry.B_inodo <= 0 {
				continue
			}

			name := strings.Trim(string(entry.B_name[:]), "\x00")
			if name == "" {
				continue
			}

			// Para directorios, incluir . y .. pero marcarlos con color especial
			isSpecial := name == "." || name == ".."

			// Leer inodo del entry
			var inode Structs.Inodos
			f.Seek(sb.S_inode_start+entry.B_inodo*int64(unsafe.Sizeof(Structs.Inodos{})), 0)
			if err := binary.Read(f, binary.LittleEndian, &inode); err != nil {
				fmt.Println(Utils.Error("REP", "Error leyendo inodo de entrada: "+err.Error()))
				continue
			}

			entriesFound = true
			isDir := inode.I_type == 0
			permisos := permToStr(inode.I_perm, isDir)

			// Mapear UID a nombre de usuario si existe
			owner := fmt.Sprintf("%d", inode.I_uid)
			if u, ok := uidToUser[owner]; ok {
				owner = u
			}

			// Mapear GID a nombre de grupo si existe
			group := fmt.Sprintf("%d", inode.I_gid)
			if g, ok := gidToGroup[group]; ok {
				group = g
			}

			size := fmt.Sprintf("%d", inode.I_size)

			// Extraer fecha y hora de las marcas de tiempo
			mtime := strings.TrimRight(string(inode.I_mtime[:]), "\x00")
			ctime := strings.TrimRight(string(inode.I_ctime[:]), "\x00")
			fecha := ""
			hora := ""

			// Intentar usar mtime primero, luego ctime como respaldo
			if mtime != "" {
				parts := strings.Split(mtime, " ")
				if len(parts) >= 2 {
					fecha = parts[0]
					hora = parts[1]
				} else {
					fecha = mtime
				}
			} else if ctime != "" {
				parts := strings.Split(ctime, " ")
				if len(parts) >= 2 {
					fecha = parts[0]
					hora = parts[1]
				} else {
					fecha = ctime
				}
			}

			tipo := "Archivo"
			if isDir {
				tipo = "Carpeta"
			}

			// Función para escapar caracteres especiales en HTML/DOT
			escape := func(s string) string {
				s = strings.ReplaceAll(s, "&", "&amp;")
				s = strings.ReplaceAll(s, "<", "&lt;")
				s = strings.ReplaceAll(s, ">", "&gt;")
				s = strings.ReplaceAll(s, "\"", "&quot;")
				s = strings.ReplaceAll(s, "\n", " ")
				return s
			}

			// Color de fondo para entradas especiales (. y ..)
			bgColor := ""
			if isSpecial {
				bgColor = " BGCOLOR=\"#e6f7ff\""
			}

			// Añadir fila a la tabla
			table += fmt.Sprintf("<TR%s><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD></TR>",
				bgColor,
				escape(permisos),
				escape(owner),
				escape(group),
				escape(size),
				escape(fecha),
				escape(hora),
				escape(tipo),
				escape(name))
		}
	}

	// Si no se encontraron entradas, añadir mensaje informativo
	if !entriesFound {
		table += "<TR><TD COLSPAN=\"8\">Directorio vacío</TD></TR>"
	}

	table += "</TABLE>>"
	dot += fmt.Sprintf("  ls [label=%s];\n", table)
	dot += "}\n"

	// Escribir archivo DOT
	dotPath := path + ".dot"
	if err := os.WriteFile(dotPath, []byte(dot), 0644); err != nil {
		fmt.Println(Utils.Error("REP", "Error escribiendo archivo DOT: "+err.Error()))
		return
	}

	// Generar imagen PNG usando Graphviz
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(Utils.Error("REP", "Error generando imagen: "+err.Error()))
		fmt.Println("Output:", string(output))
		return
	}

	// Eliminar .dot temporal si la generación fue exitosa
	os.Remove(dotPath)

	fmt.Println(Utils.Mensaje("REP", "Reporte LS generado correctamente en: "+path))
}
func generarConexionesApuntador(file *os.File, sb Structs.SuperBloque, bloqueIdx int64) string {
	var bloqueApuntadores Structs.BloquesApuntadores
	file.Seek(sb.S_block_start+bloqueIdx*int64(unsafe.Sizeof(Structs.BloquesCarpetas{})), 0)
	if err := binary.Read(file, binary.LittleEndian, &bloqueApuntadores); err != nil {
		return ""
	}

	dotContent := ""

	// Conectar con cada bloque referenciado
	for i, ptr := range bloqueApuntadores.B_pointers {
		if ptr >= 0 && ptr < sb.S_blocks_count {
			dotContent += fmt.Sprintf("  bloque_%d:p%d -> bloque_%d;\n",
				bloqueIdx, i, ptr)
		}
	}

	return dotContent
}
