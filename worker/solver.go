package worker

import (
	"context"
	"math/bits"
	"time"

	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/search"
)

// SearchSolver converte o Job numa chamada ao pacote search,
// aplicando os limites de tempo (Time Management).
func SearchSolver(ctx context.Context, job Job) (string, int, error) {
	pos, err := rules.FromFEN(job.FEN)
	if err != nil {
		return "", 0, err
	}

	//1. SYZYGY
	// Somamos os bits de todas as peças Brancas e Pretas
	numPecas := bits.OnesCount64(uint64(pos.Colors[rules.White] | pos.Colors[rules.Black]))
	
	if numPecas <= 7 {
		if bestMove, ok := search.ProbeCloudTablebase(job.FEN); ok {
			// Devolvemos um Score altíssimo para o backend perceber que é lance de Tablebase
			return bestMove, search.MateScore - 100, nil
		}
	}
	// ------------------------------------------------

	// 2. Transição normal para a IA de meio-jogo
	depth, err := search.DepthFor(job.Dificuldade)
	if err != nil {
		return "", 0, err
	}
	multiPV, err := search.MultiPVFor(job.Dificuldade)
	if err != nil {
		return "", 0, err
	}

	// 3. Gestão de Tempo
	timeout := 3 * time.Second
	if job.Dificuldade == "mestre" {
		timeout = 10 * time.Second 
	}

	searchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 4. Executa o motor Minimax normal
	res, err := search.Minimax(searchCtx, pos, depth, multiPV)
	if err != nil {
		return "", 0, err
	}

	return res.Move.UCI(), res.Score, nil
}