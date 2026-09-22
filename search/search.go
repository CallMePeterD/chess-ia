// Package search escolhe o lance: minimax de profundidade fixa sobre a
// avaliação de eval, agora otimizado com poda alfa-beta.
package search

import (
	"errors"
	"fmt"

	"github.com/CallMePeterD/chess-ia/eval"
	"github.com/CallMePeterD/chess-ia/rules"
)

// MateScore é a pontuação de um xeque-mate. Mates mais próximos valem mais
// (MateScore − ply), para a IA preferir o mate mais rápido e adiar o que sofre.
const MateScore = 100000

// InfScore representa o infinito (um valor fora do alcance de qualquer pontuação real)
// usado para inicializar os limites alfa e beta.
const InfScore = 1000000

// ErrNoMoves indica que a posição não tem lances legais (partida encerrada).
var ErrNoMoves = errors.New("posição sem lances legais")

// Result é o resultado de uma busca.
type Result struct {
	Move  rules.Move
	Score int   // centipeões, ponto de vista das brancas
	Nodes int64 // posições visitadas
}

// DepthFor mapeia a dificuldade do contrato para a profundidade da busca.
func DepthFor(dificuldade string) (int, error) {
	switch dificuldade {
	case "facil":
		return 1, nil
	case "media":
		return 2, nil
	case "dificil":
		return 3, nil
	}
	return 0, fmt.Errorf("dificuldade desconhecida %q", dificuldade)
}

// Minimax busca até a profundidade dada usando Poda Alfa-Beta e devolve o melhor
// lance para quem joga. Em empate de pontuação fica o primeiro lance encontrado.
func Minimax(p rules.Position, depth int) (Result, error) {
	if depth < 1 {
		return Result{}, fmt.Errorf("profundidade inválida %d", depth)
	}
	moves := p.LegalMoves()
	if len(moves) == 0 {
		return Result{}, ErrNoMoves
	}

	s := &searcher{}
	maximizing := p.SideToMove() == rules.White
	best := Result{Move: moves[0]}

	// Inicializamos os limites da janela de busca
	alpha := -InfScore
	beta := InfScore

	for i, mv := range moves {
		score := s.alphaBeta(p.Apply(mv), depth-1, 1, alpha, beta)
		if i == 0 || (maximizing && score > best.Score) || (!maximizing && score < best.Score) {
			best.Move, best.Score = mv, score
		}

		// Atualizamos a janela na raiz da árvore
		if maximizing {
			if score > alpha {
				alpha = score
			}
		} else {
			if score < beta {
				beta = score
			}
		}
	}
	best.Nodes = s.nodes
	return best, nil
}

type searcher struct {
	nodes int64
}

// alphaBeta devolve a pontuação (visão das brancas) de p explorando mais depth
// plies. O algoritmo corta ramos inúteis da árvore quando beta <= alpha.
func (s *searcher) alphaBeta(p rules.Position, depth, ply, alpha, beta int) int {
	s.nodes++
	moves := p.LegalMoves()
	if len(moves) == 0 {
		if p.Status() == rules.Checkmate {
			// Quem está para jogar levou mate.
			if p.SideToMove() == rules.White {
				return -(MateScore - ply)
			}
			return MateScore - ply
		}
		return 0 // afogamento
	}
	if depth == 0 {
		return eval.Evaluate(p)
	}

	maximizing := p.SideToMove() == rules.White

	if maximizing {
		best := -InfScore
		for _, mv := range moves {
			score := s.alphaBeta(p.Apply(mv), depth-1, ply+1, alpha, beta)
			if score > best {
				best = score
			}
			if best > alpha {
				alpha = best
			}
			if beta <= alpha {
				break // Poda Beta: o oponente já tem uma opção melhor no ramo anterior, corta a busca
			}
		}
		return best
	} else {
		best := InfScore
		for _, mv := range moves {
			score := s.alphaBeta(p.Apply(mv), depth-1, ply+1, alpha, beta)
			if score < best {
				best = score
			}
			if best < beta {
				beta = best
			}
			if beta <= alpha {
				break // Poda Alfa: nós (brancas) já temos uma opção melhor no ramo anterior, corta a busca
			}
		}
		return best
	}
}