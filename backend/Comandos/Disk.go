package Comandos

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"godisk-backend/Structs"
	"godisk-backend/Utils"
)

// ValidarDatosMKDISK valida los parámetros del comando MKDISK
func ValidarDatosMKDISK(tokens []string) string {
	size := ""
	fit := "FF" // Valor por defecto
	unit := "M" // Valor por defecto
	path := ""

	// Parsear tokens
	for i := 0; i < len(tokens); i++ {
		datos := strings.Split(tokens[i], "=")
		if len(datos) != 2 {
			continue
		}

		param := strings.ToLower(datos[0])
		value := datos[1]

		switch param {
		case "size":
			size = value
		case "fit":
			fit = strings.ToUpper(value)
		case "unit":
			unit = strings.ToUpper(value)
		case "path":
			path = value
		}
	}

	// Validaciones
	if path == "" || size == "" {
		return Utils.Error("MKDISK", "Se requieren parámetros path y size")
	}

	if !Utils.Comparar(fit, "BF") && !Utils.Comparar(fit, "FF") && !Utils.Comparar(fit, "WF") {
		return Utils.Error("MKDISK", "Valores en parámetro fit no válidos (BF, FF, WF)")
	}

	if !Utils.Comparar(unit, "K") && !Utils.Comparar(unit, "M") {
		return Utils.Error("MKDISK", "Valores en parámetro unit no válidos (K, M)")
	}

	// Ejecutar MKDISK
	return MKDISK(size, fit, unit, path)
}

// MKDISK crea un nuevo disco
func MKDISK(s, f, u, path string) string {
	// Validar que la extensión sea .mia
	if !strings.HasSuffix(strings.ToLower(path), ".mia") {
		return Utils.Error("MKDISK", "La extensión del archivo debe ser .mia")
	}

	// Convertir tamaño
	size, err := strconv.Atoi(s)
	if err != nil {
		return Utils.Error("MKDISK", "Size debe ser un número entero")
	}
	if size <= 0 {
		return Utils.Error("MKDISK", "Size debe ser mayor a 0")
	}

	// Calcular tamaño en bytes
	sizeBytes := int64(size)
	if Utils.Comparar(u, "M") {
		sizeBytes = sizeBytes * 1024 * 1024
	} else if Utils.Comparar(u, "K") {
		sizeBytes = sizeBytes * 1024
	}

	// Crear directorio si no existe
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Utils.Error("MKDISK", "No se pudieron crear los directorios: "+err.Error())
	}

	// Si el archivo existe, eliminarlo
	if Utils.ArchivoExiste(path) {
		if err := os.Remove(path); err != nil {
			return Utils.Error("MKDISK", "No se pudo eliminar el archivo existente: "+err.Error())
		}
	}

	// Crear archivo
	file, err := os.Create(path)
	if err != nil {
		return Utils.Error("MKDISK", "No se pudo crear el archivo: "+err.Error())
	}
	defer file.Close()

	// Llenar archivo con ceros
	buffer := make([]byte, 1024) // Buffer de 1KB
	for written := int64(0); written < sizeBytes; {
		toWrite := int64(len(buffer))
		if written+toWrite > sizeBytes {
			toWrite = sizeBytes - written
		}

		n, err := file.Write(buffer[:toWrite])
		if err != nil {
			return Utils.Error("MKDISK", "Error al escribir en el archivo: "+err.Error())
		}
		written += int64(n)
	}

	// Crear y configurar MBR
	mbr := Structs.NewMBR()
	mbr.Mbr_tamano = sizeBytes

	// Fecha de creación
	fecha := time.Now().Format("2006-01-02 15:04")
	copy(mbr.Mbr_fecha_creacion[:], fecha)

	// Signature aleatoria
	signature, _ := rand.Int(rand.Reader, big.NewInt(999999999))
	mbr.Mbr_dsk_signature = signature.Int64()

	// Fit
	copy(mbr.Dsk_fit[:], f)

	// Escribir MBR al inicio del archivo
	file.Seek(0, 0)
	var buffer_mbr bytes.Buffer
	if err := binary.Write(&buffer_mbr, binary.LittleEndian, &mbr); err != nil {
		return Utils.Error("MKDISK", "Error al escribir MBR: "+err.Error())
	}

	if _, err := file.Write(buffer_mbr.Bytes()); err != nil {
		return Utils.Error("MKDISK", "Error al escribir MBR en el archivo: "+err.Error())
	}

	// Mensaje de éxito
	nombreDisco := filepath.Base(path)
	return Utils.Mensaje("MKDISK", fmt.Sprintf("Disco \"%s\" creado correctamente!\n   📁 Ruta: %s\n   💾 Tamaño: %d bytes (%s %s)\n   🔧 Ajuste: %s",
		nombreDisco, path, sizeBytes, s, u, f))
}

// RMDISK elimina un disco
func RMDISK(tokens []string) string {
	path := ""

	// Parsear tokens
	for i := 0; i < len(tokens); i++ {
		datos := strings.Split(tokens[i], "=")
		if len(datos) == 2 && Utils.Comparar(datos[0], "path") {
			path = datos[1]
			break
		}
	}

	if path == "" {
		return Utils.Error("RMDISK", "Se requiere parámetro path")
	}

	if !Utils.ArchivoExiste(path) {
		return Utils.Error("RMDISK", "No se encontró el disco en la ruta indicada")
	}

	if !strings.HasSuffix(strings.ToLower(path), ".mia") {
		return Utils.Error("RMDISK", "Extensión de archivo no válida")
	}

	// Eliminar archivo (sin confirmación en contexto web)
	if err := os.Remove(path); err != nil {
		return Utils.Error("RMDISK", "Se produjo un error al eliminar el disco: "+err.Error())
	}

	return Utils.Mensaje("RMDISK", "Disco ubicado en "+path+" ha sido eliminado exitosamente")
}
