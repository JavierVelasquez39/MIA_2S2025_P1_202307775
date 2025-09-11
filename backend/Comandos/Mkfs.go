package Comandos

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
	"unsafe"

	"godisk-backend/Structs"
	"godisk-backend/Utils"
)

// ValidarDatosMKFS valida los parámetros del comando MKFS
func ValidarDatosMKFS(tokens []string) string {
	if len(tokens) < 1 {
		return Utils.Error("MKFS", "Se requiere al menos el parámetro id")
	}

	id := ""
	tipo := "full" // Por defecto es full
	fs := "2fs"    // Por defecto es ext2

	// Parsear tokens
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		tk := strings.Split(token, "=")
		if len(tk) != 2 {
			continue
		}

		param := strings.ToLower(tk[0])
		value := strings.ToLower(tk[1])

		switch param {
		case "id":
			id = tk[1] // Mantener el ID original (sin toLowerCase)
		case "type":
			if value == "fast" || value == "full" {
				tipo = value
			} else {
				return Utils.Error("MKFS", "El parámetro type debe ser 'fast' o 'full'")
			}
		case "fs":
			if value == "2fs" || value == "3fs" {
				fs = value
			} else {
				return Utils.Error("MKFS", "El parámetro fs debe ser '2fs' o '3fs'")
			}
		default:
			return Utils.Error("MKFS", "Parámetro no reconocido: "+param)
		}
	}

	// Validaciones
	if id == "" {
		return Utils.Error("MKFS", "El parámetro id es obligatorio")
	}

	return mkfs(id, tipo, fs)
}

// mkfs formatea una partición con el sistema de archivos especificado
func mkfs(id, tipo, fs string) string {
	fmt.Printf("🔧 DEBUG: Formateando partición - ID: %s, Tipo: %s, FS: %s\n", id, tipo, fs)

	// Obtener la partición montada
	path := ""
	particion := GetMount("MKFS", id, &path)
	if particion == nil {
		return Utils.Error("MKFS", "No se encontró una partición montada con el ID: "+id)
	}

	if fs == "2fs" {
		return formatearEXT2(*particion, path, tipo)
	} else if fs == "3fs" {
		return Utils.Error("MKFS", "EXT3 no está implementado en esta versión")
	} else {
		return Utils.Error("MKFS", "Sistema de archivos no válido")
	}
}

// formatearEXT2 formatea una partición con EXT2
func formatearEXT2(particion Structs.Particion, path, tipo string) string {
	// Calcular el número de inodos y bloques
	// n = (partition.size - sizeof(superblock)) / (4 + sizeof(inode) + 3*sizeof(block))
	superBloqueSize := int64(unsafe.Sizeof(Structs.SuperBloque{}))
	inodoSize := int64(unsafe.Sizeof(Structs.Inodos{}))
	bloqueSize := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))

	n := math.Floor(float64(particion.Part_size-superBloqueSize) / float64(4+inodoSize+3*bloqueSize))
	numInodos := int64(n)
	numBloques := int64(3 * n)

	fmt.Printf("🔧 DEBUG: Calculando estructuras - Inodos: %d, Bloques: %d\n", numInodos, numBloques)

	// Crear SuperBloque
	spr := Structs.NewSuperBloque()
	spr.S_filesystem_type = 2 // EXT2
	spr.S_inodes_count = numInodos
	spr.S_blocks_count = numBloques
	spr.S_free_inodes_count = numInodos
	spr.S_free_blocks_count = numBloques

	// Configurar fechas
	fecha := time.Now().Format("2006-01-02 15:04:05")
	copy(spr.S_mtime[:], fecha)
	fechaAntigua := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05")
	copy(spr.S_umtime[:], fechaAntigua)
	spr.S_mnt_count = 1

	// Calcular posiciones de las estructuras
	spr.S_bm_inode_start = particion.Part_start + superBloqueSize
	spr.S_bm_block_start = spr.S_bm_inode_start + numInodos
	spr.S_inode_start = spr.S_bm_block_start + numBloques
	spr.S_block_start = spr.S_inode_start + (numInodos * inodoSize)
	spr.S_firts_ino = 0
	spr.S_first_blo = 0

	fmt.Printf("🔧 DEBUG: Posiciones calculadas - BMI: %d, BMB: %d, Inodos: %d, Bloques: %d\n",
		spr.S_bm_inode_start, spr.S_bm_block_start, spr.S_inode_start, spr.S_block_start)

	// Abrir archivo para escritura
	file, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return Utils.Error("MKFS", "No se pudo abrir el disco: "+err.Error())
	}
	defer file.Close()

	// Escribir SuperBloque
	file.Seek(particion.Part_start, 0)
	if err := binary.Write(file, binary.LittleEndian, spr); err != nil {
		return Utils.Error("MKFS", "Error al escribir el superbloque")
	}

	// Inicializar bitmap de inodos (todos en '0')
	file.Seek(spr.S_bm_inode_start, 0)
	for i := 0; i < int(numInodos); i++ {
		file.Write([]byte{'0'})
	}

	// Inicializar bitmap de bloques (todos en '0')
	file.Seek(spr.S_bm_block_start, 0)
	for i := 0; i < int(numBloques); i++ {
		file.Write([]byte{'0'})
	}

	// Inicializar inodos vacíos
	inodoVacio := Structs.NewInodos()
	file.Seek(spr.S_inode_start, 0)
	for i := 0; i < int(numInodos); i++ {
		if err := binary.Write(file, binary.LittleEndian, inodoVacio); err != nil {
			return Utils.Error("MKFS", "Error al escribir inodos")
		}
	}

	// CORRECCIÓN: Inicializar TODOS los bloques vacíos (numBloques en lugar de numInodos)
	file.Seek(spr.S_block_start, 0)
	for i := 0; i < int(numBloques); i++ {
		// Los primeros bloques podrían ser carpetas o archivos
		if i < int(numInodos) {
			bloqueVacio := Structs.NewBloquesCarpetas()
			if err := binary.Write(file, binary.LittleEndian, bloqueVacio); err != nil {
				return Utils.Error("MKFS", "Error al escribir bloques de carpetas/archivos")
			}
		} else {
			// El resto serían bloques de apuntadores
			bloqueApuntadoresVacio := Structs.NewBloquesApuntadores()
			if err := binary.Write(file, binary.LittleEndian, bloqueApuntadoresVacio); err != nil {
				return Utils.Error("MKFS", "Error al escribir bloques de apuntadores")
			}
		}
	}

	if tipo == "full" {
		// Crear estructura inicial del sistema de archivos
		if err := crearEstructuraInicial(file, spr, particion); err != nil {
			return Utils.Error("MKFS", "Error al crear estructura inicial: "+err.Error())
		}

		// AGREGAR VERIFICACIÓN COMPLETA:
		verificarEstructuras(file, spr, particion, tipo)
	}

	// Obtener nombre de la partición
	nombreParticion := ""
	for _, b := range particion.Part_name {
		if b != 0 {
			nombreParticion += string(b)
		} else {
			break
		}
	}

	return Utils.Mensaje("MKFS", fmt.Sprintf("Partición '%s' formateada correctamente con EXT2", nombreParticion))
}

