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

// ValidarDatosMKDIR valida parámetros para MKDIR y llama a mkdir
func ValidarDatosMKDIR(tokens []string) string {
	// DEBUG: mostrar tokens que recibe el validador
	fmt.Printf("🔧 DEBUG: ValidarDatosMKDIR tokens=%v\n", tokens)

	if len(tokens) < 1 {
		return Utils.Error("MKDIR", "Se requieren parámetros. Uso: -path=/ruta [-p]")
	}

	var path string
	crearPadres := false

	for i := 0; i < len(tokens); i++ {
		tok := strings.TrimSpace(tokens[i])
		if tok == "" {
			continue
		}
		lower := strings.ToLower(tok)

		// casos simples
		if lower == "-p" || lower == "p" {
			crearPadres = true
			continue
		}

		// token tipo "path=/algo" o "-path=/algo" o "p-path=/algo"
		if strings.Contains(lower, "path=") {
			parts := strings.SplitN(tok, "=", 2)
			if len(parts) == 2 {
				path = strings.ReplaceAll(parts[1], "\"", "")
			}
			// marcar crearPadres solo si el prefijo indica explícitamente -p (p, -p, p-path, -p-path)
			prefix := strings.ToLower(parts[0])
			if prefix == "p" || prefix == "-p" || strings.HasPrefix(prefix, "p-") || strings.HasPrefix(prefix, "-p-") {
				crearPadres = true
			}
			continue
		}

		// token "path" o "-path" seguido del valor en el siguiente token
		if lower == "path" || lower == "-path" {
			if i+1 < len(tokens) {
				path = strings.ReplaceAll(strings.TrimSpace(tokens[i+1]), "\"", "")
				i++ // consumir el siguiente token
			}
			continue
		}

		// token pegado como "p-path=/ruta" o "-p-path=/ruta"
		if strings.HasPrefix(lower, "p-path=") || strings.HasPrefix(lower, "-p-path=") {
			if idx := strings.Index(lower, "path="); idx != -1 {
				after := tok[idx:]
				parts := strings.SplitN(after, "=", 2)
				if len(parts) == 2 {
					path = strings.ReplaceAll(parts[1], "\"", "")
					crearPadres = true
				}
			}
			continue
		}
	}

	// DEBUG: mostrar valores parseados
	fmt.Printf("🔧 DEBUG: ValidarDatosMKDIR -> path='%s' crearPadres=%t\n", path, crearPadres)

	if path == "" {
		return Utils.Error("MKDIR", "El parámetro -path es obligatorio")
	}

	// Requiere sesión activa
	if !EstaLogueado() {
		return Utils.Error("MKDIR", "Debe iniciar sesión para ejecutar este comando")
	}

	return mkdir(path, crearPadres)
}

