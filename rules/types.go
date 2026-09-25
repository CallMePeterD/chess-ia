package rules

// Color representa o lado que joga (0 = Brancas, 1 = Pretas)
type Color int8

const (
	White Color = iota
	Black
	Both
)

// PieceType vai de 0 a 5
type PieceType int8

const (
	Pawn PieceType = iota
	Knight
	Bishop
	Rook
	Queen
	King
	NoPiece
)

// Bitboard é a nossa representação de 64 casas num único número
type Bitboard uint64

// Set define o bit de uma casa como 1
func (b *Bitboard) Set(sq int) {
	*b |= (1 << sq)
}

// Clear define o bit de uma casa como 0
func (b *Bitboard) Clear(sq int) {
	*b &= ^(1 << sq)
}

// Has verifica se existe uma peça na casa especificada
func (b Bitboard) Has(sq int) bool {
	return (b & (1 << sq)) != 0
}

// PopCount conta quantos bits 1 existem no Bitboard (quantas peças daquele tipo)
func (b Bitboard) PopCount() int {
	var count int
	for b != 0 {
		count++
		b &= b - 1 // Reseta o bit 1 menos significativo
	}
	return count
}