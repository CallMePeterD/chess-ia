package rules

import (
	"math/bits"
	"math/rand"
)

var (
	RookMagics   [64]uint64
	BishopMagics [64]uint64

	RookMasks   [64]Bitboard
	BishopMasks [64]Bitboard

	RookAttacks   [64][4096]Bitboard // 12 bits max = 4096
	BishopAttacks [64][512]Bitboard  // 9 bits max = 512

	RookShifts   [64]int
	BishopShifts [64]int
)

func init() {
	initMagics()
}

// initMagics orquestra a descoberta dos números mágicos
func initMagics() {
	for sq := 0; sq < 64; sq++ {
		// 1. Gera as máscaras (quais casas podem bloquear a visão, ignorando a última borda)
		RookMasks[sq] = maskRook(sq)
		BishopMasks[sq] = maskBishop(sq)

		RookShifts[sq] = 64 - bits.OnesCount64(uint64(RookMasks[sq]))
		BishopShifts[sq] = 64 - bits.OnesCount64(uint64(BishopMasks[sq]))

		// 2. Usa o número mágico já validado (rules/magics_data.go). Só se ele
		// falhar é que caímos na busca aleatória, que custa segundos de arranque.
		RookMagics[sq] = useOrFindMagic(sq, RookShifts[sq], RookMasks[sq], false, rookMagicNumbers[sq])
		BishopMagics[sq] = useOrFindMagic(sq, BishopShifts[sq], BishopMasks[sq], true, bishopMagicNumbers[sq])
	}
}

// useOrFindMagic tenta primeiro o número pré-calculado; se não servir, procura um novo.
func useOrFindMagic(sq, shift int, mask Bitboard, isBishop bool, precomputed uint64) uint64 {
	blockers, attacks := permutations(sq, mask, isBishop)
	if precomputed != 0 && fillTable(sq, shift, isBishop, precomputed, blockers, attacks) {
		return precomputed
	}
	return findMagic(sq, shift, mask, isBishop)
}

// permutations calcula a verdade absoluta: para cada combinação de bloqueadores
// possível na máscara, os ataques reais daquela casa.
func permutations(sq int, mask Bitboard, isBishop bool) (blockers, attacks []Bitboard) {
	total := 1 << bits.OnesCount64(uint64(mask)) // 2^N permutações

	blockers = make([]Bitboard, total)
	attacks = make([]Bitboard, total)
	for i := 0; i < total; i++ {
		blockers[i] = generateBlockers(i, mask)
		if isBishop {
			attacks[i] = bishopAttacksSlow(sq, blockers[i])
		} else {
			attacks[i] = rookAttacksSlow(sq, blockers[i])
		}
	}
	return blockers, attacks
}

// fillTable testa um candidato a número mágico. Se ele mapear todas as
// permutações sem colisão, grava os ataques na tabela global e devolve true.
func fillTable(sq, shift int, isBishop bool, candidate uint64, blockers, attacks []Bitboard) bool {
	used := make([]Bitboard, len(blockers))

	for i := range blockers {
		// A fórmula de Ouro: (Blockers * Magic) >> Shift
		index := (uint64(blockers[i]) * candidate) >> shift

		if used[index] == 0 {
			used[index] = attacks[i]
		} else if used[index] != attacks[i] {
			// Colisão fatal: dois mapas de bloqueadores diferentes caíram no mesmo índice
			// e geraram ataques diferentes. Este candidato é lixo.
			return false
		}
	}

	// Validado: copia os dados para o array global
	for i := range blockers {
		index := (uint64(blockers[i]) * candidate) >> shift
		if isBishop {
			BishopAttacks[sq][index] = attacks[i]
		} else {
			RookAttacks[sq][index] = attacks[i]
		}
	}
	return true
}

// findMagic é a Forja: tenta números aleatórios até encontrar um que mapeie todas as permutações sem colisão
func findMagic(sq int, shift int, mask Bitboard, isBishop bool) uint64 {
	blockers, attacks := permutations(sq, mask, isBishop)

	// Quase sempre resolve em menos de 100.000 iterações.
	for {
		// Estratégia de "bits esparsos": AND bit-a-bit reduz drasticamente os bits 1
		candidate := rand.Uint64() & rand.Uint64() & rand.Uint64()
		if fillTable(sq, shift, isBishop, candidate, blockers, attacks) {
			return candidate
		}
	}
}

