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

// ValidarDatosMKFILE valida parámetros y llama a mkfile
func ValidarDatosMKFILE(tokens []string) string {
	fmt.Printf("🔧 DEBUG: ValidarDatosMKFILE tokens=%v\n", tokens)

	if len(tokens) < 1 {
		return Utils.Error("MKFILE", "Se requieren parámetros. Uso: -path=/ruta [-r] [-size=N] [-cont=/ruta/local]")
	}

	var path string
	crearPadres := false
	var size int64 = 0
	var cont string

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
				var s int64
				fmt.Sscanf(parts[1], "%d", &s)
				size = s
			}
			continue
		}
		if strings.Contains(lower, "cont=") || strings.Contains(lower, "content=") {
			parts := strings.SplitN(tok, "=", 2)
			if len(parts) == 2 {
				cont = strings.ReplaceAll(parts[1], "\"", "")
			}
			continue
		}
		if strings.Contains(lower, "path=") {
			parts := strings.SplitN(tok, "=", 2)
			if len(parts) == 2 {
				path = strings.ReplaceAll(parts[1], "\"", "")
			}
			continue
		}
		if lower == "path" || lower == "-path" {
			if i+1 < len(tokens) {
				path = strings.ReplaceAll(strings.TrimSpace(tokens[i+1]), "\"", "")
				i++
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

// mkfile crea el archivo en la partición montada
func mkfile(path string, crearPadres bool, size int64, cont string) string {
	fmt.Printf("🔧 DEBUG: MKFILE path='%s' -r=%t size=%d cont='%s'\n", path, crearPadres, size, cont)

	sesion := ObtenerSesionActiva()
	var pathDisco string
	particion := GetMount("MKFILE", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("MKFILE", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	filePath := strings.ReplaceAll(pathDisco, "\"", "")
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return Utils.Error("MKFILE", "No se pudo abrir el disco: "+err.Error())
	}
	defer file.Close()

	// Leer superbloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	readSize := int(unsafe.Sizeof(Structs.SuperBloque{}))
	tmp := make([]byte, readSize)
	if _, err := file.Read(tmp); err != nil {
		return Utils.Error("MKFILE", "Error al leer superbloque: "+err.Error())
	}
	if err := binary.Read(bytes.NewBuffer(tmp), binary.BigEndian, &super); err != nil {
		return Utils.Error("MKFILE", "Error al decodificar superbloque: "+err.Error())
	}

	// tamaños y offsets
	tamInodo := int(unsafe.Sizeof(Structs.Inodos{}))
	tamBloqueArch := int(unsafe.Sizeof(Structs.BloquesArchivos{}))
	tamBloqueCarp := int(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	inodoStart := super.S_inode_start
	blockStart := super.S_block_start
	bmInodosStart := super.S_bm_inode_start
	bmBloquesStart := super.S_bm_block_start

	trimmed := strings.TrimSpace(path)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return Utils.Error("MKFILE", "Ruta inválida")
	}
	parts := strings.Split(trimmed, "/")[1:]
	if len(parts) == 0 {
		return Utils.Error("MKFILE", "Ruta inválida")
	}
	filename := parts[len(parts)-1]
	parentComponents := parts[:len(parts)-1]
	parentPath := "/"
	if len(parentComponents) > 0 {
		parentPath = "/" + strings.Join(parentComponents, "/")
	}

	// Asegurar existencia del padre (usar mkdir si hace falta)
	if parentPath != "/" {
		res := mkdir(parentPath, crearPadres)
		if strings.Contains(res, "ERROR") || strings.Contains(res, "❌") {
			if !crearPadres {
				return Utils.Error("MKFILE", "No existe el directorio padre: "+parentPath)
			}
		}
	}

	// Leer inodo raíz y recorrer hasta el inodo padre
	var inode Structs.Inodos
	file.Seek(inodoStart, 0)
	dataInodo := make([]byte, tamInodo)
	if _, err := file.Read(dataInodo); err != nil {
		return Utils.Error("MKFILE", "Error al leer inodo raíz: "+err.Error())
	}
	if err := binary.Read(bytes.NewBuffer(dataInodo), binary.BigEndian, &inode); err != nil {
		return Utils.Error("MKFILE", "Error al decodificar inodo raíz: "+err.Error())
	}

	currentInode := inode
	currentInodeOffset := inodoStart

	if parentPath != "/" {
		comps := strings.Split(strings.Trim(parentPath, "/"), "/")
		for _, comp := range comps {
			if comp == "" {
				continue
			}
			found := false
			for b := 0; b < len(currentInode.I_block); b++ {
				blk := currentInode.I_block[b]
				if blk == int64(-1) {
					continue
				}
				posBloque := blockStart + int64(tamBloqueCarp)*blk
				var bc Structs.BloquesCarpetas
				file.Seek(posBloque, 0)
				tmpb := make([]byte, tamBloqueCarp)
				if _, err := file.Read(tmpb); err != nil {
					return Utils.Error("MKFILE", "Error al leer bloque de carpetas: "+err.Error())
				}
				if err := binary.Read(bytes.NewBuffer(tmpb), binary.BigEndian, &bc); err != nil {
					return Utils.Error("MKFILE", "Error al decodificar bloque de carpetas: "+err.Error())
				}
				for e := 0; e < len(bc.B_content); e++ {
					name := strings.Trim(string(bc.B_content[e].B_name[:]), "\x00")
					if name == comp && bc.B_content[e].B_inodo != int64(-1) {
						inodoIdx := bc.B_content[e].B_inodo
						posInodo := inodoStart + int64(tamInodo)*inodoIdx
						file.Seek(posInodo, 0)
						dataIn := make([]byte, tamInodo)
						if _, err := file.Read(dataIn); err != nil {
							return Utils.Error("MKFILE", "Error al leer inodo destino: "+err.Error())
						}
						var iTmp Structs.Inodos
						if err := binary.Read(bytes.NewBuffer(dataIn), binary.BigEndian, &iTmp); err != nil {
							return Utils.Error("MKFILE", "Error al decodificar inodo destino: "+err.Error())
						}
						currentInode = iTmp
						currentInodeOffset = posInodo
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return Utils.Error("MKFILE", "No existe el directorio padre: "+parentPath)
			}
		}
	}

	// Buscar entrada libre en bloque padre o asignar bloque
	placed := false
	var parentBloquePos int64
	var parentBlock Structs.BloquesCarpetas
	for b := 0; b < len(currentInode.I_block) && !placed; b++ {
		blk := currentInode.I_block[b]
		if blk == int64(-1) {
			// asignar nuevo bloque al padre
			super.S_first_blo++
			nBloque := super.S_first_blo
			currentInode.I_block[b] = nBloque
			// marcar bitmap block
			file.Seek(bmBloquesStart+int64(nBloque), 0)
			file.Write([]byte{'1'})

			var newBlock Structs.BloquesCarpetas
			for i := 0; i < len(newBlock.B_content); i++ {
				newBlock.B_content[i].B_inodo = int64(-1)
				for j := 0; j < len(newBlock.B_content[i].B_name); j++ {
					newBlock.B_content[i].B_name[j] = 0
				}
			}
			copy(newBlock.B_content[0].B_name[:], ".")
			newBlock.B_content[0].B_inodo = int64((currentInodeOffset - inodoStart) / int64(tamInodo))
			copy(newBlock.B_content[1].B_name[:], "..")
			newBlock.B_content[1].B_inodo = int64((currentInodeOffset - inodoStart) / int64(tamInodo))

			posNewBlock := blockStart + int64(tamBloqueCarp)*nBloque
			file.Seek(posNewBlock, 0)
			var bufNb bytes.Buffer
			if err := binary.Write(&bufNb, binary.BigEndian, &newBlock); err != nil {
				return Utils.Error("MKFILE", "Error al serializar nuevo bloque padre: "+err.Error())
			}
			if _, err := file.Write(bufNb.Bytes()); err != nil {
				return Utils.Error("MKFILE", "Error al escribir nuevo bloque padre: "+err.Error())
			}

			// persistir inodo padre actualizado
			file.Seek(currentInodeOffset, 0)
			var bufPad bytes.Buffer
			if err := binary.Write(&bufPad, binary.BigEndian, &currentInode); err != nil {
				return Utils.Error("MKFILE", "Error al serializar inodo padre: "+err.Error())
			}
			if _, err := file.Write(bufPad.Bytes()); err != nil {
				return Utils.Error("MKFILE", "Error al escribir inodo padre: "+err.Error())
			}

			// actualizar superbloque en disco
			file.Seek(particion.Part_start, 0)
			var bufS1 bytes.Buffer
			if err := binary.Write(&bufS1, binary.BigEndian, &super); err != nil {
				return Utils.Error("MKFILE", "Error al serializar superbloque: "+err.Error())
			}
			if _, err := file.Write(bufS1.Bytes()); err != nil {
				return Utils.Error("MKFILE", "Error al escribir superbloque: "+err.Error())
			}
		}

		blk2 := currentInode.I_block[b]
		if blk2 == int64(-1) {
			continue
		}
		posBloque2 := blockStart + int64(tamBloqueCarp)*blk2
		file.Seek(posBloque2, 0)
		tmpb2 := make([]byte, tamBloqueCarp)
		if _, err := file.Read(tmpb2); err != nil {
			return Utils.Error("MKFILE", "Error al leer bloque padre: "+err.Error())
		}
		if err := binary.Read(bytes.NewBuffer(tmpb2), binary.BigEndian, &parentBlock); err != nil {
			return Utils.Error("MKFILE", "Error al decodificar bloque padre: "+err.Error())
		}
		for e := 0; e < len(parentBlock.B_content); e++ {
			if parentBlock.B_content[e].B_inodo == int64(-1) {
				// reservar entrada temporal
				parentBlock.B_content[e].B_inodo = int64(-2)
				copy(parentBlock.B_content[e].B_name[:], filename)
				file.Seek(posBloque2, 0)
				var bufPb bytes.Buffer
				if err := binary.Write(&bufPb, binary.BigEndian, &parentBlock); err != nil {
					return Utils.Error("MKFILE", "Error al serializar bloque padre: "+err.Error())
				}
				if _, err := file.Write(bufPb.Bytes()); err != nil {
					return Utils.Error("MKFILE", "Error al escribir bloque padre: "+err.Error())
				}
				parentBloquePos = posBloque2
				placed = true
				break
			}
		}
	}

	if !placed {
		return Utils.Error("MKFILE", "No hay espacio para crear el archivo: "+filename)
	}

	// asignar nuevo inodo
	super.S_firts_ino++
	nInodo := super.S_firts_ino
	// marcar bitmap inodo
	file.Seek(bmInodosStart+int64(nInodo), 0)
	file.Write([]byte{'1'})

	var newInodo Structs.Inodos
	newInodo.I_uid = int64(sesion.Uid)
	newInodo.I_gid = int64(sesion.Gid)
	newInodo.I_size = int64(size)
	newInodo.I_type = 1 // archivo
	newInodo.I_perm = 664
	for ib := 0; ib < len(newInodo.I_block); ib++ {
		newInodo.I_block[ib] = int64(-1)
	}

	// preparar contenido
	var contentBytes []byte
	if cont != "" {
		cb, err := os.ReadFile(strings.ReplaceAll(cont, "\"", ""))
		if err != nil {
			return Utils.Error("MKFILE", "No se pudo leer archivo de contenido: "+err.Error())
		}
		if size > 0 && int64(len(cb)) > size {
			contentBytes = cb[:size]
		} else {
			contentBytes = cb
			for int64(len(contentBytes)) < size {
				contentBytes = append(contentBytes, byte('0'+(len(contentBytes)%10)))
			}
		}
	} else {
		for int64(len(contentBytes)) < size {
			contentBytes = append(contentBytes, byte('0'+(len(contentBytes)%10)))
		}
	}

	// escribir bloques de archivo
	bytesRemaining := int64(len(contentBytes))
	offset := int64(0)
	blockIdx := 0
	for bytesRemaining > 0 && blockIdx < len(newInodo.I_block) {
		super.S_first_blo++
		nBloque := super.S_first_blo
		newInodo.I_block[blockIdx] = nBloque

		// marcar bitmap block
		file.Seek(bmBloquesStart+int64(nBloque), 0)
		file.Write([]byte{'1'})

		var block Structs.BloquesArchivos
		// inicializar
		for i := 0; i < len(block.B_content); i++ {
			block.B_content[i] = 0
		}
		toWrite := int64(len(block.B_content))
		if bytesRemaining < toWrite {
			toWrite = bytesRemaining
		}
		copy(block.B_content[:toWrite], contentBytes[offset:offset+toWrite])

		posBloque := blockStart + int64(tamBloqueArch)*nBloque
		file.Seek(posBloque, 0)
		var bufB bytes.Buffer
		if err := binary.Write(&bufB, binary.BigEndian, &block); err != nil {
			return Utils.Error("MKFILE", "Error al serializar bloque archivo: "+err.Error())
		}
		if _, err := file.Write(bufB.Bytes()); err != nil {
			return Utils.Error("MKFILE", "Error al escribir bloque archivo: "+err.Error())
		}

		bytesRemaining -= toWrite
		offset += toWrite
		blockIdx++
	}

	// escribir inodo nuevo
	posNewInodo := inodoStart + int64(tamInodo)*nInodo
	file.Seek(posNewInodo, 0)
	var bufNi bytes.Buffer
	if err := binary.Write(&bufNi, binary.BigEndian, &newInodo); err != nil {
		return Utils.Error("MKFILE", "Error al serializar nuevo inodo: "+err.Error())
	}
	if _, err := file.Write(bufNi.Bytes()); err != nil {
		return Utils.Error("MKFILE", "Error al escribir nuevo inodo: "+err.Error())
	}

	// actualizar bloque padre: buscar entrada con nombre y -2
	file.Seek(parentBloquePos, 0)
	var updatedParent Structs.BloquesCarpetas
	tmpb := make([]byte, tamBloqueCarp)
	if _, err := file.Read(tmpb); err != nil {
		return Utils.Error("MKFILE", "Error al leer bloque padre para actualizar entrada: "+err.Error())
	}
	if err := binary.Read(bytes.NewBuffer(tmpb), binary.BigEndian, &updatedParent); err != nil {
		return Utils.Error("MKFILE", "Error al decodificar bloque padre: "+err.Error())
	}
	for e := 0; e < len(updatedParent.B_content); e++ {
		name := strings.Trim(string(updatedParent.B_content[e].B_name[:]), "\x00")
		if name == filename && updatedParent.B_content[e].B_inodo == int64(-2) {
			updatedParent.B_content[e].B_inodo = nInodo
			break
		}
	}
	file.Seek(parentBloquePos, 0)
	var bufPf bytes.Buffer
	if err := binary.Write(&bufPf, binary.BigEndian, &updatedParent); err != nil {
		return Utils.Error("MKFILE", "Error al serializar bloque padre final: "+err.Error())
	}
	if _, err := file.Write(bufPf.Bytes()); err != nil {
		return Utils.Error("MKFILE", "Error al escribir bloque padre final: "+err.Error())
	}

	// persistir superbloque actualizado
	file.Seek(particion.Part_start, 0)
	var bufS bytes.Buffer
	if err := binary.Write(&bufS, binary.BigEndian, &super); err != nil {
		return Utils.Error("MKFILE", "Error al serializar superbloque: "+err.Error())
	}
	if _, err := file.Write(bufS.Bytes()); err != nil {
		return Utils.Error("MKFILE", "Error al escribir superbloque: "+err.Error())
	}
	file.Sync()

	return Utils.Mensaje("MKFILE", fmt.Sprintf("Archivo '%s' creado correctamente", path))
}