// verificarEstructuras muestra las posiciones reales y verifica el contenido
func verificarEstructuras(file *os.File, spr Structs.SuperBloque, particion Structs.Particion, tipo string) {
	fmt.Println("\n🔍 POSICIONES REALES DE LAS ESTRUCTURAS:")
	fmt.Println("═══════════════════════════════════════════")

	fmt.Printf("Partición inicia en: %d (0x%x)\n", particion.Part_start, particion.Part_start)
	fmt.Printf("SuperBloque en: %d (0x%x)\n", particion.Part_start, particion.Part_start)
	fmt.Printf("Bitmap inodos en: %d (0x%x)\n", spr.S_bm_inode_start, spr.S_bm_inode_start)
	fmt.Printf("Bitmap bloques en: %d (0x%x)\n", spr.S_bm_block_start, spr.S_bm_block_start)
	fmt.Printf("Tabla inodos en: %d (0x%x)\n", spr.S_inode_start, spr.S_inode_start)
	fmt.Printf("Área bloques en: %d (0x%x)\n", spr.S_block_start, spr.S_block_start)

	// Calcular posición del bloque 1 (users.txt)
	tamañoBloque := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	posicionBloque1 := spr.S_block_start + tamañoBloque
	fmt.Printf("Bloque users.txt en: %d (0x%x)\n", posicionBloque1, posicionBloque1)

	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("🔧 COMANDOS PARA VERIFICAR:")
	fmt.Printf("xxd -s +%d -l 100 Disco2.mia  # SuperBloque\n", particion.Part_start)
	fmt.Printf("xxd -s +%d -l 20 Disco2.mia   # Bitmap inodos\n", spr.S_bm_inode_start)
	fmt.Printf("xxd -s +%d -l 20 Disco2.mia   # Bitmap bloques\n", spr.S_bm_block_start)
	fmt.Printf("xxd -s +%d -l 100 Disco2.mia  # Tabla inodos\n", spr.S_inode_start)
	fmt.Printf("xxd -s +%d -l 100 Disco2.mia  # Directorio raíz\n", spr.S_block_start)
	fmt.Printf("xxd -s +%d -l 50 Disco2.mia   # Archivo users.txt\n", posicionBloque1)

	if tipo == "full" {
		fmt.Println("\n🔍 VERIFICANDO CONTENIDO CREADO:")
		fmt.Println("═══════════════════════════════════════════")

		// 1. Verificar SuperBloque
		file.Seek(particion.Part_start, 0)
		var sprLeido Structs.SuperBloque
		if err := binary.Read(file, binary.LittleEndian, &sprLeido); err != nil {
			fmt.Printf("❌ Error leyendo superbloque: %v\n", err)
		} else {
			fmt.Printf("✅ SuperBloque leído correctamente:\n")
			fmt.Printf("   - Tipo FS: %d (debería ser 2)\n", sprLeido.S_filesystem_type)
			fmt.Printf("   - Inodos totales: %d\n", sprLeido.S_inodes_count)
			fmt.Printf("   - Bloques totales: %d\n", sprLeido.S_blocks_count)
			fmt.Printf("   - Inodos libres: %d\n", sprLeido.S_free_inodes_count)
			fmt.Printf("   - Bloques libres: %d\n", sprLeido.S_free_blocks_count)
		}

		// 2. Verificar bitmap de inodos
		file.Seek(spr.S_bm_inode_start, 0)
		bitmapInodos := make([]byte, 10) // Leer primeros 10 bytes
		file.Read(bitmapInodos)
		fmt.Printf("✅ Bitmap inodos (primeros 10): %s\n", string(bitmapInodos))

		// 3. Verificar bitmap de bloques
		file.Seek(spr.S_bm_block_start, 0)
		bitmapBloques := make([]byte, 10) // Leer primeros 10 bytes
		file.Read(bitmapBloques)
		fmt.Printf("✅ Bitmap bloques (primeros 10): %s\n", string(bitmapBloques))

		// 4. Verificar inodo raíz
		file.Seek(spr.S_inode_start, 0)
		var inodoRaiz Structs.Inodos
		if err := binary.Read(file, binary.LittleEndian, &inodoRaiz); err != nil {
			fmt.Printf("❌ Error leyendo inodo raíz: %v\n", err)
		} else {
			fmt.Printf("✅ Inodo raíz leído:\n")
			fmt.Printf("   - Tipo: %d (0=directorio)\n", inodoRaiz.I_type)
			fmt.Printf("   - Tamaño: %d bytes\n", inodoRaiz.I_size)
			fmt.Printf("   - Bloque[0]: %d\n", inodoRaiz.I_block[0])
		}

		// 5. Verificar inodo users.txt
		var inodoUsers Structs.Inodos
		if err := binary.Read(file, binary.LittleEndian, &inodoUsers); err != nil {
			fmt.Printf("❌ Error leyendo inodo users.txt: %v\n", err)
		} else {
			fmt.Printf("✅ Inodo users.txt leído:\n")
			fmt.Printf("   - Tipo: %d (1=archivo)\n", inodoUsers.I_type)
			fmt.Printf("   - Tamaño: %d bytes\n", inodoUsers.I_size)
			fmt.Printf("   - Bloque[0]: %d\n", inodoUsers.I_block[0])
		}

		// 6. Verificar contenido del bloque del directorio raíz
		file.Seek(spr.S_block_start, 0)
		var bloqueRaiz Structs.BloquesCarpetas
		if err := binary.Read(file, binary.LittleEndian, &bloqueRaiz); err != nil {
			fmt.Printf("❌ Error leyendo bloque directorio raíz: %v\n", err)
		} else {
			nombre2 := ""
			for _, b := range bloqueRaiz.B_content[2].B_name {
				if b != 0 {
					nombre2 += string(b)
				} else {
					break
				}
			}
			fmt.Printf("✅ Directorio raíz leído:\n")
			fmt.Printf("   - Entrada[2]: '%s' -> inodo %d\n", nombre2, bloqueRaiz.B_content[2].B_inodo)
		}

		// 7. Verificar contenido del archivo users.txt
		file.Seek(posicionBloque1, 0)
		var bloqueUsers Structs.BloquesArchivos
		if err := binary.Read(file, binary.LittleEndian, &bloqueUsers); err != nil {
			fmt.Printf("❌ Error leyendo bloque users.txt: %v\n", err)
		} else {
			contenido := ""
			for _, b := range bloqueUsers.B_content {
				if b != 0 {
					contenido += string(b)
				} else {
					break
				}
			}
			fmt.Printf("✅ Archivo users.txt leído:\n")
			fmt.Printf("   - Contenido: %q\n", contenido)
		}
	}

	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("🎯 VERIFICACIÓN COMPLETADA")
}