// maskRook cria os raios da Torre, parando antes da borda do tabuleiro
func maskRook(sq int) Bitboard {
	var m Bitboard
	r, f := sq/8, sq%8
	for i := r + 1; i <= 6; i++ {
		m.Set(i*8 + f)
	} // Cima
	for i := r - 1; i >= 1; i-- {
		m.Set(i*8 + f)
	} // Baixo
	for i := f + 1; i <= 6; i++ {
		m.Set(r*8 + i)
	} // Direita
	for i := f - 1; i >= 1; i-- {
		m.Set(r*8 + i)
	} // Esquerda
	return m
}

// maskBishop cria as diagonais, parando antes da borda
func maskBishop(sq int) Bitboard {
	var m Bitboard
	r, f := sq/8, sq%8
	for i, j := r+1, f+1; i <= 6 && j <= 6; i, j = i+1, j+1 {
		m.Set(i*8 + j)
	}
	for i, j := r+1, f-1; i <= 6 && j >= 1; i, j = i+1, j-1 {
		m.Set(i*8 + j)
	}
	for i, j := r-1, f+1; i >= 1 && j <= 6; i, j = i-1, j+1 {
		m.Set(i*8 + j)
	}
	for i, j := r-1, f-1; i >= 1 && j >= 1; i, j = i-1, j-1 {
		m.Set(i*8 + j)
	}
	return m
}

// rookAttacksSlow calcula fisicamente onde a Torre bate (avançando casa a casa até encontrar uma peça)
func rookAttacksSlow(sq int, blockers Bitboard) Bitboard {
	var a Bitboard
	r, f := sq/8, sq%8
	for i := r + 1; i <= 7; i++ {
		a.Set(i*8 + f)
		if blockers.Has(i*8 + f) {
			break
		}
	}
	for i := r - 1; i >= 0; i-- {
		a.Set(i*8 + f)
		if blockers.Has(i*8 + f) {
			break
		}
	}
	for i := f + 1; i <= 7; i++ {
		a.Set(r*8 + i)
		if blockers.Has(r*8 + i) {
			break
		}
	}
	for i := f - 1; i >= 0; i-- {
		a.Set(r*8 + i)
		if blockers.Has(r*8 + i) {
			break
		}
	}
	return a
}

// bishopAttacksSlow calcula fisicamente as diagonais
func bishopAttacksSlow(sq int, blockers Bitboard) Bitboard {
	var a Bitboard
	r, f := sq/8, sq%8
	for i, j := r+1, f+1; i <= 7 && j <= 7; i, j = i+1, j+1 {
		a.Set(i*8 + j)
		if blockers.Has(i*8 + j) {
			break
		}
	}
	for i, j := r+1, f-1; i <= 7 && j >= 0; i, j = i+1, j-1 {
		a.Set(i*8 + j)
		if blockers.Has(i*8 + j) {
			break
		}
	}
	for i, j := r-1, f+1; i >= 0 && j <= 7; i, j = i-1, j+1 {
		a.Set(i*8 + j)
		if blockers.Has(i*8 + j) {
			break
		}
	}
	for i, j := r-1, f-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		a.Set(i*8 + j)
		if blockers.Has(i*8 + j) {
			break
		}
	}
	return a
}

// generateBlockers pega no índice iterativo (0 até 2^N) e distribui os bits nos locais da máscara
func generateBlockers(index int, mask Bitboard) Bitboard {
	var b Bitboard
	bitsInMask := bits.OnesCount64(uint64(mask))
	for i := 0; i < bitsInMask; i++ {
		// Encontra onde está o próximo bit 1 da máscara
		sq := bits.TrailingZeros64(uint64(mask))
		mask.Clear(sq)

		// Se o i-ésimo bit do nosso índice estiver ligado, colocamos um bloqueador naquela casa
		if (index & (1 << i)) != 0 {
			b.Set(sq)
		}
	}
	return b
}
