// Package eval contém a função de avaliação estática.
//
// A pontuação é em centipeões (peão = 100) e sempre do ponto de vista das
// brancas: positivo é bom para as brancas, negativo para as pretas.
package eval

import "github.com/CallMePeterD/chess-ia/rules"

// PieceValue é o valor material de cada tipo de peça (v1 — material).
// O rei vale 0: nunca sai do tabuleiro, então não entra na soma.
var PieceValue = [...]int{
	rules.NoPiece: 0,
	rules.Pawn:    100,
	rules.Knight:  300,
	rules.Bishop:  300,
	rules.Rook:    500,
	rules.Queen:   900,
	rules.King:    0,
}

// Evaluate devolve a avaliação da posição (brancas − pretas).
func Evaluate(p rules.Position) int {
	return Material(p)
}

// Material soma o valor das peças, brancas − pretas.
func Material(p rules.Position) int {
	score := 0
	for _, pc := range p.Pieces() {
		v := PieceValue[pc.Type]
		if pc.Color == rules.White {
			score += v
		} else {
			score -= v
		}
	}
	return score
}
