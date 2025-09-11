package Comandos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"

	"godisk-backend/Structs"
	"godisk-backend/Utils"
)

const (
	MAX_DIRECT_BLOCKS = 12
	BLOCK_SIZE        = 64
	FILE_TYPE         = 1
	DIR_TYPE          = 0
	DEFAULT_PERM      = 664
)

func ValidarDatosMKFILE(tokens []string) string {
	fmt.Printf("🔧 DEBUG: ValidarDatosMKFILE tokens=%v\n", tokens)

	if len(tokens) < 1 {
		return Utils.Error("MKFILE", "Se requieren parámetros. Uso: -path=/ruta [-r] [-size=N] [-cont=/ruta/local]")
	}

	var path string
	crearPadres := false
	var size int64 = 0
	var cont string

	// Parsear tokens
	for i := 0; i < len(tokens); i++ {
		tok := strings.TrimSpace(tokens[i])
		if tok == "" {
			continue
		}
		lower := strings.ToLower(tok)

		if lower == "-r" || lower == "r" {
			crearPadres = true
			continue
		}
		if strings.HasPrefix(lower, "size=") {
			parts := strings.SplitN(tok, "=", 2)
			if len(parts) == 2 {
				fmt.Sscanf(parts[1], "%d", &size)
			}
			continue
		}
		if strings.Contains(lower, "cont=") || strings.Contains(lower, "content=") {
			parts := strings.SplitN(tok, "=", 2)
			if len(parts) == 2 {
				cont = strings.Trim(parts[1], "\"")
			}
			continue
		}
		if strings.Contains(lower, "path=") {
			parts := strings.SplitN(tok, "=", 2)
			if len(parts) == 2 {
				path = strings.Trim(parts[1], "\"")
			}
			continue
		}
	}

	if path == "" {
		return Utils.Error("MKFILE", "El parámetro -path es obligatorio")
	}
	if size < 0 {
		return Utils.Error("MKFILE", "El parámetro -size no puede ser negativo")
	}
	if !EstaLogueado() {
		return Utils.Error("MKFILE", "Debe iniciar sesión para ejecutar este comando")
	}

	return mkfile(path, crearPadres, size, cont)
}

