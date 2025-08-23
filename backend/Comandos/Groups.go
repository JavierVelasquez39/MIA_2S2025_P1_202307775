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

// ValidarDatosCHGRP valida los parámetros del comando CHGRP
func ValidarDatosCHGRP(tokens []string) string {
	if len(tokens) < 2 {
		return Utils.Error("CHGRP", "Se requieren los parámetros: -user, -grp")
	}

	var usuario, grupo string

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
		case "grp":
			grupo = value
		default:
			return Utils.Error("CHGRP", "Parámetro no reconocido: "+param)
		}
	}

	if usuario == "" {
		return Utils.Error("CHGRP", "El parámetro -user es obligatorio")
	}
	if grupo == "" {
		return Utils.Error("CHGRP", "El parámetro -grp es obligatorio")
	}

	// Validar longitud máxima (según especificación: máximo 10 caracteres)
	if len(usuario) > 10 {
		return Utils.Error("CHGRP", "El nombre de usuario no puede exceder 10 caracteres")
	}
	if len(grupo) > 10 {
		return Utils.Error("CHGRP", "El nombre del grupo no puede exceder 10 caracteres")
	}

	// Verificar que hay una sesión activa
	if !EstaLogueado() {
		return Utils.Error("CHGRP", "Debe iniciar sesión para ejecutar este comando")
	}

	// Verificar que el usuario es root
	if !EsUsuarioRoot() {
		return Utils.Error("CHGRP", "Solo el usuario \"root\" puede acceder a estos comandos")
	}

	return chgrp(usuario, grupo)
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

	// Verificar si el grupo ya existe y obtener el max ID (ignorar entradas marcadas como eliminadas)
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	maxID := 0

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if len(linea) < 3 {
			continue
		}

		if linea[2] == 'G' || linea[2] == 'g' {
			campos := strings.Split(linea, ",")
			if len(campos) >= 3 {
				idStr := strings.TrimSpace(campos[0])
				if idStr != "0" {
					if id, err := strconv.Atoi(idStr); err == nil {
						if id > maxID {
							maxID = id
						}
					}
				}

				nombreGrupo := strings.TrimSpace(campos[2])
				if nombreGrupo == nombre {
					return Utils.Error("MKGRP", "El grupo '"+nombre+"' ya existe")
				}
			}
		}
	}

	// Crear nueva línea del grupo usando maxID+1 para evitar duplicados
	nuevoID := maxID + 1
	nuevaLinea := fmt.Sprintf("%d,G,%s\n", nuevoID, nombre)

	// Agregar la nueva línea al contenido
	nuevoContenido := contenidoActual + nuevaLinea

	fmt.Printf("🔧 DEBUG: Nuevo contenido users.txt:\n%s\n", nuevoContenido)

	// Escribir el contenido actualizado
	if err := escribirContenidoArchivo(pathDisco, *particion, super, inodo, nuevoContenido); err != nil {
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
	if err := escribirContenidoArchivo(pathDisco, *particion, super, inodo, nuevoContenido); err != nil {
		return Utils.Error("RMGRP", "Error al escribir en users.txt: "+err.Error())
	}

	fmt.Printf("✅ RMGRP: Grupo '%s' eliminado correctamente\n", nombre)
	return Utils.Mensaje("RMGRP", fmt.Sprintf("Grupo '%s' eliminado correctamente", nombre))
}

