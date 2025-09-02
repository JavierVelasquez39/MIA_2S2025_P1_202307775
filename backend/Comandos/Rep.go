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
		File_Reporte(path, id, path_file_ls)
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

	for currentPos > 0 {
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

					// Conectar EBR con su partición lógica
					dotContent += fmt.Sprintf("    %s -> %s;\n", ebrID, logicaID)

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

				// Si hay nodos lógicos, conectar el último con el espacio libre
				if len(nodosLogicos) > 0 {
					dotContent += fmt.Sprintf("    %s -> libre_ext [style=dashed];\n",
						nodosLogicos[len(nodosLogicos)-1])
				}
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

func ReporteTree(path string, id string) {
	// Implementation will go here
}

func BitMap_inodo(path string, id string) {
	// Implementation will go here
}

func BitMap_block(path string, id string) {
	// Implementation will go here
}

func Report_Inode(path string, id string) {
	// Implementation will go here
}

func Report_Block(path string, id string) {
	// Implementation will go here
}

func SB_Reporte(path string, id string) {
	// Implementation will go here
}

func File_Reporte(path string, id string, pathFile string) {
	// Implementation will go here
}

func LS_Reporte(path string, id string, pathDir string) {
	// Implementation will go here
}