func mkfile(path string, crearPadres bool, size int64, cont string) string {
	fmt.Printf("🔧 DEBUG: MKFILE path='%s' -r=%t size=%d cont='%s'\n", path, crearPadres, size, cont)

	// Obtener partición montada
	sesion := ObtenerSesionActiva()
	var pathDisco string
	particion := GetMount("MKFILE", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("MKFILE", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	// Abrir archivo del disco
	file, err := os.OpenFile(pathDisco, os.O_RDWR, 0644)
	if err != nil {
		return Utils.Error("MKFILE", "No se pudo abrir el disco: "+err.Error())
	}
	defer file.Close()

	// Leer superbloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	if err := binary.Read(file, binary.LittleEndian, &super); err != nil {
		return Utils.Error("MKFILE", "Error al leer superbloque: "+err.Error())
	}

	// Calcular tamaos
	tamInodo := int64(unsafe.Sizeof(Structs.Inodos{}))
	tamBloqueCarp := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))

	// Preparar y validar ruta
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		return Utils.Error("MKFILE", "La ruta debe ser absoluta (comenzar con /)")
	}

	// Separar componentes de la ruta
	components := strings.Split(path, "/")
	var cleanComponents []string
	for _, comp := range components {
		if comp != "" {
			cleanComponents = append(cleanComponents, comp)
		}
	}
	if len(cleanComponents) == 0 {
		return Utils.Error("MKFILE", "Ruta inválida")
	}

	fileName := cleanComponents[len(cleanComponents)-1]
	parentComponents := cleanComponents[:len(cleanComponents)-1]

	fmt.Printf("🔧 DEBUG: Ruta procesada - Directorio: %v, Archivo: %s\n",
		parentComponents, fileName)

	// Preparar contenido
	var contentBytes []byte
	if cont != "" {
		contentBytes, err = os.ReadFile(cont)
		if err != nil {
			return Utils.Error("MKFILE", "Error leyendo archivo de contenido: "+err.Error())
		}
	}
	if size > 0 {
		if int64(len(contentBytes)) > size {
			contentBytes = contentBytes[:size]
		} else {
			for int64(len(contentBytes)) < size {
				contentBytes = append(contentBytes, '0')
			}
		}
	}

	// Verificar espacio disponible
	requiredBlocks := (int64(len(contentBytes)) + BLOCK_SIZE - 1) / BLOCK_SIZE
	if requiredBlocks > MAX_DIRECT_BLOCKS {
		return Utils.Error("MKFILE", "Archivo demasiado grande para bloques directos")
	}
	if super.S_free_blocks_count < requiredBlocks || super.S_free_inodes_count < 1 {
		return Utils.Error("MKFILE", "No hay espacio suficiente")
	}

	// Comenzar desde el inodo raíz
	var currentInode Structs.Inodos
	currentInodePos := super.S_inode_start

	// Leer inodo raíz
	file.Seek(currentInodePos, 0)
	if err := binary.Read(file, binary.LittleEndian, &currentInode); err != nil {
		return Utils.Error("MKFILE", "Error leyendo inodo raíz")
	}

	// Navegar la ruta del directorio padre
	for _, component := range parentComponents {
		found := false

		// Buscar en bloques directos del directorio actual
		for i := 0; i < MAX_DIRECT_BLOCKS && !found; i++ {
			if currentInode.I_block[i] == -1 {
				continue
			}

			var dirBlock Structs.BloquesCarpetas
			blockPos := super.S_block_start + (currentInode.I_block[i] * tamBloqueCarp)

			file.Seek(blockPos, 0)
			if err := binary.Read(file, binary.LittleEndian, &dirBlock); err != nil {
				continue
			}

			// Buscar el componente en las entradas
			for _, entry := range dirBlock.B_content {
				name := strings.Trim(string(entry.B_name[:]), "\x00")
				if name == component {
					// Verificar que sea un directorio
					var nextInode Structs.Inodos
					nextInodePos := super.S_inode_start + (entry.B_inodo * tamInodo)

					file.Seek(nextInodePos, 0)
					if err := binary.Read(file, binary.LittleEndian, &nextInode); err != nil {
						return Utils.Error("MKFILE", "Error leyendo inodo del directorio")
					}

					if nextInode.I_type != DIR_TYPE {
						return Utils.Error("MKFILE", fmt.Sprintf("'%s' no es un directorio", component))
					}

					currentInode = nextInode
					currentInodePos = nextInodePos
					found = true
					break
				}
			}
		}

		if !found {
			if !crearPadres {
				return Utils.Error("MKFILE", fmt.Sprintf("No existe el directorio '%s'", component))
			}
			// Crear directorio padre si está permitido
			newDirInode, newDirPos, err := crearDirectorio(&super, file, component, sesion.Uid, sesion.Gid)
			if err != nil {
				return Utils.Error("MKFILE", fmt.Sprintf("Error creando directorio '%s': %v", component, err))
			}
			currentInode = newDirInode
			currentInodePos = newDirPos
		}
	}

	// Verificar si el archivo ya existe
	for i := 0; i < MAX_DIRECT_BLOCKS; i++ {
		if currentInode.I_block[i] == -1 {
			continue
		}

		var dirBlock Structs.BloquesCarpetas
		blockPos := super.S_block_start + (currentInode.I_block[i] * tamBloqueCarp)

		file.Seek(blockPos, 0)
		if err := binary.Read(file, binary.LittleEndian, &dirBlock); err != nil {
			continue
		}

		for _, entry := range dirBlock.B_content {
			name := strings.Trim(string(entry.B_name[:]), "\x00")
			if name == fileName {
				return Utils.Error("MKFILE", fmt.Sprintf("El archivo '%s' ya existe", fileName))
			}
		}
	}

	// Crear nuevo inodo para el archivo
	newInode := Structs.NewInodos()
	for i := range newInode.I_block {
		newInode.I_block[i] = -1
	}

	newInode.I_uid = int64(sesion.Uid)
	newInode.I_gid = int64(sesion.Gid)
	newInode.I_size = int64(len(contentBytes))
	newInode.I_type = FILE_TYPE
	newInode.I_perm = DEFAULT_PERM

	timeStr := time.Now().Format("2006-01-02 15:04:05")
	copy(newInode.I_atime[:], timeStr)
	copy(newInode.I_ctime[:], timeStr)
	copy(newInode.I_mtime[:], timeStr)

	// Escribir contenido en bloques
	for i := 0; i < len(contentBytes); i += BLOCK_SIZE {
		blockIndex := i / BLOCK_SIZE
		if blockIndex >= MAX_DIRECT_BLOCKS {
			break
		}

		super.S_first_blo++
		newInode.I_block[blockIndex] = super.S_first_blo

		var fileBlock Structs.BloquesArchivos
		end := i + BLOCK_SIZE
		if end > len(contentBytes) {
			end = len(contentBytes)
		}
		copy(fileBlock.B_content[:], contentBytes[i:end])

		blockPos := fileBlockOffset(super, newInode.I_block[blockIndex])
		file.Seek(blockPos, 0)
		// Escribir el BloquesArchivos dentro de una ranura del tamao de BloquesCarpetas
		slotSize := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
		slotBuf := make([]byte, slotSize)
		var tempBuf bytes.Buffer
		if err := binary.Write(&tempBuf, binary.LittleEndian, fileBlock); err != nil {
			return Utils.Error("MKFILE", "Error preparando bloque de archivo")
		}
		copy(slotBuf, tempBuf.Bytes())
		if _, err := file.Write(slotBuf); err != nil {
			return Utils.Error("MKFILE", "Error escribiendo bloque de archivo")
		}

		file.Seek(super.S_bm_block_start+newInode.I_block[blockIndex], 0)
		file.Write([]byte{'1'})
		super.S_free_blocks_count--
	}

	// Asignar y escribir nuevo inodo
	super.S_firts_ino++
	super.S_free_inodes_count--
	newInodePos := super.S_inode_start + (super.S_firts_ino * tamInodo)

	file.Seek(newInodePos, 0)
	if err := binary.Write(file, binary.LittleEndian, &newInode); err != nil {
		return Utils.Error("MKFILE", "Error escribiendo nuevo inodo")
	}

	// Marcar inodo como usado
	file.Seek(super.S_bm_inode_start+super.S_firts_ino, 0)
	file.Write([]byte{'1'})

	// Agregar entrada en directorio padre
	found := false
	for i := 0; i < MAX_DIRECT_BLOCKS && !found; i++ {
		if currentInode.I_block[i] == -1 {
			// Crear nuevo bloque de directorio si es necesario
			super.S_first_blo++
			currentInode.I_block[i] = super.S_first_blo

			var newDirBlock Structs.BloquesCarpetas
			for j := range newDirBlock.B_content {
				newDirBlock.B_content[j].B_inodo = -1
			}

			blockPos := super.S_block_start + (currentInode.I_block[i] * tamBloqueCarp)
			file.Seek(blockPos, 0)
			if err := binary.Write(file, binary.LittleEndian, &newDirBlock); err != nil {
				return Utils.Error("MKFILE", "Error creando bloque directorio")
			}

			file.Seek(super.S_bm_block_start+currentInode.I_block[i], 0)
			file.Write([]byte{'1'})
			super.S_free_blocks_count--
		}

		var dirBlock Structs.BloquesCarpetas
		blockPos := super.S_block_start + (currentInode.I_block[i] * tamBloqueCarp)
		file.Seek(blockPos, 0)
		if err := binary.Read(file, binary.LittleEndian, &dirBlock); err != nil {
			continue
		}

		// Buscar espacio libre en el bloque
		for j := range dirBlock.B_content {
			if dirBlock.B_content[j].B_inodo == -1 {
				copy(dirBlock.B_content[j].B_name[:], fileName)
				dirBlock.B_content[j].B_inodo = super.S_firts_ino

				file.Seek(blockPos, 0)
				if err := binary.Write(file, binary.LittleEndian, &dirBlock); err != nil {
					return Utils.Error("MKFILE", "Error actualizando bloque directorio")
				}
				found = true
				break
			}
		}
	}

	if !found {
		return Utils.Error("MKFILE", "No hay espacio en el directorio para crear el archivo")
	}

	// Actualizar inodo padre
	file.Seek(currentInodePos, 0)
	if err := binary.Write(file, binary.LittleEndian, &currentInode); err != nil {
		return Utils.Error("MKFILE", "Error actualizando inodo padre")
	}

	// Actualizar superbloque
	file.Seek(particion.Part_start, 0)
	if err := binary.Write(file, binary.LittleEndian, &super); err != nil {
		return Utils.Error("MKFILE", "Error actualizando superbloque")
	}

	return Utils.Mensaje("MKFILE", fmt.Sprintf("Archivo '%s' creado exitosamente", path))
}

