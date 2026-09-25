package rules

// Position representa o estado completo do jogo sem alocações dinâmicas
type Position struct {
	Pieces     [6]Bitboard // Peões, Cavalos, Bispos, Torres, Damas, Reis
	Colors     [2]Bitboard // Todas as peças Brancas e Todas as peças Pretas
	SideToMove Color
	Castling   uint8       // 4 bits: 1=WK, 2=WQ, 4=BK, 8=BQ
	EnPassant  int         // Casa válida para en passant (ou -1)
	Halfmove   int         // Regra dos 50 lances
	Fullmove   int         // Número do lance
	HashKey    uint64      // O Zobrist hash atualizado em tempo real
}

// Hash devolve a chave instantaneamente, sem precisar de reprocessar nada
func (p *Position) Hash() uint64 {
	return p.HashKey
}

// ComputeHash varre a posição para gerar a chave do zero (usado apenas ao ler uma FEN)
func (p *Position) ComputeHash() uint64 {
	var h uint64
	for c := White; c <= Black; c++ {
		for pt := Pawn; pt <= King; pt++ {
			bb := p.Pieces[pt] & p.Colors[c]
			for sq := 0; sq < 64; sq++ {
				if bb.Has(sq) {
					h ^= ZobristPieces[c][pt][sq]
				}
			}
		}
	}
	if p.SideToMove == Black {
		h ^= ZobristSide
	}
	h ^= ZobristCastling[p.Castling]
	if p.EnPassant != -1 {
		h ^= ZobristEnPassant[p.EnPassant]
	}
	return h
}

// Occupied devolve um Bitboard com todas as peças do tabuleiro
func (p *Position) Occupied() Bitboard {
	return p.Colors[White] | p.Colors[Black]
}