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

// ValidarDatosMKGRP valida los parámetros del comando MKGRP
func ValidarDatosMKGRP(tokens []string) string {
	if len(tokens) < 1 {
		return Utils.Error("MKGRP", "Se requiere el parámetro -name")
	}

	var nombre string

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
			nombre = value
		default:
			return Utils.Error("MKGRP", "Parámetro no reconocido: "+param)
		}
	}

	// Validaciones
	if nombre == "" {
		return Utils.Error("MKGRP", "El parámetro -name es obligatorio")
	}

	// Verificar que hay una sesión activa
	if !EstaLogueado() {
		return Utils.Error("MKGRP", "Debe iniciar sesión para ejecutar este comando")
	}

	// Verificar que el usuario es root
	if !EsUsuarioRoot() {
		return Utils.Error("MKGRP", "Solo el usuario root puede crear grupos")
	}

	return mkgrp(nombre)
}

// ValidarDatosRMGRP valida los parámetros del comando RMGRP
func ValidarDatosRMGRP(tokens []string) string {
	if len(tokens) < 1 {
		return Utils.Error("RMGRP", "Se requiere el parámetro -name")
	}

	var nombre string

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
			nombre = value
		default:
			return Utils.Error("RMGRP", "Parámetro no reconocido: "+param)
		}
	}

	// Validaciones
	if nombre == "" {
		return Utils.Error("RMGRP", "El parámetro -name es obligatorio")
	}

	// Verificar que hay una sesión activa
	if !EstaLogueado() {
		return Utils.Error("RMGRP", "Debe iniciar sesión para ejecutar este comando")
	}

	// Verificar que el usuario es root
	if !EsUsuarioRoot() {
		return Utils.Error("RMGRP", "Solo el usuario root puede eliminar grupos")
	}

	return rmgrp(nombre)
}

