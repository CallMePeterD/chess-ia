// Package search escolhe o lance: minimax de profundidade fixa sobre a
// avaliação de eval, com poda alfa-beta, ordenação de lances e busca de quiescência.
package search

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"

	"github.com/CallMePeterD/chess-ia/book"
	"github.com/CallMePeterD/chess-ia/eval"
	"github.com/CallMePeterD/chess-ia/rules"
)

var OpeningBook *book.Book

const MateScore = 100000
const InfScore = 1000000

var ErrNoMoves = errors.New("posição sem lances legais")

type Result struct {
	Move  rules.Move
	Score int
	Nodes int64
}

// DepthFor mapeia a dificuldade para a profundidade máxima.
// "mestre" usa profundidade 100 (efetivamente infinita, será travada pelo tempo).
func DepthFor(dificuldade string) (int, error) {
	switch dificuldade {
	case "facil":
		return 1, nil
	case "media":
		return 2, nil
	case "dificil":
		return 3, nil
	case "mestre":
		return 100, nil
	}
	return 0, fmt.Errorf("dificuldade desconhecida %q", dificuldade)
}

func MultiPVFor(dificuldade string) (int, error) {
	switch dificuldade {
	case "facil":
		return 3, nil
	case "media":
		return 2, nil
	case "dificil", "mestre":
		return 1, nil
	case "":
		return 1, nil
	}
	return 0, fmt.Errorf("dificuldade desconhecida %q", dificuldade)
}

type searcher struct {
	nodes   int64
	tt      *TT
	aborted bool // Flag de interrupção ativada pelo limite de tempo
}

// Minimax agora recebe um context.Context para permitir interrupção da busca (Time Management)
func Minimax(ctx context.Context, p rules.Position, maxDepth int, multiPV int) (Result, error) {
	if maxDepth < 1 {
		return Result{}, fmt.Errorf("profundidade inválida %d", maxDepth)
	}
	moves := p.ValidMoves()
	if len(moves) == 0 {
		return Result{}, ErrNoMoves
	}

	s := &searcher{tt: NewTT(32)}

	targetPV := multiPV
	if targetPV > len(moves) {
		targetPV = len(moves)
	}

	var finalResults []Result
	var lastCompletedResults []Result // Guarda a última profundidade que terminou 100%

	if OpeningBook != nil {
		if uci, err := OpeningBook.GetMove(p); err == nil {
			for _, m := range moves {
				if m.UCI() == uci {
					return Result{Move: m, Score: 0, Nodes: 0}, nil
				}
			}
		}
	}

	// Iterative Deepening
	for d := 1; d <= maxDepth; d++ {
		s.aborted = false
		finalResults = make([]Result, 0, targetPV)
		excluded := make(map[string]bool)

		for n := 0; n < targetPV; n++ {
			var hashMove rules.Move
			if _, ttMove, ok := s.tt.Probe(p.Hash(), 0, -InfScore, InfScore); ok || ttMove.UCI() != "" {
				hashMove = ttMove
			}

			orderMoves(moves, hashMove)

			maximizing := p.SideToMove == rules.White
			bestScore := -InfScore
			if !maximizing { bestScore = InfScore }
			
			var bestMove rules.Move
			hasValidMove := false
			alpha := -InfScore
			beta := InfScore

			for _, mv := range moves {
				if excluded[mv.UCI()] { continue }
				hasValidMove = true

				score := s.alphaBeta(ctx, p.Apply(mv), d-1, 1, alpha, beta)
				
				// Se o tempo acabou a meio deste lance, a avaliação não é fiável. Abortar!
				if s.aborted {
					break
				}

				if maximizing {
					if score > bestScore || bestScore == -InfScore {
						bestScore = score
						bestMove = mv
					}
					if score > alpha { alpha = score }
				} else {
					if score < bestScore || bestScore == InfScore {
						bestScore = score
						bestMove = mv
					}
					if score < beta { beta = score }
				}
			}

			if s.aborted { break }

			if hasValidMove {
				excluded[bestMove.UCI()] = true
				finalResults = append(finalResults, Result{Move: bestMove, Score: bestScore, Nodes: s.nodes})
			}
		}

		// Gestão de Tempo: Se o relógio interrompeu esta profundidade a meio, 
		// deitamos fora os cálculos parciais e usamos os resultados da profundidade anterior.
		if s.aborted {
			if len(lastCompletedResults) > 0 {
				finalResults = lastCompletedResults
			}
			break
		}

		lastCompletedResults = finalResults

		bestOverall := finalResults[0]
		if bestOverall.Score > MateScore-100 || bestOverall.Score < -MateScore+100 {
			break
		}
	}

	if len(finalResults) > 0 {
		if finalResults[0].Score > MateScore-100 || finalResults[0].Score < -MateScore+100 {
			return finalResults[0], nil
		}
	} else {
		// Fallback de segurança absoluto (joga o primeiro lance legal)
		return Result{Move: moves[0], Score: 0, Nodes: s.nodes}, nil 
	}

	pickedIndex := rand.Intn(len(finalResults))
	return finalResults[pickedIndex], nil
}

