package worker

import (
	"context"
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

	depth, err := search.DepthFor(job.Dificuldade)
	if err != nil {
		return "", 0, err
	}
	multiPV, err := search.MultiPVFor(job.Dificuldade)
	if err != nil {
		return "", 0, err
	}

	// 1. Define o tempo disponível para pensar neste lance.
	timeout := 3 * time.Second
	if job.Dificuldade == "mestre" {
		timeout = 10 * time.Second 
	}

	// 2. Deriva um contexto com cronómetro a partir do contexto principal do Worker
	searchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 3. Executa a busca passando o contexto com prazo de validade
	res, err := search.Minimax(searchCtx, pos, depth, multiPV)
	if err != nil {
		return "", 0, err
	}

	// 4. Devolve exatamente as três variáveis que a interface Solver exige
	return res.Move.UCI(), res.Score, nil
}