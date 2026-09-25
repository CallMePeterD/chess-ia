package rules

import "math/bits"

// IsAttacked verifica se uma determinada casa está a ser atacada por peças de uma cor específica.
func (p *Position) IsAttacked(sq int, attackerColor Color) bool {
	occupied := p.Occupied()
	attackers := p.Colors[attackerColor]

	// 1. Deteção de Cavalos (O Rei finge ser um Cavalo e vê se acerta nalgum Cavalo inimigo)
	if (KnightAttacks[sq] & attackers & p.Pieces[Knight]) != 0 {
		return true
	}

	// 2. Deteção de Reis (Impede que dois Reis fiquem colados lado a lado)
	if (KingAttacks[sq] & attackers & p.Pieces[King]) != 0 {
		return true
	}

	// 3. Deteção de Bispos e Damas (O Rei dispara raios diagonais)
	if (getBishopAttacks(sq, occupied) & attackers & (p.Pieces[Bishop] | p.Pieces[Queen])) != 0 {
		return true
	}

	// 4. Deteção de Torres e Damas (O Rei dispara raios retos)
	if (getRookAttacks(sq, occupied) & attackers & (p.Pieces[Rook] | p.Pieces[Queen])) != 0 {
		return true
	}

	// 5. Deteção de Peões
	bbSq := Bitboard(1 << sq)
	if attackerColor == White {
		// Os peões brancos atacam "para cima" (índices maiores). 
		// Logo, para o Rei saber se está em perigo, ele olha "para baixo" (>> 7 e >> 9).
		attacksFromKing := ((bbSq >> 7) &^ FileA) | ((bbSq >> 9) &^ FileH)
		if (attacksFromKing & attackers & p.Pieces[Pawn]) != 0 {
			return true
		}
	} else {
		// Os peões pretos atacam "para baixo". O Rei olha "para cima" (<< 7 e << 9).
		attacksFromKing := ((bbSq << 7) &^ FileH) | ((bbSq << 9) &^ FileA)
		if (attacksFromKing & attackers & p.Pieces[Pawn]) != 0 {
			return true
		}
	}

	return false
}

// InCheck é a função principal que a engine chama. Localiza o Rei e verifica o perigo.
func (p *Position) InCheck(c Color) bool {
	kingBB := p.Pieces[King] & p.Colors[c]
	if kingBB == 0 {
		return false // Prevenção de segurança (fail-safe) se o tabuleiro estiver vazio
	}
	
	// Acha a coordenada do Rei num microsegundo
	sq := bits.TrailingZeros64(uint64(kingBB))
	enemyColor := 1 ^ c
	
	return p.IsAttacked(sq, enemyColor)
}