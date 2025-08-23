package Comandos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"godisk-backend/Structs"
)

// escribirContenidoArchivo: función compartida para escribir archivos tipo users.txt
// Firma compatible con implementaciones previas: (pathDisco string, particion Structs.Particion, super Structs.SuperBloque, inodo Structs.Inodos, nuevoContenido string) error
func escribirContenidoArchivo(pathDisco string, particion Structs.Particion, super Structs.SuperBloque, inodo Structs.Inodos, nuevoContenido string) error {
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

				// Marcar el bitmap de bloques como ocupado (1) - usar ASCII '1' para compatibilidad con mkfs
				bitmapPos := super.S_bm_block_start + int64(super.S_first_blo-1)
				file.Seek(bitmapPos, 0)
				if _, err := file.Write([]byte{'1'}); err != nil {
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
