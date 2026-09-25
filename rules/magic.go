package rules

import (
	"math/bits"
	"math/rand"
)

var (
	RookMagics   [64]uint64
	BishopMagics [64]uint64
	
	RookMasks    [64]Bitboard
	BishopMasks  [64]Bitboard
	
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

		// 2. Descobre o Número Mágico para esta casa
		RookMagics[sq] = findMagic(sq, RookShifts[sq], RookMasks[sq], false)
		BishopMagics[sq] = findMagic(sq, BishopShifts[sq], BishopMasks[sq], true)
	}
}

// findMagic é a Forja: tenta números aleatórios até encontrar um que mapeie todas as permutações sem colisão
func findMagic(sq int, shift int, mask Bitboard, isBishop bool) uint64 {
	// Conta quantos bits a máscara tem (N)
	maskBits := bits.OnesCount64(uint64(mask))
	permutations := 1 << maskBits // 2^N permutações

	blockerConfigs := make([]Bitboard, permutations)
	attackConfigs := make([]Bitboard, permutations)

	// Calcula a verdade absoluta (os ataques reais para cada permutação possível)
	for i := 0; i < permutations; i++ {
		blockerConfigs[i] = generateBlockers(i, mask)
		if isBishop {
			attackConfigs[i] = bishopAttacksSlow(sq, blockerConfigs[i])
		} else {
			attackConfigs[i] = rookAttacksSlow(sq, blockerConfigs[i])
		}
	}

	// Tenta descobrir o mágico. Quase sempre resolve em menos de 100.000 iterações.
	for {
		// Estratégia de "bits esparsos": AND bit-a-bit reduz drasticamente os bits 1
		candidate := rand.Uint64() & rand.Uint64() & rand.Uint64()
		
		// Array temporário para testar colisões
		used := make([]Bitboard, permutations)
		success := true

		for i := 0; i < permutations; i++ {
			// A fórmula de Ouro: (Blockers * Magic) >> Shift
			index := (uint64(blockerConfigs[i]) * candidate) >> shift
			
			if used[index] == 0 {
				used[index] = attackConfigs[i]
			} else if used[index] != attackConfigs[i] {
				// Colisão fatal: dois mapas de bloqueadores diferentes caíram no mesmo índice 
				// e geraram ataques diferentes. Este candidato é lixo.
				success = false
				break
			}
		}

		if success {
			// Encontramos! Copia os dados validados para o array global
			for i := 0; i < permutations; i++ {
				index := (uint64(blockerConfigs[i]) * candidate) >> shift
				if isBishop {
					BishopAttacks[sq][index] = attackConfigs[i]
				} else {
					RookAttacks[sq][index] = attackConfigs[i]
				}
			}
			return candidate
		}
	}
}

// maskRook cria os raios da Torre, parando antes da borda do tabuleiro
func maskRook(sq int) Bitboard {
	var m Bitboard
	r, f := sq/8, sq%8
	for i := r + 1; i <= 6; i++ { m.Set(i*8 + f) } // Cima
	for i := r - 1; i >= 1; i-- { m.Set(i*8 + f) } // Baixo
	for i := f + 1; i <= 6; i++ { m.Set(r*8 + i) } // Direita
	for i := f - 1; i >= 1; i-- { m.Set(r*8 + i) } // Esquerda
	return m
}

// maskBishop cria as diagonais, parando antes da borda
func maskBishop(sq int) Bitboard {
	var m Bitboard
	r, f := sq/8, sq%8
	for i, j := r+1, f+1; i <= 6 && j <= 6; i, j = i+1, j+1 { m.Set(i*8 + j) }
	for i, j := r+1, f-1; i <= 6 && j >= 1; i, j = i+1, j-1 { m.Set(i*8 + j) }
	for i, j := r-1, f+1; i >= 1 && j <= 6; i, j = i-1, j+1 { m.Set(i*8 + j) }
	for i, j := r-1, f-1; i >= 1 && j >= 1; i, j = i-1, j-1 { m.Set(i*8 + j) }
	return m
}

// rookAttacksSlow calcula fisicamente onde a Torre bate (avançando casa a casa até encontrar uma peça)
func rookAttacksSlow(sq int, blockers Bitboard) Bitboard {
	var a Bitboard
	r, f := sq/8, sq%8
	for i := r + 1; i <= 7; i++ { a.Set(i*8 + f); if blockers.Has(i*8 + f) { break } }
	for i := r - 1; i >= 0; i-- { a.Set(i*8 + f); if blockers.Has(i*8 + f) { break } }
	for i := f + 1; i <= 7; i++ { a.Set(r*8 + i); if blockers.Has(r*8 + i) { break } }
	for i := f - 1; i >= 0; i-- { a.Set(r*8 + i); if blockers.Has(r*8 + i) { break } }
	return a
}

// bishopAttacksSlow calcula fisicamente as diagonais
func bishopAttacksSlow(sq int, blockers Bitboard) Bitboard {
	var a Bitboard
	r, f := sq/8, sq%8
	for i, j := r+1, f+1; i <= 7 && j <= 7; i, j = i+1, j+1 { a.Set(i*8 + j); if blockers.Has(i*8 + j) { break } }
	for i, j := r+1, f-1; i <= 7 && j >= 0; i, j = i+1, j-1 { a.Set(i*8 + j); if blockers.Has(i*8 + j) { break } }
	for i, j := r-1, f+1; i >= 0 && j <= 7; i, j = i-1, j+1 { a.Set(i*8 + j); if blockers.Has(i*8 + j) { break } }
	for i, j := r-1, f-1; i >= 0 && j >= 0; i, j = i-1, j-1 { a.Set(i*8 + j); if blockers.Has(i*8 + j) { break } }
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