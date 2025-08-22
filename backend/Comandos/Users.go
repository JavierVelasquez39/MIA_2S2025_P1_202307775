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

// ValidarDatosMKUSR valida los parámetros del comando MKUSR
func ValidarDatosMKUSR(tokens []string) string {
	if len(tokens) < 3 {
		return Utils.Error("MKUSR", "Se requieren los parámetros: -user, -pass, -grp")
	}

	var usuario, password, grupo string

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
		case "pass":
			password = value
		case "grp":
			grupo = value
		default:
			return Utils.Error("MKUSR", "Parámetro no reconocido: "+param)
		}
	}

	// Validaciones
	if usuario == "" {
		return Utils.Error("MKUSR", "El parámetro -user es obligatorio")
	}
	if password == "" {
		return Utils.Error("MKUSR", "El parámetro -pass es obligatorio")
	}
	if grupo == "" {
		return Utils.Error("MKUSR", "El parámetro -grp es obligatorio")
	}

	// Validar longitud máxima (según especificación: máximo 10 caracteres)
	if len(usuario) > 10 {
		return Utils.Error("MKUSR", "El nombre de usuario no puede exceder 10 caracteres")
	}
	if len(password) > 10 {
		return Utils.Error("MKUSR", "La contraseña no puede exceder 10 caracteres")
	}
	if len(grupo) > 10 {
		return Utils.Error("MKUSR", "El nombre del grupo no puede exceder 10 caracteres")
	}

	// Verificar que hay una sesión activa
	if !EstaLogueado() {
		return Utils.Error("MKUSR", "Debe iniciar sesión para ejecutar este comando")
	}

	// Verificar que el usuario es root
	if !EsUsuarioRoot() {
		return Utils.Error("MKUSR", "Solo el usuario \"root\" puede acceder a estos comandos")
	}

	return mkusr(usuario, password, grupo)
}

// ValidarDatosRMUSR valida los parámetros del comando RMUSR
func ValidarDatosRMUSR(tokens []string) string {
	if len(tokens) < 1 {
		return Utils.Error("RMUSR", "Se requiere el parámetro -user")
	}

	var usuario string

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
		default:
			return Utils.Error("RMUSR", "Parámetro no reconocido: "+param)
		}
	}

	// Validaciones
	if usuario == "" {
		return Utils.Error("RMUSR", "El parámetro -user es obligatorio")
	}

	// Verificar que hay una sesión activa
	if !EstaLogueado() {
		return Utils.Error("RMUSR", "Debe iniciar sesión para ejecutar este comando")
	}

	// Verificar que el usuario es root
	if !EsUsuarioRoot() {
		return Utils.Error("RMUSR", "Solo el usuario \"root\" puede acceder a estos comandos")
	}

	return rmusr(usuario)
}

