package rules

import "math/bits"

// Máscaras de colunas para evitar saltos ilegais pelas bordas
const (
	FileA Bitboard = 0x0101010101010101
	FileB Bitboard = 0x0202020202020202
	FileG Bitboard = 0x4040404040404040
	FileH Bitboard = 0x8080808080808080
	
	Rank3 Bitboard = 0x0000000000FF0000
	Rank6 Bitboard = 0x0000FF0000000000
)

var KnightAttacks [64]Bitboard
var KingAttacks [64]Bitboard

func init() {
	// Pré-calcula os saltos do Cavalo para todas as 64 casas apenas uma vez no arranque
	for sq := 0; sq < 64; sq++ {
		bb := Bitboard(1 << sq)
		var attacks Bitboard
		
		attacks |= (bb << 17) &^ FileA
		attacks |= (bb << 15) &^ FileH
		attacks |= (bb << 10) &^ (FileA | FileB)
		attacks |= (bb << 6)  &^ (FileG | FileH)
		attacks |= (bb >> 17) &^ FileH
		attacks |= (bb >> 15) &^ FileA
		attacks |= (bb >> 10) &^ (FileG | FileH)
		attacks |= (bb >> 6)  &^ (FileA | FileB)
		
		KnightAttacks[sq] = attacks
	}
	// Pré-calcula os ataques do Rei para as 64 casas
	for sq := 0; sq < 64; sq++ {
		bb := Bitboard(1 << sq)
		var attacks Bitboard

		attacks |= (bb << 8) // Cima
		attacks |= (bb >> 8) // Baixo
		
		attacks |= (bb << 1) &^ FileA // Direita
		attacks |= (bb >> 1) &^ FileH // Esquerda
		
		attacks |= (bb << 9) &^ FileA // Diagonal Cima-Direita
		attacks |= (bb << 7) &^ FileH // Diagonal Cima-Esquerda
		attacks |= (bb >> 7) &^ FileA // Diagonal Baixo-Direita
		attacks |= (bb >> 9) &^ FileH // Diagonal Baixo-Esquerda

		KingAttacks[sq] = attacks
	}
}

// generateKings extrai os lances baseados no array pré-calculado.
func (p *Position) generateKings(moves *[]Move) {
	us := p.SideToMove
	them := 1 ^ us
	ourPieces := p.Colors[us]
	enemyPieces := p.Colors[them]
	
	king := p.Pieces[King] & ourPieces

	// O Rei só pode estar numa casa, mas usamos o loop na mesma por consistência
	for king != 0 {
		sq := bits.TrailingZeros64(uint64(king))
		attacks := KingAttacks[sq] &^ ourPieces 
		
		for attacks != 0 {
			target := bits.TrailingZeros64(uint64(attacks))
			flag := FlagQuiet
			if enemyPieces.Has(target) {
				flag = FlagCapture
			}
			*moves = append(*moves, NewMove(sq, target, flag))
			
			attacks &= attacks - 1
		}
		king &= king - 1
	}
	// Lógica de Roque (Castling)
	occupied := p.Occupied()
	if us == White {
		if p.Castling & 1 != 0 { // Roque Curto Branco
			// As casas de passagem e destino estão vazias? O Rei não está em cheque e não vai passar por perigo?
			if !occupied.Has(5) && !occupied.Has(6) && !p.IsAttacked(4, Black) && !p.IsAttacked(5, Black) {
				*moves = append(*moves, NewMove(4, 6, FlagKingCastle))
			}
		}
		if p.Castling & 2 != 0 { // Roque Longo Branco
			if !occupied.Has(1) && !occupied.Has(2) && !occupied.Has(3) && !p.IsAttacked(4, Black) && !p.IsAttacked(3, Black) {
				*moves = append(*moves, NewMove(4, 2, FlagQueenCastle))
			}
		}
	} else {
		if p.Castling & 4 != 0 { // Roque Curto Preto
			if !occupied.Has(61) && !occupied.Has(62) && !p.IsAttacked(60, White) && !p.IsAttacked(61, White) {
				*moves = append(*moves, NewMove(60, 62, FlagKingCastle))
			}
		}
		if p.Castling & 8 != 0 { // Roque Longo Preto
			if !occupied.Has(57) && !occupied.Has(58) && !occupied.Has(59) && !p.IsAttacked(60, White) && !p.IsAttacked(59, White) {
				*moves = append(*moves, NewMove(60, 58, FlagQueenCastle))
			}
		}
	}
}