func (s *searcher) alphaBeta(ctx context.Context, p rules.Position, depth, ply, alpha, beta int) int {
	// A cada 2048 nós verificamos se o tempo acabou (evita penalizar a performance lendo o relógio a toda a hora)
	if s.nodes&2047 == 0 {
		if ctx.Err() != nil {
			s.aborted = true
			return 0
		}
	}
	if s.aborted { return 0 }

	originalAlpha := alpha
	hash := p.Hash()
	ttScore, ttMove, ok := s.tt.Probe(hash, depth, alpha, beta)
	if ok {
		return ttScore 
	}

	s.nodes++
	moves := p.ValidMoves()

	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			if p.SideToMove == rules.White { return -(MateScore - ply) }
			return MateScore - ply
		}
		return 0 
	}

	if depth == 0 {
		return s.quiescence(ctx, p, alpha, beta)
	}

	orderMoves(moves, ttMove)

	maximizing := p.SideToMove == rules.White
	bestScore := -InfScore
	if !maximizing { bestScore = InfScore }
	
	var bestMove rules.Move

	for _, mv := range moves {
		var score int
		if maximizing {
			score = s.alphaBeta(ctx, p.Apply(mv), depth-1, ply+1, alpha, beta)
			if s.aborted { return 0 }
			if score > bestScore {
				bestScore = score
				bestMove = mv
			}
			if bestScore > alpha { alpha = bestScore }
			if beta <= alpha { break }
		} else {
			score = s.alphaBeta(ctx, p.Apply(mv), depth-1, ply+1, alpha, beta)
			if s.aborted { return 0 }
			if score < bestScore {
				bestScore = score
				bestMove = mv
			}
			if bestScore < beta { beta = bestScore }
			if beta <= alpha { break }
		}
	}

	flag := FlagExact
	if bestScore <= originalAlpha {
		flag = FlagUpperBound
	} else if bestScore >= beta {
		flag = FlagLowerBound
	}
	s.tt.Store(hash, depth, bestScore, flag, bestMove)

	return bestScore
}

func (s *searcher) quiescence(ctx context.Context, p rules.Position, alpha, beta int) int {
	if s.nodes&2047 == 0 {
		if ctx.Err() != nil {
			s.aborted = true
			return 0
		}
	}
	if s.aborted { return 0 }

	s.nodes++
	standPat := eval.Evaluate(p)

	maximizing := p.SideToMove == rules.White

	if maximizing {
		if standPat >= beta { return beta }
		if standPat > alpha { alpha = standPat }
	} else {
		if standPat <= alpha { return alpha }
		if standPat < beta { beta = standPat }
	}

	moves := p.ValidMoves()
	orderMoves(moves, 0)

	if maximizing {
		best := standPat
		for _, mv := range moves {
			if !mv.IsCapture() && !mv.IsPromotion() { continue }
			score := s.quiescence(ctx, p.Apply(mv), alpha, beta)
			if s.aborted { return 0 }
			if score > best { best = score }
			if best > alpha { alpha = best }
			if beta <= alpha { break }
		}
		return best
	} else {
		best := standPat
		for _, mv := range moves {
			if !mv.IsCapture() && !mv.IsPromotion() { continue }
			score := s.quiescence(ctx, p.Apply(mv), alpha, beta)
			if s.aborted { return 0 }
			if score < best { best = score }
			if best < beta { beta = best }
			if beta <= alpha { break }
		}
		return best
	}
}

func orderMoves(moves []rules.Move, hashMove rules.Move) {
	sort.Slice(moves, func(i, j int) bool {
		return scoreMove(moves[i], hashMove) > scoreMove(moves[j], hashMove)
	})
}

func scoreMove(mv rules.Move, hashMove rules.Move) int {
	if hashMove.UCI() != "" && mv.UCI() == hashMove.UCI() {
		return 10000 
	}
	score := 0
	if mv.IsCapture() { score += 10 }
	if mv.IsPromotion() { score += 5 }
	return score
}