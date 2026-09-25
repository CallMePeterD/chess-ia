package rules

import (
	"math/rand"
	
)

var (
	ZobristPieces  [2][6][64]uint64 // [Color][PieceType][Square]
	ZobristSide    uint64           // Chave a aplicar se for a vez das Pretas
	ZobristCastling [16]uint64      // 4 bits de roque (K, Q, k, q)
	ZobristEnPassant [64]uint64     // Casa de En Passant
)

// init corre automaticamente quando o pacote é importado
func init() {
	// Semente fixa para garantir que as chaves são consistentes entre execuções
	// Se formos usar PolyGlot, teremos de substituir isto pelos valores exatos do padrão PolyGlot
	rng := rand.New(rand.NewSource(123456789))

	for c := 0; c < 2; c++ {
		for p := 0; p < 6; p++ {
			for sq := 0; sq < 64; sq++ {
				ZobristPieces[c][p][sq] = rng.Uint64()
			}
		}
	}
	ZobristSide = rng.Uint64()

	for i := 0; i < 16; i++ {
		ZobristCastling[i] = rng.Uint64()
	}

	for i := 0; i < 64; i++ {
		ZobristEnPassant[i] = rng.Uint64()
	}
}