// generateKnights extrai os lances baseados no array pré-calculado.
func (p *Position) generateKnights(moves *[]Move) {
	us := p.SideToMove
	them := 1 ^ us
	ourPieces := p.Colors[us]
	enemyPieces := p.Colors[them]
	
	knights := p.Pieces[Knight] & ourPieces

	// Enquanto houver cavalos no tabuleiro
	for knights != 0 {
		sq := bits.TrailingZeros64(uint64(knights))
		// Pega na máscara de ataques e remove as casas ocupadas pelas nossas próprias peças
		attacks := KnightAttacks[sq] &^ ourPieces 
		
		for attacks != 0 {
			target := bits.TrailingZeros64(uint64(attacks))
			flag := FlagQuiet
			if enemyPieces.Has(target) {
				flag = FlagCapture
			}
			*moves = append(*moves, NewMove(sq, target, flag))
			
			attacks &= attacks - 1 // Desliga o bit processado
		}
		knights &= knights - 1 // Desliga o cavalo processado
	}
}

func (p *Position) generatePawns(moves *[]Move) {
	us := p.SideToMove
	them := 1 ^ us
	pawns := p.Pieces[Pawn] & p.Colors[us]
	enemies := p.Colors[them]
	occupied := p.Occupied()

	if us == White {
		singlePush := (pawns << 8) &^ occupied
		pushTargets := singlePush
		for pushTargets != 0 {
			to := bits.TrailingZeros64(uint64(pushTargets))
			from := to - 8
			addPawnMoves(moves, from, to, false)
			pushTargets &= pushTargets - 1
		}

		doublePush := ((singlePush & Rank3) << 8) &^ occupied
		for doublePush != 0 {
			to := bits.TrailingZeros64(uint64(doublePush))
			from := to - 16
			*moves = append(*moves, NewMove(from, to, FlagDoublePawn))
			doublePush &= doublePush - 1
		}

		attacksLeft := (pawns << 7) &^ FileH & enemies
		for attacksLeft != 0 {
			to := bits.TrailingZeros64(uint64(attacksLeft))
			from := to - 7
			addPawnMoves(moves, from, to, true)
			attacksLeft &= attacksLeft - 1
		}

		attacksRight := (pawns << 9) &^ FileA & enemies
		for attacksRight != 0 {
			to := bits.TrailingZeros64(uint64(attacksRight))
			from := to - 9
			addPawnMoves(moves, from, to, true)
			attacksRight &= attacksRight - 1
		}
	} else {
		singlePush := (pawns >> 8) &^ occupied
		pushTargets := singlePush
		for pushTargets != 0 {
			to := bits.TrailingZeros64(uint64(pushTargets))
			from := to + 8
			addPawnMoves(moves, from, to, false)
			pushTargets &= pushTargets - 1
		}

		doublePush := ((singlePush & Rank6) >> 8) &^ occupied
		for doublePush != 0 {
			to := bits.TrailingZeros64(uint64(doublePush))
			from := to + 16
			*moves = append(*moves, NewMove(from, to, FlagDoublePawn))
			doublePush &= doublePush - 1
		}

		attacksLeft := (pawns >> 9) &^ FileH & enemies
		for attacksLeft != 0 {
			to := bits.TrailingZeros64(uint64(attacksLeft))
			from := to + 9
			addPawnMoves(moves, from, to, true)
			attacksLeft &= attacksLeft - 1
		}

		attacksRight := (pawns >> 7) &^ FileA & enemies
		for attacksRight != 0 {
			to := bits.TrailingZeros64(uint64(attacksRight))
			from := to + 7
			addPawnMoves(moves, from, to, true)
			attacksRight &= attacksRight - 1
		}
	}

	// En Passant
	if p.EnPassant != -1 {
		epBB := Bitboard(1 << p.EnPassant)
		if us == White {
			attackers := ((epBB >> 7) &^ FileA) | ((epBB >> 9) &^ FileH)
			attackers &= pawns
			for attackers != 0 {
				from := bits.TrailingZeros64(uint64(attackers))
				*moves = append(*moves, NewMove(from, p.EnPassant, FlagEP))
				attackers &= attackers - 1
			}
		} else {
			attackers := ((epBB << 7) &^ FileH) | ((epBB << 9) &^ FileA)
			attackers &= pawns
			for attackers != 0 {
				from := bits.TrailingZeros64(uint64(attackers))
				*moves = append(*moves, NewMove(from, p.EnPassant, FlagEP))
				attackers &= attackers - 1
			}
		}
	}
}