// mkusr crea un nuevo usuario - USANDO EXACTAMENTE LA MISMA ARQUITECTURA QUE MKGRP
func mkusr(usuario, password, grupo string) string {
	fmt.Printf("🔧 DEBUG: Creando usuario '%s' con grupo '%s'\n", usuario, grupo)

	sesion := ObtenerSesionActiva()

	// Obtener la partición montada de la sesión activa
	var pathDisco string
	particion := GetMount("MKUSR", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("MKUSR", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	fmt.Printf("🔧 DEBUG: Partición encontrada en: %s\n", pathDisco)

	// Abrir archivo para lectura primero
	file, err := os.Open(strings.ReplaceAll(pathDisco, "\"", ""))
	if err != nil {
		return Utils.Error("MKUSR", "No se encontró el disco: "+err.Error())
	}
	defer file.Close()

	// Leer SuperBloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	data := leerBytesUsers(file, int(unsafe.Sizeof(Structs.SuperBloque{})))
	buffer := bytes.NewBuffer(data)
	err_ := binary.Read(buffer, binary.BigEndian, &super)
	if err_ != nil {
		return Utils.Error("MKUSR", "Error al leer superbloque: "+err_.Error())
	}

	fmt.Printf("🔧 DEBUG: SuperBloque leído correctamente\n")

	// Leer inodo del archivo users.txt (inodo 1)
	inodo := Structs.NewInodos()
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	data = leerBytesUsers(file, int(unsafe.Sizeof(Structs.Inodos{})))
	buffer = bytes.NewBuffer(data)
	err_ = binary.Read(buffer, binary.BigEndian, &inodo)
	if err_ != nil {
		return Utils.Error("MKUSR", "Error al leer inodo users.txt: "+err_.Error())
	}

	fmt.Printf("🔧 DEBUG: Inodo users.txt leído - Tamaño: %d\n", inodo.I_size)

	// ✅ USAR EXACTAMENTE LA MISMA FUNCIÓN QUE MKGRP (de Login.go)
	contenidoActual := leerContenidoUsersArchivo(file, super, inodo)
	if contenidoActual == "" {
		return Utils.Error("MKUSR", "No se pudo leer el archivo users.txt")
	}

	fmt.Printf("🔧 DEBUG: Contenido actual users.txt:\n%s\n", contenidoActual)

	// Verificar que el grupo existe
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	grupoExiste := false

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if len(linea) < 3 {
			continue
		}

		if (linea[2] == 'G' || linea[2] == 'g') && linea[0] != '0' {
			campos := strings.Split(linea, ",")
			if len(campos) >= 3 && campos[2] == grupo {
				grupoExiste = true
				fmt.Printf("🔧 DEBUG: Grupo '%s' encontrado en línea: %s\n", grupo, linea)
				break
			}
		}
	}

	if !grupoExiste {
		return Utils.Error("MKUSR", "No se encontró el grupo \""+grupo+"\".")
	}

	// Verificar si el usuario ya existe y contar usuarios
	contadorUsuarios := 0
	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if len(linea) < 3 {
			continue
		}

		if linea[2] == 'U' || linea[2] == 'u' {
			contadorUsuarios++
			campos := strings.Split(linea, ",")
			if len(campos) >= 4 {
				nombreUsuario := campos[3]
				if nombreUsuario == usuario {
					if linea[0] != '0' { // Si no está eliminado
						return Utils.Error("MKUSR", "EL nombre "+usuario+", ya está en uso.")
					}
				}
			}
		}
	}

	// Crear nueva línea del usuario - USAR strconv.Itoa como en el código de referencia
	nuevoID := contadorUsuarios + 1
	nuevaLinea := strconv.Itoa(nuevoID) + ",U," + grupo + "," + usuario + "," + password + "\n"

	// ✅ PRESERVAR CONTENIDO ANTERIOR - Agregar la nueva línea al contenido existente
	nuevoContenido := contenidoActual + nuevaLinea

	fmt.Printf("🔧 DEBUG: Nuevo contenido users.txt:\n%s\n", nuevoContenido)

	// ✅ USAR EXACTAMENTE LA MISMA FUNCIÓN QUE MKGRP (de Groups.go)
	if err := escribirContenidoUsersLocal(pathDisco, *particion, super, inodo, nuevoContenido); err != nil {
		return Utils.Error("MKUSR", "Error al escribir en users.txt: "+err.Error())
	}

	fmt.Printf("✅ MKUSR: Usuario '%s' creado correctamente\n", usuario)
	return Utils.Mensaje("MKUSR", "Usuario "+usuario+", creado correctamente!")
}