// mkgrp crea un nuevo grupo en el sistema
func mkgrp(nombre string) string {
	fmt.Printf("🔧 DEBUG: Creando grupo '%s'\n", nombre)

	sesion := ObtenerSesionActiva()

	// Obtener la partición montada de la sesión activa
	var pathDisco string
	particion := GetMount("MKGRP", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("MKGRP", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	fmt.Printf("🔧 DEBUG: Partición encontrada en: %s\n", pathDisco)

	// Abrir archivo para lectura primero
	file, err := os.Open(strings.ReplaceAll(pathDisco, "\"", ""))
	if err != nil {
		return Utils.Error("MKGRP", "No se encontró el disco: "+err.Error())
	}
	defer file.Close()

	// Leer SuperBloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	data := leerBytesGroups(file, int(unsafe.Sizeof(Structs.SuperBloque{})))
	buffer := bytes.NewBuffer(data)
	err_ := binary.Read(buffer, binary.BigEndian, &super)
	if err_ != nil {
		return Utils.Error("MKGRP", "Error al leer superbloque: "+err_.Error())
	}

	fmt.Printf("🔧 DEBUG: SuperBloque leído correctamente\n")

	// Leer inodo del archivo users.txt (inodo 1)
	inodo := Structs.NewInodos()
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	data = leerBytesGroups(file, int(unsafe.Sizeof(Structs.Inodos{})))
	buffer = bytes.NewBuffer(data)
	err_ = binary.Read(buffer, binary.BigEndian, &inodo)
	if err_ != nil {
		return Utils.Error("MKGRP", "Error al leer inodo users.txt: "+err_.Error())
	}

	fmt.Printf("🔧 DEBUG: Inodo users.txt leído - Tamaño: %d\n", inodo.I_size)

	// ✅ USAR FUNCIÓN DE LOGIN.GO - Leer contenido actual de users.txt
	contenidoActual := leerContenidoUsersArchivo(file, super, inodo)
	if contenidoActual == "" {
		return Utils.Error("MKGRP", "No se pudo leer el archivo users.txt")
	}

	fmt.Printf("🔧 DEBUG: Contenido actual users.txt:\n%s\n", contenidoActual)

	// Verificar si el grupo ya existe y contar grupos
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	contadorGrupos := 0

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if len(linea) < 3 {
			continue
		}

		if linea[2] == 'G' || linea[2] == 'g' {
			contadorGrupos++
			campos := strings.Split(linea, ",")
			if len(campos) >= 3 {
				nombreGrupo := campos[2]
				if nombreGrupo == nombre {
					if linea[0] != '0' { // Si no está eliminado
						return Utils.Error("MKGRP", "El grupo '"+nombre+"' ya existe")
					}
				}
			}
		}
	}

	// Crear nueva línea del grupo
	nuevoID := contadorGrupos + 1
	nuevaLinea := fmt.Sprintf("%d,G,%s\n", nuevoID, nombre)

	// Agregar la nueva línea al contenido
	nuevoContenido := contenidoActual + nuevaLinea

	fmt.Printf("🔧 DEBUG: Nuevo contenido users.txt:\n%s\n", nuevoContenido)

	// Escribir el contenido actualizado
	if err := escribirContenidoUsers(pathDisco, *particion, super, inodo, nuevoContenido); err != nil {
		return Utils.Error("MKGRP", "Error al escribir en users.txt: "+err.Error())
	}

	fmt.Printf("✅ MKGRP: Grupo '%s' creado correctamente\n", nombre)
	return Utils.Mensaje("MKGRP", fmt.Sprintf("Grupo '%s' creado correctamente", nombre))
}

// rmgrp elimina un grupo del sistema
func rmgrp(nombre string) string {
	fmt.Printf("🔧 DEBUG: Eliminando grupo '%s'\n", nombre)

	sesion := ObtenerSesionActiva()

	// Obtener la partición montada de la sesión activa
	var pathDisco string
	particion := GetMount("RMGRP", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("RMGRP", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	fmt.Printf("🔧 DEBUG: Partición encontrada en: %s\n", pathDisco)

	// Abrir archivo para lectura primero
	file, err := os.Open(strings.ReplaceAll(pathDisco, "\"", ""))
	if err != nil {
		return Utils.Error("RMGRP", "No se encontró el disco: "+err.Error())
	}
	defer file.Close()

	// Leer SuperBloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	data := leerBytesGroups(file, int(unsafe.Sizeof(Structs.SuperBloque{})))
	buffer := bytes.NewBuffer(data)
	err_ := binary.Read(buffer, binary.BigEndian, &super)
	if err_ != nil {
		return Utils.Error("RMGRP", "Error al leer superbloque: "+err_.Error())
	}

	// Leer inodo del archivo users.txt (inodo 1)
	inodo := Structs.NewInodos()
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	data = leerBytesGroups(file, int(unsafe.Sizeof(Structs.Inodos{})))
	buffer = bytes.NewBuffer(data)
	err_ = binary.Read(buffer, binary.BigEndian, &inodo)
	if err_ != nil {
		return Utils.Error("RMGRP", "Error al leer inodo users.txt: "+err_.Error())
	}

	// ✅ USAR FUNCIÓN DE LOGIN.GO - Leer contenido actual de users.txt
	contenidoActual := leerContenidoUsersArchivo(file, super, inodo)
	if contenidoActual == "" {
		return Utils.Error("RMGRP", "No se pudo leer el archivo users.txt")
	}

	fmt.Printf("🔧 DEBUG: Contenido actual users.txt:\n%s\n", contenidoActual)

	// Procesar líneas y marcar grupo como eliminado
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	var nuevasLineas []string
	grupoEncontrado := false

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		if len(linea) >= 3 && (linea[2] == 'G' || linea[2] == 'g') && linea[0] != '0' {
			campos := strings.Split(linea, ",")
			if len(campos) >= 3 && campos[2] == nombre {
				// Marcar como eliminado (cambiar ID a 0)
				nuevaLinea := fmt.Sprintf("0,G,%s", campos[2])
				nuevasLineas = append(nuevasLineas, nuevaLinea)
				grupoEncontrado = true
				fmt.Printf("🔧 DEBUG: Grupo '%s' marcado como eliminado\n", nombre)
				continue
			}
		}

		nuevasLineas = append(nuevasLineas, linea)
	}

	if !grupoEncontrado {
		return Utils.Error("RMGRP", "No se encontró el grupo '"+nombre+"'")
	}

	// Reconstruir contenido
	nuevoContenido := strings.Join(nuevasLineas, "\n") + "\n"

	fmt.Printf("🔧 DEBUG: Nuevo contenido users.txt:\n%s\n", nuevoContenido)

	// Escribir el contenido actualizado
	if err := escribirContenidoUsers(pathDisco, *particion, super, inodo, nuevoContenido); err != nil {
		return Utils.Error("RMGRP", "Error al escribir en users.txt: "+err.Error())
	}

	fmt.Printf("✅ RMGRP: Grupo '%s' eliminado correctamente\n", nombre)
	return Utils.Mensaje("RMGRP", fmt.Sprintf("Grupo '%s' eliminado correctamente", nombre))
}

// escribirContenidoUsers escribe el contenido actualizado en users.txt
func escribirContenidoUsers(pathDisco string, particion Structs.Particion, super Structs.SuperBloque, inodo Structs.Inodos, nuevoContenido string) error {
	// Dividir contenido en bloques de 64 bytes
	tamBloque := 64
	var bloques []string

	contenido := nuevoContenido
	for len(contenido) > tamBloque {
		bloques = append(bloques, contenido[:tamBloque])
		contenido = contenido[tamBloque:]
	}

	if len(contenido) > 0 {
		bloques = append(bloques, contenido)
	}

	if len(bloques) > 16 {
		return fmt.Errorf("contenido demasiado grande para el archivo")
	}

	// Abrir archivo para escritura
	file, err := os.OpenFile(strings.ReplaceAll(pathDisco, "\"", ""), os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// Calcular posiciones
	mitadBA := super.S_block_start + int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	tamBA := int64(unsafe.Sizeof(Structs.BloquesArchivos{}))

	// Escribir cada bloque
	for i, contenidoBloque := range bloques {
		var bloqueArchivo Structs.BloquesArchivos

		// Si necesitamos un nuevo bloque
		if inodo.I_block[i] == -1 {
			super.S_first_blo++
			inodo.I_block[i] = super.S_first_blo

			// Actualizar superbloque
			file.Seek(particion.Part_start, 0)
			var bufferSuper bytes.Buffer
			binary.Write(&bufferSuper, binary.BigEndian, super)
			file.Write(bufferSuper.Bytes())
		}

		// Copiar contenido al bloque
		copy(bloqueArchivo.B_content[:], contenidoBloque)

		// Escribir bloque
		posicionBloque := mitadBA + (int64(inodo.I_block[i]-1) * tamBA)
		file.Seek(posicionBloque, 0)

		var bufferBloque bytes.Buffer
		binary.Write(&bufferBloque, binary.BigEndian, bloqueArchivo)
		file.Write(bufferBloque.Bytes())
	}

	// Actualizar tamaño del inodo
	inodo.I_size = int64(len(nuevoContenido))

	// Escribir inodo actualizado
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	var bufferInodo bytes.Buffer
	binary.Write(&bufferInodo, binary.BigEndian, inodo)
	file.Write(bufferInodo.Bytes())

	file.Sync()
	return nil
}

// leerBytesGroups función auxiliar para leer bytes del archivo (renombrada para evitar conflictos)
func leerBytesGroups(file *os.File, size int) []byte {
	bytes := make([]byte, size)
	file.Read(bytes)
	return bytes
}
