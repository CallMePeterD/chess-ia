package book

import (
	"math/bits"
	"github.com/CallMePeterD/chess-ia/rules"
)

// PolyglotHash calcula a chave Zobrist estritamente de acordo com as regras PolyGlot
func PolyglotHash(p rules.Position) uint64 {
	var hash uint64

	// 1. Peças (Total de 768 deslocamentos: 64 casas * 12 peças PolyGlot)
	for pt := rules.Pawn; pt <= rules.King; pt++ {
		for c := rules.White; c <= rules.Black; c++ {
			// A fórmula Polyglot: (peão=0..rei=5) * 2 + (branco=1, preto=0)
			polyPiece := (int(pt)) * 2
			if c == rules.White {
				polyPiece += 1
			}

			pieces := p.Pieces[pt] & p.Colors[c]
			for pieces != 0 {
				sq := bits.TrailingZeros64(uint64(pieces))
				
				offset := 64 * polyPiece + sq
				hash ^= PolyglotRandoms[offset]
				
				pieces &= pieces - 1
			}
		}
	}

	// 2. Direitos de Roque (Deslocamentos 768 a 771)
	if p.Castling & 1 != 0 { hash ^= PolyglotRandoms[768] } // Branco Rei
	if p.Castling & 2 != 0 { hash ^= PolyglotRandoms[769] } // Branco Dama
	if p.Castling & 4 != 0 { hash ^= PolyglotRandoms[770] } // Preto Rei
	if p.Castling & 8 != 0 { hash ^= PolyglotRandoms[771] } // Preto Dama

	// 3. En Passant Exigente (Deslocamentos 772 a 779)
	// O Polyglot OBRIGA a que um peão inimigo exista na fileira lateral para o hash contar
	if p.EnPassant != -1 {
		epFile := p.EnPassant % 8
		hasValidCapturer := false
		
		if p.SideToMove == rules.White {
			// Pretas jogaram e7e5. O alvo é e6. Queremos saber se as Brancas têm peão em d5 ou f5.
			if epFile > 0 && p.Pieces[rules.Pawn].Has(p.EnPassant-9) && p.Colors[rules.White].Has(p.EnPassant-9) { hasValidCapturer = true }
			if epFile < 7 && p.Pieces[rules.Pawn].Has(p.EnPassant-7) && p.Colors[rules.White].Has(p.EnPassant-7) { hasValidCapturer = true }
		} else {
			// Brancas jogaram d2d4. O alvo é d3. Queremos saber se as Pretas têm peão em c4 ou e4.
			if epFile > 0 && p.Pieces[rules.Pawn].Has(p.EnPassant+7) && p.Colors[rules.Black].Has(p.EnPassant+7) { hasValidCapturer = true }
			if epFile < 7 && p.Pieces[rules.Pawn].Has(p.EnPassant+9) && p.Colors[rules.Black].Has(p.EnPassant+9) { hasValidCapturer = true }
		}

		if hasValidCapturer {
			hash ^= PolyglotRandoms[772 + epFile]
		}
	}

	// 4. Vez de Jogar (Deslocamento 780)
	if p.SideToMove == rules.White {
		hash ^= PolyglotRandoms[780]
	}

	return hash
}