// rmusr elimina un usuario - USANDO EXACTAMENTE LA MISMA ARQUITECTURA QUE RMGRP
func rmusr(usuario string) string {
	fmt.Printf("🔧 DEBUG: Eliminando usuario '%s'\n", usuario)

	sesion := ObtenerSesionActiva()

	// Obtener la partición montada de la sesión activa
	var pathDisco string
	particion := GetMount("RMUSR", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("RMUSR", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	fmt.Printf("🔧 DEBUG: Partición encontrada en: %s\n", pathDisco)

	// Abrir archivo para lectura primero
	file, err := os.Open(strings.ReplaceAll(pathDisco, "\"", ""))
	if err != nil {
		return Utils.Error("RMUSR", "No se encontró el disco: "+err.Error())
	}
	defer file.Close()

	// Leer SuperBloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	data := leerBytesUsers(file, int(unsafe.Sizeof(Structs.SuperBloque{})))
	buffer := bytes.NewBuffer(data)
	err_ := binary.Read(buffer, binary.BigEndian, &super)
	if err_ != nil {
		return Utils.Error("RMUSR", "Error al leer superbloque: "+err_.Error())
	}

	// Leer inodo del archivo users.txt (inodo 1)
	inodo := Structs.NewInodos()
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	data = leerBytesUsers(file, int(unsafe.Sizeof(Structs.Inodos{})))
	buffer = bytes.NewBuffer(data)
	err_ = binary.Read(buffer, binary.BigEndian, &inodo)
	if err_ != nil {
		return Utils.Error("RMUSR", "Error al leer inodo users.txt: "+err_.Error())
	}

	// ✅ USAR EXACTAMENTE LA MISMA FUNCIÓN QUE RMGRP (de Login.go)
	contenidoActual := leerContenidoUsersArchivo(file, super, inodo)
	if contenidoActual == "" {
		return Utils.Error("RMUSR", "No se pudo leer el archivo users.txt")
	}

	fmt.Printf("🔧 DEBUG: Contenido actual users.txt:\n%s\n", contenidoActual)

	// Procesar líneas y marcar usuario como eliminado
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	var nuevasLineas []string
	usuarioEncontrado := false

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		if len(linea) >= 3 && (linea[2] == 'U' || linea[2] == 'u') && linea[0] != '0' {
			campos := strings.Split(linea, ",")
			if len(campos) >= 4 && campos[3] == usuario {
				// Marcar como eliminado (cambiar ID a 0) - USAR strconv.Itoa
				nuevaLinea := strconv.Itoa(0) + ",U," + campos[2] + "," + campos[3] + "," + campos[4]
				nuevasLineas = append(nuevasLineas, nuevaLinea)
				usuarioEncontrado = true
				fmt.Printf("🔧 DEBUG: Usuario '%s' marcado como eliminado\n", usuario)
				continue
			}
		}

		nuevasLineas = append(nuevasLineas, linea)
	}

	if !usuarioEncontrado {
		return Utils.Error("RMUSR", "No se encontró el usuario  \""+usuario+"\".")
	}

	// ✅ PRESERVAR CONTENIDO - Reconstruir contenido manteniendo todas las líneas
	nuevoContenido := strings.Join(nuevasLineas, "\n") + "\n"

	fmt.Printf("🔧 DEBUG: Nuevo contenido users.txt:\n%s\n", nuevoContenido)

	// ✅ USAR EXACTAMENTE LA MISMA FUNCIÓN QUE RMGRP (de Groups.go)
	if err := escribirContenidoUsersLocal(pathDisco, *particion, super, inodo, nuevoContenido); err != nil {
		return Utils.Error("RMUSR", "Error al escribir en users.txt: "+err.Error())
	}

	fmt.Printf("✅ RMUSR: Usuario '%s' eliminado correctamente\n", usuario)
	return Utils.Mensaje("RMUSR", "Usuario "+usuario+", eliminado correctamente!")
}

// leerBytesUsers función auxiliar para leer bytes del archivo
func leerBytesUsers(file *os.File, size int) []byte {
	bytes := make([]byte, size)
	file.Read(bytes)
	return bytes
}

// escribirContenidoUsersLocal función auxiliar para escribir contenido en el archivo users.txt
func escribirContenidoUsersLocal(pathDisco string, particion Structs.Particion, super Structs.SuperBloque, inodo Structs.Inodos, nuevoContenido string) error {
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

	// Tamaños de bloque
	tamBA := int64(unsafe.Sizeof(Structs.BloquesArchivos{}))

	// Asignar bloques únicos para cada fragmento
	for i := 0; i < 16; i++ {
		if i < len(bloques) {
			// Si el bloque no está asignado, asignar uno nuevo
			if inodo.I_block[i] == -1 {
				super.S_first_blo++
				inodo.I_block[i] = super.S_first_blo

				// Actualizar superbloque en disco
				file.Seek(particion.Part_start, 0)
				var bufferSuper bytes.Buffer
				if err := binary.Write(&bufferSuper, binary.BigEndian, super); err != nil {
					return err
				}
				if _, err := file.Write(bufferSuper.Bytes()); err != nil {
					return err
				}

				// Marcar el bitmap de bloques como ocupado (1)
				// El desplazamiento S_bm_block_start ya es absoluto en las operaciones del proyecto
				bitmapPos := super.S_bm_block_start + int64(super.S_first_blo-1)
				file.Seek(bitmapPos, 0)
				if _, err := file.Write([]byte{1}); err != nil {
					return err
				}
			}
		} else {
			// Si no hay más contenido, marcar el bloque como -1
			inodo.I_block[i] = -1
		}
	}

	// Escribir cada bloque y verificar lectura inmediata
	for i, contenidoBloque := range bloques {
		var bloqueArchivo Structs.BloquesArchivos
		// asegurar que el bloque esté limpio y copiar el contenido
		for k := range bloqueArchivo.B_content {
			bloqueArchivo.B_content[k] = 0
		}
		copy(bloqueArchivo.B_content[:], contenidoBloque)

		// Calcular posición del bloque usando el mismo offset que el lector (desplazamiento de BloquesCarpetas)
		mitadBA := super.S_block_start + int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
		posicionBloque := mitadBA + (int64(inodo.I_block[i]-1) * tamBA)
		file.Seek(posicionBloque, 0)

		var bufferBloque bytes.Buffer
		if err := binary.Write(&bufferBloque, binary.BigEndian, bloqueArchivo); err != nil {
			return err
		}
		if _, err := file.Write(bufferBloque.Bytes()); err != nil {
			return err
		}

		// Verificación inmediata: leer de vuelta lo escrito
		var verificacion Structs.BloquesArchivos
		file.Seek(posicionBloque, 0)
		data := make([]byte, tamBA)
		file.Read(data)
		buf := bytes.NewBuffer(data)
		if err := binary.Read(buf, binary.BigEndian, &verificacion); err != nil {
			return err
		}
		// Debug: imprimir los primeros bytes escritos (opcional)
		fmt.Printf("🔧 DEBUG: Bloque %d escrito en offset %d, muestra: %q\n", inodo.I_block[i], posicionBloque, string(verificacion.B_content[:min(16, len(verificacion.B_content))]))
	}

	// Actualizar tamaño del inodo
	inodo.I_size = int64(len(nuevoContenido))

	// Escribir inodo actualizado
	file.Seek(super.S_inode_start+int64(unsafe.Sizeof(Structs.Inodos{})), 0)
	var bufferInodo bytes.Buffer
	if err := binary.Write(&bufferInodo, binary.BigEndian, inodo); err != nil {
		return err
	}
	if _, err := file.Write(bufferInodo.Bytes()); err != nil {
		return err
	}

	file.Sync()
	return nil
}

// helper min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
