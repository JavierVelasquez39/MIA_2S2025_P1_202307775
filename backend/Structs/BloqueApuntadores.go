package Structs

type BloquesApuntadores struct {
	B_pointers [16]int64
}

func NewBloquesApuntadores() BloquesApuntadores {
	var bloque BloquesApuntadores
	for i := 0; i < len(bloque.B_pointers); i++ {
		bloque.B_pointers[i] = -1
	}
	return bloque
}
