package Comandos

import (
	"fmt"
	"strings"

	"godisk-backend/Utils"
)

// ValidarDatosREP valida los parámetros del comando REP
func ValidarDatosREP(tokens []string) string {
	if len(tokens) < 3 {
		return Utils.Error("REP", "Se requieren al menos 3 parámetros para este comando.")
	}

	name := ""
	path := ""
	id := ""
	pathFile := ""

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
				name = strings.ToUpper(value)
			} else {
				return Utils.Error("REP", "parámetro name repetido")
			}
		case "path":
			if path == "" {
				path = value
			} else {
				return Utils.Error("REP", "parámetro path repetido")
			}
		case "id":
			if id == "" {
				id = value
			} else {
				return Utils.Error("REP", "parámetro id repetido")
			}
		case "path_file_ls":
			if pathFile == "" {
				pathFile = value
			} else {
				return Utils.Error("REP", "parámetro path_file_ls repetido")
			}
		default:
			return Utils.Error("REP", "parámetro no reconocido: "+param)
		}
	}

	// Validaciones básicas
	if name == "" || path == "" || id == "" {
		return Utils.Error("REP", "Los parámetros name, path e id son obligatorios")
	}

	// Validar tipos de reporte
	validReports := []string{"MBR", "DISK", "INODE", "JOURNALING", "BLOCK", "BM_INODE", "BM_BLOCK", "TREE", "SB", "FILE", "LS"}
	if !Utils.ValidarParametro(name, validReports) {
		return Utils.Error("REP", "Tipos de reporte válidos: MBR, DISK, INODE, JOURNALING, BLOCK, BM_INODE, BM_BLOCK, TREE, SB, FILE, LS")
	}

	// Ejecutar reporte
	return generarReporte(name, path, id, pathFile)
}

// generarReporte genera el reporte solicitado
func generarReporte(name, path, id, pathFile string) string {
	fmt.Printf("🔧 DEBUG: Generando reporte - Type: %s, Path: %s, ID: %s\n", name, path, id)

	// Verificar que la partición esté montada
	diskPath := ""
	particion := GetMount("REP", id, &diskPath)
	if particion == nil {
		return Utils.Error("REP", "La partición no está montada o el ID es inválido")
	}

	// Crear directorio de destino si no existe
	if err := Utils.CrearDirectorio(path); err != nil {
		return Utils.Error("REP", "Error al crear directorio: "+err.Error())
	}

	// Generar reporte según el tipo
	switch name {
	case "MBR":
		return generarReporteMBR(path, diskPath, id)
	case "DISK":
		return generarReporteDisk(path, diskPath, id)
	case "INODE":
		return generarReporteInode(path, diskPath, id)
	case "JOURNALING":
		return generarReporteJournaling(path, diskPath, id)
	case "BLOCK":
		return generarReporteBlock(path, diskPath, id)
	case "BM_INODE":
		return generarReporteBMInode(path, diskPath, id)
	case "BM_BLOCK":
		return generarReporteBMBlock(path, diskPath, id)
	case "TREE":
		return generarReporteTree(path, diskPath, id)
	case "SB":
		return generarReporteSB(path, diskPath, id)
	case "FILE":
		if pathFile == "" {
			return Utils.Error("REP", "El reporte FILE requiere el parámetro path_file_ls")
		}
		return generarReporteFile(path, diskPath, id, pathFile)
	case "LS":
		if pathFile == "" {
			return Utils.Error("REP", "El reporte LS requiere el parámetro path_file_ls")
		}
		return generarReporteLS(path, diskPath, id, pathFile)
	default:
		return Utils.Error("REP", "Tipo de reporte no implementado: "+name)
	}
}

// Funciones de generación de reportes (implementaciones básicas)

func generarReporteMBR(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte MBR
	return Utils.Mensaje("REP", "Reporte MBR generado correctamente (pendiente de implementar)")
}

func generarReporteDisk(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte DISK
	return Utils.Mensaje("REP", "Reporte DISK generado correctamente (pendiente de implementar)")
}

func generarReporteInode(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte INODE
	return Utils.Mensaje("REP", "Reporte INODE generado correctamente (pendiente de implementar)")
}

func generarReporteJournaling(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte JOURNALING
	return Utils.Mensaje("REP", "Reporte JOURNALING generado correctamente (pendiente de implementar)")
}

func generarReporteBlock(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte BLOCK
	return Utils.Mensaje("REP", "Reporte BLOCK generado correctamente (pendiente de implementar)")
}

func generarReporteBMInode(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte BM_INODE
	return Utils.Mensaje("REP", "Reporte BM_INODE generado correctamente (pendiente de implementar)")
}

func generarReporteBMBlock(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte BM_BLOCK
	return Utils.Mensaje("REP", "Reporte BM_BLOCK generado correctamente (pendiente de implementar)")
}

func generarReporteTree(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte TREE
	return Utils.Mensaje("REP", "Reporte TREE generado correctamente (pendiente de implementar)")
}

func generarReporteSB(outputPath, diskPath, id string) string {
	// TODO: Implementar reporte SB
	return Utils.Mensaje("REP", "Reporte SB generado correctamente (pendiente de implementar)")
}

func generarReporteFile(outputPath, diskPath, id, pathFile string) string {
	// TODO: Implementar reporte FILE
	return Utils.Mensaje("REP", "Reporte FILE generado correctamente (pendiente de implementar)")
}

func generarReporteLS(outputPath, diskPath, id, pathFile string) string {
	// TODO: Implementar reporte LS
	return Utils.Mensaje("REP", "Reporte LS generado correctamente (pendiente de implementar)")
}
