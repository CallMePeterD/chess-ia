// Package eval contém a função de avaliação estática.
//
// A pontuação é em centipeões (peão = 100) e sempre do ponto de vista das
// brancas: positivo é bom para as brancas, negativo para as pretas.
package eval

import (
	"math/bits"

	"github.com/CallMePeterD/chess-ia/rules"
)

// PieceValue é o valor material de cada tipo de peça.
var PieceValue = [...]int{
	rules.NoPiece: 0,
	rules.Pawn:    100,
	rules.Knight:  300,
	rules.Bishop:  300,
	rules.Rook:    500,
	rules.Queen:   900,
	rules.King:    0,
}

// Matrizes de posicionamento (PST). Mapeiam as 64 casas (0 a 63).
// Valores ajustados para incentivar domínio do centro e desenvolvimento seguro.
// A visão visual destas tabelas assume que o índice 0 (A1) está no topo esquerdo.
// Como o Xadrez padrão trata A1 como canto inferior, usamos índices diretos.

var pstPawn = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, -20, -20, 10, 10, 5,
	5, -5, -10, 0, 0, -10, -5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, 5, 10, 25, 25, 10, 5, 5,
	10, 10, 20, 30, 30, 20, 10, 10,
	50, 50, 50, 50, 50, 50, 50, 50,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var pstKnight = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var pstBishop = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var pstRook = [64]int{
	0, 0, 0, 5, 5, 0, 0, 0,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	5, 10, 10, 10, 10, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var pstQueen = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-5, 0, 5, 5, 5, 5, 0, -5,
	0, 0, 5, 5, 5, 5, 0, -5,
	-10, 5, 5, 5, 5, 5, 0, -10,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var pstKing = [64]int{
	20, 30, 10, 0, 0, 10, 30, 20,
	20, 20, 0, 0, 0, 0, 20, 20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
}

// pstMapping conecta o tipo da peça à sua tabela correspondente
var pstMapping = [...]*[64]int{
	rules.NoPiece: nil,
	rules.Pawn:    &pstPawn,
	rules.Knight:  &pstKnight,
	rules.Bishop:  &pstBishop,
	rules.Rook:    &pstRook,
	rules.Queen:   &pstQueen,
	rules.King:    &pstKing,
}

// Material soma apenas o valor bruto das peças, brancas − pretas.
// Com Bitboards, basta contar os bits '1' ativos para cada tipo de peça.
func Material(p rules.Position) int {
	score := 0
	for pt := rules.Pawn; pt <= rules.King; pt++ {
		wCount := bits.OnesCount64(uint64(p.Pieces[pt] & p.Colors[rules.White]))
		bCount := bits.OnesCount64(uint64(p.Pieces[pt] & p.Colors[rules.Black]))
		
		score += (wCount - bCount) * PieceValue[pt]
	}
	return score
}

// Evaluate devolve a avaliação combinada (Material + Posicionamento).
func Evaluate(p rules.Position) int {
	score := 0

	for pt := rules.Pawn; pt <= rules.King; pt++ {
		val := PieceValue[pt]
		table := pstMapping[pt]

		// Avalia as peças Brancas
		whitePieces := p.Pieces[pt] & p.Colors[rules.White]
		for whitePieces != 0 {
			sq := bits.TrailingZeros64(uint64(whitePieces)) // Acha a casa exata
			
			score += val
			if table != nil {
				score += table[sq]
			}
			
			whitePieces &= whitePieces - 1 // Desliga o bit processado
		}

		// Avalia as peças Pretas
		blackPieces := p.Pieces[pt] & p.Colors[rules.Black]
		for blackPieces != 0 {
			sq := bits.TrailingZeros64(uint64(blackPieces)) 
			
			score -= val
			if table != nil {
				score -= table[sq^56] // Espelha o índice da casa para as Pretas
			}
			
			blackPieces &= blackPieces - 1 
		}
	}

	// A pontuação continua sempre do ponto de vista das Brancas
	return score
}