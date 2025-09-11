package Comandos

import (
	"bytes"
	"os"
	"unsafe"

	"godisk-backend/Structs"
)

// fileBlockOffset devuelve la posición en bytes en el disco para un bloque de archivo
// Los bloques (slots) en disco se indexan en ranuras del tamaño de BloquesCarpetas
// Por convención del código existente: bloque 0 está en S_block_start,
// bloque n está en S_block_start + n * sizeof(BloquesCarpetas{}).
// Esta función devuelve la posición de inicio de la ranura para el número de bloque
// provisto. Si blockNum < 0 devuelve S_block_start.
func fileBlockOffset(sb Structs.SuperBloque, blockNum int64) int64 {
	if blockNum < 0 {
		return sb.S_block_start
	}
	return sb.S_block_start + (blockNum * int64(unsafe.Sizeof(Structs.BloquesCarpetas{})))
}

// readFileBlock lee la ranura completa del disco donde se almacena un BloquesArchivos
// y retorna los bytes reales del BloquesArchivos (64 bytes) sin los nulos de relleno.
func readFileBlock(file *os.File, sb Structs.SuperBloque, blockNum int64) ([]byte, error) {
	pos := fileBlockOffset(sb, blockNum)
	if _, err := file.Seek(pos, 0); err != nil {
		return nil, err
	}
	slotSize := int64(unsafe.Sizeof(Structs.BloquesCarpetas{}))
	slotBuf := make([]byte, slotSize)
	if _, err := file.Read(slotBuf); err != nil {
		return nil, err
	}
	// El BloquesArchivos está almacenado en los primeros N bytes de la ranura
	archSize := int64(unsafe.Sizeof(Structs.BloquesArchivos{}))
	if archSize > int64(len(slotBuf)) {
		archSize = int64(len(slotBuf))
	}
	content := slotBuf[:archSize]
	trimmed := bytes.TrimRight(content, "\x00")
	return trimmed, nil
}