// chgrp cambia el grupo de un usuario en users.txt
func chgrp(usuario, grupo string) string {
	fmt.Printf("🔧 DEBUG: Cambiando grupo del usuario '%s' a '%s'\n", usuario, grupo)

	sesion := ObtenerSesionActiva()

	// Obtener la partición montada de la sesión activa
	var pathDisco string
	particion := GetMount("CHGRP", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("CHGRP", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	fmt.Printf("🔧 DEBUG: Partición encontrada en: %s\n", pathDisco)

	// Abrir archivo para lectura primero
	file, err := os.Open(strings.ReplaceAll(pathDisco, "\"", ""))
	if err != nil {
		return Utils.Error("CHGRP", "No se encontró el disco: "+err.Error())
	}
	defer file.Close()

	// Leer SuperBloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	data := leerBytesGroups(file, int(unsafe.Sizeof(Structs.SuperBloque{})))
	buffer := bytes.NewBuffer(data)
	err_ := binary.Read(buffer, binary.BigEndian, &super)
	if err_ != nil {
		return Utils.Error("CHGRP", "Error al leer superbloque: "+err_.Error())
	}

	fmt.Printf("🔧 DEBUG: SuperBloque leído correctamente\n")

	// Leer inodo del archivo users.txt (inodo 1)
	inodo := Structs.NewInodos()
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	data = leerBytesGroups(file, int(unsafe.Sizeof(Structs.Inodos{})))
	buffer = bytes.NewBuffer(data)
	err_ = binary.Read(buffer, binary.BigEndian, &inodo)
	if err_ != nil {
		return Utils.Error("CHGRP", "Error al leer inodo users.txt: "+err_.Error())
	}

	fmt.Printf("🔧 DEBUG: Inodo users.txt leído - Tamaño: %d\n", inodo.I_size)

	// Leer contenido actual
	contenidoActual := leerContenidoUsersArchivo(file, super, inodo)
	if contenidoActual == "" {
		return Utils.Error("CHGRP", "No se pudo leer el archivo users.txt")
	}

	fmt.Printf("🔧 DEBUG: Contenido actual users.txt:\n%s\n", contenidoActual)

	// Buscar existencia de usuario y del grupo destino (mejorando validaciones)
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	usuarioEncontrado := false
	grupoExiste := false

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		if len(linea) >= 3 && (linea[2] == 'G' || linea[2] == 'g') {
			campos := strings.Split(linea, ",")
			if len(campos) >= 3 {
				grupoNombre := strings.TrimSpace(campos[2])
				if grupoNombre == grupo && strings.TrimSpace(campos[0]) != "0" {
					grupoExiste = true
					fmt.Printf("🔧 DEBUG: Grupo destino '%s' encontrado\n", grupo)
				}
			}
		}

		if len(linea) >= 3 && (linea[2] == 'U' || linea[2] == 'u') {
			campos := strings.Split(linea, ",")
			if len(campos) >= 4 {
				username := strings.TrimSpace(campos[3])
				if username == usuario && strings.TrimSpace(campos[0]) != "0" {
					usuarioEncontrado = true
					fmt.Printf("🔧 DEBUG: Usuario '%s' encontrado en línea: %s\n", usuario, linea)
				}
			}
		}
	}

	if !usuarioEncontrado {
		return Utils.Error("CHGRP", "No se encontró el usuario '"+usuario+"'.")
	}
	if !grupoExiste {
		return Utils.Error("CHGRP", "No se encontró el grupo '"+grupo+"'.")
	}

	// Modificar la línea del usuario (reconstruir con validaciones)
	var nuevasLineas []string
	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		if len(linea) >= 3 && (linea[2] == 'U' || linea[2] == 'u') {
			campos := strings.Split(linea, ",")
			if len(campos) >= 5 {
				username := strings.TrimSpace(campos[3])
				password := strings.TrimSpace(campos[4])
				if username == usuario && strings.TrimSpace(campos[0]) != "0" {
					// Reconstruir línea con nuevo grupo (mantener id y password)
					nueva := strings.TrimSpace(campos[0]) + ",U," + grupo + "," + username + "," + password
					nuevasLineas = append(nuevasLineas, nueva)
					continue
				}
			}
		}

		nuevasLineas = append(nuevasLineas, linea)
	}

	// Reconstruir contenido
	nuevoContenido := strings.Join(nuevasLineas, "\n") + "\n"

	fmt.Printf("🔧 DEBUG: Nuevo contenido users.txt:\n%s\n", nuevoContenido)

	// Escribir cambios con la función compartida
	if err := escribirContenidoArchivo(pathDisco, *particion, super, inodo, nuevoContenido); err != nil {
		return Utils.Error("CHGRP", "Error al escribir en users.txt: "+err.Error())
	}

	fmt.Printf("✅ CHGRP: Grupo del usuario '%s' cambiado a '%s' correctamente\n", usuario, grupo)
	return Utils.Mensaje("CHGRP", fmt.Sprintf("Grupo del usuario '%s' cambiado a '%s'", usuario, grupo))
}

// leerBytesGroups función auxiliar para leer bytes del archivo (renombrada para evitar conflictos)
func leerBytesGroups(file *os.File, size int) []byte {
	bytes := make([]byte, size)
	file.Read(bytes)
	return bytes
}