// Función auxiliar para crear un directorio
func crearDirectorio(super *Structs.SuperBloque, file *os.File, name string, uid, gid int) (Structs.Inodos, int64, error) {
	tamInodo := int64(unsafe.Sizeof(Structs.Inodos{}))
	tamBloqueCarp := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))

	// Crear nuevo inodo para el directorio
	newInode := Structs.NewInodos()
	for i := range newInode.I_block {
		newInode.I_block[i] = -1
	}

	newInode.I_uid = int64(uid)
	newInode.I_gid = int64(gid)
	newInode.I_size = 0
	newInode.I_type = DIR_TYPE
	newInode.I_perm = DEFAULT_PERM

	timeStr := time.Now().Format("2006-01-02 15:04:05")
	copy(newInode.I_atime[:], timeStr)
	copy(newInode.I_ctime[:], timeStr)
	copy(newInode.I_mtime[:], timeStr)

	// Crear primer bloque de directorio
	super.S_first_blo++
	newInode.I_block[0] = super.S_first_blo

	var dirBlock Structs.BloquesCarpetas
	for i := range dirBlock.B_content {
		dirBlock.B_content[i].B_inodo = -1
	}

	// Escribir bloque de directorio
	blockPos := super.S_block_start + (newInode.I_block[0] * tamBloqueCarp)
	file.Seek(blockPos, 0)
	if err := binary.Write(file, binary.LittleEndian, &dirBlock); err != nil {
		return Structs.Inodos{}, 0, err
	}

	// Marcar bloque como usado
	file.Seek(super.S_bm_block_start+newInode.I_block[0], 0)
	file.Write([]byte{'1'})
	super.S_free_blocks_count--

	// Asignar y escribir nuevo inodo
	super.S_firts_ino++
	super.S_free_inodes_count--
	newInodePos := super.S_inode_start + (super.S_firts_ino * tamInodo)

	file.Seek(newInodePos, 0)
	if err := binary.Write(file, binary.LittleEndian, &newInode); err != nil {
		return Structs.Inodos{}, 0, err
	}

	// Marcar inodo como usado
	file.Seek(super.S_bm_inode_start+super.S_firts_ino, 0)
	file.Write([]byte{'1'})

	return newInode, newInodePos, nil
}
