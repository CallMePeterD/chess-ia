// Package search escolhe o lance: minimax de profundidade fixa sobre a
// avaliação de eval, com poda alfa-beta, ordenação de lances e busca de quiescência.
package search

import (
	"errors"
	"fmt"
	"sort"
	"math/rand"

	"github.com/CallMePeterD/chess-ia/eval"
	"github.com/CallMePeterD/chess-ia/rules"
)

// MateScore é a pontuação de um xeque-mate. Mates mais próximos valem mais.
const MateScore = 100000

// InfScore representa o infinito para os limites da janela alfa-beta.
const InfScore = 1000000

// ErrNoMoves indica que a posição não tem lances legais.
var ErrNoMoves = errors.New("posição sem lances legais")

// Result é o resultado de uma busca.
type Result struct {
	Move  rules.Move
	Score int
	Nodes int64
}

// DepthFor mapeia a dificuldade para a profundidade da busca.
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

// MultiPVFor mapeia a dificuldade para a quantidade de lances avaliados na raiz (simulando erro).
func MultiPVFor(dificuldade string) (int, error) {
	switch dificuldade {
	case "facil":
		return 3, nil // Avalia as 3 melhores opções e sorteia uma
	case "media":
		return 2, nil // Avalia as 2 melhores e sorteia
	case "dificil":
		return 1, nil // Joga sempre o melhor lance possível
	case "":
		return 1, nil // Fallback se testado sem dificuldade definida
	}
	return 0, fmt.Errorf("dificuldade desconhecida %q", dificuldade)
}

// Adicionamos a TT na struct do searcher
type searcher struct {
	nodes int64
	tt    *TT
}

// Minimax inicia a busca com Poda Alfa-Beta, Iterative Deepening e Multi-PV.
func Minimax(p rules.Position, maxDepth int, multiPV int) (Result, error) {
	if maxDepth < 1 {
		return Result{}, fmt.Errorf("profundidade inválida %d", maxDepth)
	}
	moves := p.ValidMoves()
	if len(moves) == 0 {
		return Result{}, ErrNoMoves
	}

	s := &searcher{tt: NewTT(32)}

	// Garante que não tentamos encontrar mais lances do que os fisicamente possíveis no tabuleiro
	targetPV := multiPV
	if targetPV > len(moves) {
		targetPV = len(moves)
	}

	var finalResults []Result

	for d := 1; d <= maxDepth; d++ {
		finalResults = make([]Result, 0, targetPV)
		excluded := make(map[string]bool)

		// Loop do Multi-PV: corre a busca targetPV vezes
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
				if excluded[mv.UCI()] {
					continue // Ignora o lance se ele já faz parte do nosso "Top N"
				}
				hasValidMove = true

				score := s.alphaBeta(p.Apply(mv), d-1, 1, alpha, beta)

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

			if hasValidMove {
				excluded[bestMove.UCI()] = true
				finalResults = append(finalResults, Result{Move: bestMove, Score: bestScore, Nodes: s.nodes})
			}
		}

		// Otimização: Se o lance primário já garante Mate, aborta o aprofundamento e joga logo
		bestOverall := finalResults[0]
		if bestOverall.Score > MateScore-100 || bestOverall.Score < -MateScore+100 {
			break
		}
	}

	// O segredo do Nerf: Sorteia aleatoriamente um lance entre o Top N
	// EXCEÇÃO: Se o melhor lance absoluto for um Xeque-Mate, 
	// a IA recusa-se a sortear lances inferiores e joga com 100% de precisão.
	if len(finalResults) > 0 {
		if finalResults[0].Score > MateScore-100 || finalResults[0].Score < -MateScore+100 {
			return finalResults[0], nil
		}
	}

	pickedIndex := rand.Intn(len(finalResults))
	return finalResults[pickedIndex], nil
}
	