// crearEstructuraInicial crea la estructura inicial del sistema de archivos
func crearEstructuraInicial(file *os.File, spr Structs.SuperBloque, particion Structs.Particion) error {
	fecha := time.Now().Format("2006-01-02 15:04:05")

	// Actualizar contadores del superbloque
	// Reservamos inodos 0 y 1 y bloques 0 y 1
	spr.S_free_inodes_count -= 2 // Restamos 2 inodos (raíz + users.txt)
	spr.S_free_blocks_count -= 2 // Restamos 2 bloques (raíz + users.txt)

	// S_firts_ino y S_first_blo deben reflejar el último índice asignado (1)
	spr.S_firts_ino = 1
	spr.S_first_blo = 1

	// Reescribir superbloque actualizado
	file.Seek(particion.Part_start, 0)
	if err := binary.Write(file, binary.LittleEndian, spr); err != nil {
		return err
	}

	// Marcar inodos y bloques como ocupados en los bitmaps (usar '1')
	file.Seek(spr.S_bm_inode_start, 0)
	file.Write([]byte{'1'}) // Inodo 0 (directorio raíz)
	file.Write([]byte{'1'}) // Inodo 1 (archivo users.txt)

	file.Seek(spr.S_bm_block_start, 0)
	file.Write([]byte{'1'}) // Bloque 0 (directorio raíz)
	file.Write([]byte{'1'}) // Bloque 1 (archivo users.txt)

	// Crear contenido del archivo users.txt con la estructura correcta
	inodoUsersData := "1,G,root\n1,U,root,root,123\n"
	fmt.Printf("🔧 DEBUG: Creando users.txt con contenido: %q\n", inodoUsersData)

	// Crear inodo del directorio raíz
	inodoRaiz := Structs.NewInodos()
	// CORRECCIÓN: Cambiar el UID y GID a 1 (usuario root, grupo root)
	inodoRaiz.I_uid = 1 // Cambiado de 0 a 1 para usuario root
	inodoRaiz.I_gid = 1 // Cambiado de 0 a 1 para grupo root
	inodoRaiz.I_size = int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	copy(inodoRaiz.I_atime[:], fecha)
	copy(inodoRaiz.I_ctime[:], fecha)
	copy(inodoRaiz.I_mtime[:], fecha)
	inodoRaiz.I_type = 0 // Directorio
	inodoRaiz.I_perm = 664
	inodoRaiz.I_block[0] = 0 // Apunta al bloque 0

	// Crear inodo del archivo users.txt
	inodoUsers := Structs.NewInodos()
	// CORRECCIÓN: Cambiar el UID y GID a 1 (usuario root, grupo root)
	inodoUsers.I_uid = 1 // Cambiado de 0 a 1 para usuario root
	inodoUsers.I_gid = 1 // Cambiado de 0 a 1 para grupo root
	inodoUsers.I_size = int64(len(inodoUsersData))
	copy(inodoUsers.I_atime[:], fecha)
	copy(inodoUsers.I_ctime[:], fecha)
	copy(inodoUsers.I_mtime[:], fecha)
	inodoUsers.I_type = 1 // Archivo
	inodoUsers.I_perm = 664
	inodoUsers.I_block[0] = 1 // Apunta al bloque 1

	// Escribir inodos
	file.Seek(spr.S_inode_start, 0)
	if err := binary.Write(file, binary.LittleEndian, inodoRaiz); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, inodoUsers); err != nil {
		return err
	}

	// Crear bloque del directorio raíz
	bloqueRaiz := Structs.NewBloquesCarpetas()
	copy(bloqueRaiz.B_content[0].B_name[:], ".")
	bloqueRaiz.B_content[0].B_inodo = 0
	copy(bloqueRaiz.B_content[1].B_name[:], "..")
	bloqueRaiz.B_content[1].B_inodo = 0
	copy(bloqueRaiz.B_content[2].B_name[:], "users.txt")
	bloqueRaiz.B_content[2].B_inodo = 1

	// Crear bloque del archivo users.txt
	var bloqueUsers Structs.BloquesArchivos

	// Inicializar el buffer con ceros antes de copiar el contenido
	for i := range bloqueUsers.B_content {
		bloqueUsers.B_content[i] = 0
	}

	// Copiar el contenido asegurando que solo se copie hasta el tamaño del contenido
	copy(bloqueUsers.B_content[:], inodoUsersData)

	// Escribir bloque del directorio raíz (posición exacta del bloque 0)
	file.Seek(spr.S_block_start, 0)
	if err := binary.Write(file, binary.LittleEndian, bloqueRaiz); err != nil {
		return err
	}

	// Escribir bloque del archivo users.txt (posición exacta del bloque 1)
	tamañoBloque := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	posicionBloque1 := spr.S_block_start + tamañoBloque

	file.Seek(posicionBloque1, 0)
	if err := binary.Write(file, binary.LittleEndian, bloqueUsers); err != nil {
		return err
	}

	// Forzar escritura al disco
	file.Sync()

	fmt.Printf("✅ DEBUG: users.txt creado correctamente en posición %d\n", posicionBloque1)
	fmt.Printf("✅ DEBUG: Contenido: %q\n", inodoUsersData)

	return nil
}
