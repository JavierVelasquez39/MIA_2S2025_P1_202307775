package Structs

type BloquesCarpetas struct {
	B_content [4]Content
}

func NewBloquesCarpetas() BloquesCarpetas {
	var bl BloquesCarpetas
	for i := 0; i < len(bl.B_content); i++ {
		bl.B_content[i] = NewContent()
	}
	return bl
}

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