// getRookAttacks usa os Magic Bitboards para devolver instantaneamente os ataques da Torre
func getRookAttacks(sq int, occupied Bitboard) Bitboard {
	blockers := uint64(occupied & RookMasks[sq])
	index := (blockers * RookMagics[sq]) >> RookShifts[sq]
	return RookAttacks[sq][index]
}

// getBishopAttacks usa os Magic Bitboards para as diagonais
func getBishopAttacks(sq int, occupied Bitboard) Bitboard {
	blockers := uint64(occupied & BishopMasks[sq])
	index := (blockers * BishopMagics[sq]) >> BishopShifts[sq]
	return BishopAttacks[sq][index]
}

// getQueenAttacks é simplesmente a união (OR bit-a-bit) da Torre com o Bispo
func getQueenAttacks(sq int, occupied Bitboard) Bitboard {
	return getRookAttacks(sq, occupied) | getBishopAttacks(sq, occupied)
}

// generateSlidingPieces extrai os lances para Bispos, Torres e Damas
func (p *Position) generateSlidingPieces(moves *[]Move) {
	us := p.SideToMove
	them := 1 ^ us
	ourPieces := p.Colors[us]
	enemyPieces := p.Colors[them]
	occupied := p.Occupied()

	piecesToProcess := []struct {
		pt   PieceType
		calc func(sq int, occ Bitboard) Bitboard
	}{
		{Bishop, getBishopAttacks},
		{Rook, getRookAttacks},
		{Queen, getQueenAttacks}, // A Dama usa a função composta
	}

	for _, cfg := range piecesToProcess {
		pieces := p.Pieces[cfg.pt] & ourPieces

		for pieces != 0 {
			sq := bits.TrailingZeros64(uint64(pieces))
			// Pega nos ataques e remove as casas onde estão as nossas próprias peças (fogo amigo)
			attacks := cfg.calc(sq, occupied) &^ ourPieces
			
			for attacks != 0 {
				target := bits.TrailingZeros64(uint64(attacks))
				flag := FlagQuiet
				if enemyPieces.Has(target) {
					flag = FlagCapture
				}
				*moves = append(*moves, NewMove(sq, target, flag))
				
				attacks &= attacks - 1
			}
			pieces &= pieces - 1
		}
	}
}

// LegalMoves agora só precisa de chamar os geradores específicos
func (p *Position) LegalMoves() []Move {
	// Pré-aloca capacidade para evitar reajustes de memória durante a busca
	moves := make([]Move, 0, 40) 
	
	p.generatePawns(&moves)
	p.generateKnights(&moves)
	p.generateKings(&moves)
	p.generateSlidingPieces(&moves)
	
	// Nota: Em motores avançados (Pseudo-Legal MoveGen), geramos todos estes lances
	// e apenas verificamos se o Rei ficou em xeque na hora de os jogar na função Apply().
	return moves
}
// ValidMoves devolve apenas os lances estritamente legais (que não deixam o próprio Rei em xeque).
func (p *Position) ValidMoves() []Move {
	pseudo := p.LegalMoves()
	valid := make([]Move, 0, len(pseudo))

	us := p.SideToMove
	for _, mv := range pseudo {
		newP := p.Apply(mv)
		// Se depois de mexer a peça, o nosso Rei não está em xeque, o lance é válido
		if !newP.InCheck(us) {
			valid = append(valid, mv)
		}
	}
	return valid
}

// addPawnMoves verifica se o peão atingiu a última fileira e gera as 4 promoções matemáticas
func addPawnMoves(moves *[]Move, from, to int, isCapture bool) {
	rankTo := to / 8
	if rankTo == 0 || rankTo == 7 { // Chegou ao fim do tabuleiro
		flag := FlagPromoQueen
		if isCapture { flag = FlagPromoCapQueen }
		
		*moves = append(*moves, NewMove(from, to, flag))       // Dama (11 ou 15)
		*moves = append(*moves, NewMove(from, to, flag-1))     // Torre
		*moves = append(*moves, NewMove(from, to, flag-2))     // Bispo
		*moves = append(*moves, NewMove(from, to, flag-3))     // Cavalo (8 ou 12)
	} else {
		flag := FlagQuiet
		if isCapture { flag = FlagCapture }
		*moves = append(*moves, NewMove(from, to, flag))
	}
}