// alphaBeta agora utiliza o lance da memória (ttMove) para ordenar os caminhos
func (s *searcher) alphaBeta(p rules.Position, depth, ply, alpha, beta int) int {
	originalAlpha := alpha

	hash := p.Hash()
	// Trazemos a variável ttMove de volta à vida
	ttScore, ttMove, ok := s.tt.Probe(hash, depth, alpha, beta)
	if ok {
		return ttScore 
	}

	s.nodes++
	moves := p.ValidMoves()

	if len(moves) == 0 {
		// Se não há lances válidos e o Rei está em xeque, é Xeque-Mate
		if p.InCheck(p.SideToMove) {
			if p.SideToMove == rules.White {
				return -(MateScore - ply)
			}
			return MateScore - ply
		}
		// Se não há lances mas não há xeque, é Afogamento (Stalemate)
		return 0 
	}

	if depth == 0 {
		return s.quiescence(p, alpha, beta)
	}

	// O segredo: Passamos o lance da memória para ser o primeiro a ser testado
	orderMoves(moves, ttMove)

	maximizing := p.SideToMove == rules.White
	bestScore := -InfScore
	if !maximizing { bestScore = InfScore }
	
	var bestMove rules.Move

	for _, mv := range moves {
		var score int
		if maximizing {
			score = s.alphaBeta(p.Apply(mv), depth-1, ply+1, alpha, beta)
			if score > bestScore {
				bestScore = score
				bestMove = mv
			}
			if bestScore > alpha { alpha = bestScore }
			if beta <= alpha { break }
		} else {
			score = s.alphaBeta(p.Apply(mv), depth-1, ply+1, alpha, beta)
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

// quiescence continua a busca além da profundidade limite, avaliando apenas capturas.
func (s *searcher) quiescence(p rules.Position, alpha, beta int) int {
	s.nodes++
	standPat := eval.Evaluate(p) // Avaliação da posição "como está" (Stand Pat)

	maximizing := p.SideToMove == rules.White

	// Tenta cortar a busca imediatamente se a posição estática já for muito boa para o jogador
	if maximizing {
		if standPat >= beta {
			return beta
		}
		if standPat > alpha {
			alpha = standPat
		}
	} else {
		if standPat <= alpha {
			return alpha
		}
		if standPat < beta {
			beta = standPat
		}
	}

	moves := p.ValidMoves()
	orderMoves(moves, 0)

	if maximizing {
		best := standPat
		for _, mv := range moves {
			// Na quiescência, só nos importamos com lances instáveis (capturas ou promoções)
			if !mv.IsCapture() && !mv.IsPromotion() {
				continue
			}

			score := s.quiescence(p.Apply(mv), alpha, beta)
			if score > best {
				best = score
			}
			if best > alpha {
				alpha = best
			}
			if beta <= alpha {
				break
			}
		}
		return best
	} else {
		best := standPat
		for _, mv := range moves {
			if !mv.IsCapture() && !mv.IsPromotion() {
				continue
			}

			score := s.quiescence(p.Apply(mv), alpha, beta)
			if score < best {
				best = score
			}
			if best < beta {
				beta = best
			}
			if beta <= alpha {
				break
			}
		}
		return best
	}
}

// orderMoves agora recebe o lance da memória para dar prioridade máxima
func orderMoves(moves []rules.Move, hashMove rules.Move) {
	sort.Slice(moves, func(i, j int) bool {
		return scoreMove(moves[i], hashMove) > scoreMove(moves[j], hashMove)
	})
}

// scoreMove atribui 10.000 pontos ao lance da Hash, garantindo que seja o índice 0
func scoreMove(mv rules.Move, hashMove rules.Move) int {
	if hashMove.UCI() != "" && mv.UCI() == hashMove.UCI() {
		return 10000 // Prioridade máxima absoluta: já sabemos que este lance é forte
	}
	score := 0
	if mv.IsCapture() {
		score += 10
	}
	if mv.IsPromotion() {
		score += 5
	}
	return score
}