// mkdir crea la(s) carpeta(s) en el filesystem de la partición montada
func mkdir(path string, crearPadres bool) string {
	fmt.Printf("🔧 DEBUG: MKDIR path='%s' -p=%t\n", path, crearPadres)

	sesion := ObtenerSesionActiva()
	var pathDisco string
	particion := GetMount("MKDIR", sesion.Id, &pathDisco)
	if particion == nil {
		return Utils.Error("MKDIR", "No se encontró la partición montada con el ID: "+sesion.Id)
	}

	filePath := strings.ReplaceAll(pathDisco, "\"", "")
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return Utils.Error("MKDIR", "No se pudo abrir el disco: "+err.Error())
	}
	defer file.Close()

	// Leer SuperBloque
	super := Structs.NewSuperBloque()
	file.Seek(particion.Part_start, 0)
	var bufSuper bytes.Buffer
	readSize := int(unsafe.Sizeof(Structs.SuperBloque{}))
	tmp := make([]byte, readSize)
	if _, err := file.Read(tmp); err != nil {
		return Utils.Error("MKDIR", "Error al leer superbloque: "+err.Error())
	}
	bufSuper.Write(tmp)
	if err := binary.Read(&bufSuper, binary.BigEndian, &super); err != nil {
		return Utils.Error("MKDIR", "Error al decodificar superbloque: "+err.Error())
	}

	// Tamaños y punteros
	tamInodo := int(unsafe.Sizeof(Structs.Inodos{}))
	tamBloque := int(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	inodoStart := super.S_inode_start
	blockStart := super.S_block_start
	bmInodosStart := super.S_bm_inode_start
	bmBloquesStart := super.S_bm_block_start

	// Normalizar path y obtener componentes
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return Utils.Error("MKDIR", "Ruta inválida")
	}
	components := strings.Split(trimmed, "/")[1:] // eliminar primer elemento vacío

	// Leer inodo raíz (índice 0)
	var inode Structs.Inodos
	file.Seek(inodoStart, 0)
	dataInodo := make([]byte, tamInodo)
	if _, err := file.Read(dataInodo); err != nil {
		return Utils.Error("MKDIR", "Error al leer inodo raíz: "+err.Error())
	}
	if err := binary.Read(bytes.NewBuffer(dataInodo), binary.BigEndian, &inode); err != nil {
		return Utils.Error("MKDIR", "Error al decodificar inodo raíz: "+err.Error())
	}

	currentInode := inode
	currentInodeOffset := inodoStart // posición del inodo actual en disco (int64)

	// recorrer components
	for idx, comp := range components {
		if comp == "" {
			continue
		}
		found := false

		// buscar en los bloques del inodo actual
		for b := 0; b < len(currentInode.I_block); b++ {
			blk := currentInode.I_block[b]
			if blk == int64(-1) {
				continue
			}
			// calcular offset bloque en disco
			posBloque := blockStart + int64(tamBloque)*blk
			var bc Structs.BloquesCarpetas
			file.Seek(posBloque, 0)
			tmpb := make([]byte, tamBloque)
			if _, err := file.Read(tmpb); err != nil {
				return Utils.Error("MKDIR", "Error al leer bloque de carpetas: "+err.Error())
			}
			if err := binary.Read(bytes.NewBuffer(tmpb), binary.BigEndian, &bc); err != nil {
				return Utils.Error("MKDIR", "Error al decodificar bloque de carpetas: "+err.Error())
			}

			// revisar entradas
			for e := 0; e < len(bc.B_content); e++ {
				name := strings.Trim(string(bc.B_content[e].B_name[:]), "\x00")
				if name == comp && bc.B_content[e].B_inodo != int64(-1) {
					// avanzar al inodo apuntado
					inodoIdx := bc.B_content[e].B_inodo
					// leer inodo destino
					posInodo := inodoStart + int64(tamInodo)*inodoIdx
					file.Seek(posInodo, 0)
					dataIn := make([]byte, tamInodo)
					if _, err := file.Read(dataIn); err != nil {
						return Utils.Error("MKDIR", "Error al leer inodo destino: "+err.Error())
					}
					var iTmp Structs.Inodos
					if err := binary.Read(bytes.NewBuffer(dataIn), binary.BigEndian, &iTmp); err != nil {
						return Utils.Error("MKDIR", "Error al decodificar inodo destino: "+err.Error())
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
			// no existe componente comp
			if !crearPadres && idx != len(components)-1 {
				// padre inexistente
				return Utils.Error("MKDIR", "No existe el directorio padre: "+comp)
			}
			// crear el directorio comp dentro del currentInode
			placed := false
			for b := 0; b < len(currentInode.I_block) && !placed; b++ {
				blk := currentInode.I_block[b]
				if blk == int64(-1) {
					// asignar nuevo bloque al padre
					super.S_first_blo++
					nBloque := super.S_first_blo // int64
					currentInode.I_block[b] = nBloque
					// marcar bitmap block (offset en bytes: bmBloquesStart + nBloque)
					file.Seek(bmBloquesStart+int64(nBloque), 0)
					file.Write([]byte{'1'})

					// crear bloque vacío y escribirlo (inicializar entradas como libres)
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

					// escribir block en disco
					posNewBlock := blockStart + int64(tamBloque)*nBloque
					file.Seek(posNewBlock, 0)
					var bufNb bytes.Buffer
					if err := binary.Write(&bufNb, binary.BigEndian, &newBlock); err != nil {
						return Utils.Error("MKDIR", "Error al serializar nuevo bloque: "+err.Error())
					}
					if _, err := file.Write(bufNb.Bytes()); err != nil {
						return Utils.Error("MKDIR", "Error al escribir nuevo bloque: "+err.Error())
					}

					// persistir inodo padre actualizado (I_block modificado)
					file.Seek(currentInodeOffset, 0)
					var bufPad bytes.Buffer
					if err := binary.Write(&bufPad, binary.BigEndian, &currentInode); err != nil {
						return Utils.Error("MKDIR", "Error al serializar inodo padre: "+err.Error())
					}
					if _, err := file.Write(bufPad.Bytes()); err != nil {
						return Utils.Error("MKDIR", "Error al escribir inodo padre: "+err.Error())
					}

					// actualizar superbloque en disco tras asignación de bloque
					file.Seek(particion.Part_start, 0)
					var bufS1 bytes.Buffer
					if err := binary.Write(&bufS1, binary.BigEndian, &super); err != nil {
						return Utils.Error("MKDIR", "Error al serializar superbloque: "+err.Error())
					}
					if _, err := file.Write(bufS1.Bytes()); err != nil {
						return Utils.Error("MKDIR", "Error al escribir superbloque: "+err.Error())
					}
				}

				// leer de nuevo el bloque (sea asignado o existente) y buscar entrada libre
				blk2 := currentInode.I_block[b]
				if blk2 == int64(-1) {
					continue
				}
				posBloque2 := blockStart + int64(tamBloque)*blk2
				file.Seek(posBloque2, 0)
				var bc2 Structs.BloquesCarpetas
				tmpb2 := make([]byte, tamBloque)
				if _, err := file.Read(tmpb2); err != nil {
					return Utils.Error("MKDIR", "Error al leer bloque para inserción: "+err.Error())
				}
				if err := binary.Read(bytes.NewBuffer(tmpb2), binary.BigEndian, &bc2); err != nil {
					return Utils.Error("MKDIR", "Error al decodificar bloque para inserción: "+err.Error())
				}

				for e := 0; e < len(bc2.B_content); e++ {
					if bc2.B_content[e].B_inodo == int64(-1) {
						// asignar nuevo inodo para el directorio
						super.S_firts_ino++
						nInodo := super.S_firts_ino // int64
						// marcar bitmap inodo
						file.Seek(bmInodosStart+int64(nInodo), 0)
						file.Write([]byte{'1'})

						// crear inodo nuevo
						var newInodo Structs.Inodos
						newInodo.I_uid = int64(sesion.Uid)
						newInodo.I_gid = int64(sesion.Gid)
						newInodo.I_size = int64(tamBloque)
						newInodo.I_type = 0 // directorio
						newInodo.I_perm = 664
						// asignar bloque para el nuevo inodo
						super.S_first_blo++
						nBloque2 := super.S_first_blo // int64
						// inicializar blocks del inodo y asignar el primero
						for ib := 0; ib < len(newInodo.I_block); ib++ {
							newInodo.I_block[ib] = int64(-1)
						}
						newInodo.I_block[0] = nBloque2

						// escribir inodo en su posición
						posNewInodo := inodoStart + int64(tamInodo)*nInodo
						file.Seek(posNewInodo, 0)
						var bufNi bytes.Buffer
						if err := binary.Write(&bufNi, binary.BigEndian, &newInodo); err != nil {
							return Utils.Error("MKDIR", "Error al serializar nuevo inodo: "+err.Error())
						}
						if _, err := file.Write(bufNi.Bytes()); err != nil {
							return Utils.Error("MKDIR", "Error al escribir nuevo inodo: "+err.Error())
						}

						// crear bloque de la carpeta nueva (con . y ..) e inicializar entradas libres
						var newBlock Structs.BloquesCarpetas
						for i := 0; i < len(newBlock.B_content); i++ {
							newBlock.B_content[i].B_inodo = int64(-1)
							for j := 0; j < len(newBlock.B_content[i].B_name); j++ {
								newBlock.B_content[i].B_name[j] = 0
							}
						}
						copy(newBlock.B_content[0].B_name[:], ".")
						newBlock.B_content[0].B_inodo = nInodo
						copy(newBlock.B_content[1].B_name[:], "..")
						padreIdx := int64((currentInodeOffset - inodoStart) / int64(tamInodo))
						newBlock.B_content[1].B_inodo = padreIdx

						// escribir nuevo bloque en disco
						posBloqueNew := blockStart + int64(tamBloque)*nBloque2
						file.Seek(posBloqueNew, 0)
						var bufBn bytes.Buffer
						if err := binary.Write(&bufBn, binary.BigEndian, &newBlock); err != nil {
							return Utils.Error("MKDIR", "Error al serializar bloque nuevo: "+err.Error())
						}
						if _, err := file.Write(bufBn.Bytes()); err != nil {
							return Utils.Error("MKDIR", "Error al escribir bloque nuevo: "+err.Error())
						}

						// actualizar la entrada del bloque padre
						bc2.B_content[e].B_inodo = nInodo
						copy(bc2.B_content[e].B_name[:], comp)
						// escribir el bloque padre actualizado
						file.Seek(posBloque2, 0)
						var bufBp bytes.Buffer
						if err := binary.Write(&bufBp, binary.BigEndian, &bc2); err != nil {
							return Utils.Error("MKDIR", "Error al serializar bloque padre: "+err.Error())
						}
						if _, err := file.Write(bufBp.Bytes()); err != nil {
							return Utils.Error("MKDIR", "Error al escribir bloque padre: "+err.Error())
						}

						// escribir superbloque actualizado (contadores)
						file.Seek(particion.Part_start, 0)
						var bufS bytes.Buffer
						if err := binary.Write(&bufS, binary.BigEndian, &super); err != nil {
							return Utils.Error("MKDIR", "Error al serializar superbloque: "+err.Error())
						}
						if _, err := file.Write(bufS.Bytes()); err != nil {
							return Utils.Error("MKDIR", "Error al escribir superbloque: "+err.Error())
						}
						file.Sync()

						placed = true
						break
					}
				}
			}
			if !placed {
				return Utils.Error("MKDIR", "No hay espacio para crear el directorio: "+comp)
			}

			// re-lectura del inodo padre para continuar
			file.Seek(currentInodeOffset, 0)
			dtmp := make([]byte, tamInodo)
			if _, err := file.Read(dtmp); err == nil {
				var itmp Structs.Inodos
				_ = binary.Read(bytes.NewBuffer(dtmp), binary.BigEndian, &itmp)
				currentInode = itmp
			}
		}
	} // fin recorrido components

	return Utils.Mensaje("MKDIR", fmt.Sprintf("Directorio '%s' creado correctamente", path))
}
