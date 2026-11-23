package worker

import (
	"context"
	"time"

	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/search"
)

// Orçamento de tempo padrão, quando o backend não envia timeBudgetMs.
const defaultTimeBudget = 3 * time.Second

// margem reservada do orçamento para serializar e entregar o resultado antes
// de o trabalho expirar.
const budgetMargin = 0.15

// LevelFor escolhe o nível de busca a partir do orçamento de tempo do job.
// O contrato Backend-IA v1 não envia dificuldade: envia timeBudgetMs, e é o
// orçamento que define quão fundo dá para procurar.
func LevelFor(budgetMs int) string {
	switch {
	case budgetMs <= 0:
		return "media"
	case budgetMs < 400:
		return "facil"
	case budgetMs < 1500:
		return "media"
	case budgetMs < 5000:
		return "dificil"
	default:
		return "mestre"
	}
}

// searchBudget converte timeBudgetMs no prazo real da busca, descontando a
// margem de entrega.
func searchBudget(budgetMs int) time.Duration {
	if budgetMs <= 0 {
		return defaultTimeBudget
	}
	d := time.Duration(float64(budgetMs)*(1-budgetMargin)) * time.Millisecond
	if d < 50*time.Millisecond {
		d = 50 * time.Millisecond
	}
	return d
}

// SearchSolver converte o Job numa chamada ao pacote search, respeitando o
// orçamento de tempo enviado pelo backend.
func SearchSolver(ctx context.Context, job Job) (string, int, error) {
	pos, err := rules.FromFEN(job.Position.FEN)
	if err != nil {
		return "", 0, err
	}

	nivel := LevelFor(job.TimeBudgetMs)
	depth, err := search.DepthFor(nivel)
	if err != nil {
		return "", 0, err
	}
	multiPV, err := search.MultiPVFor(nivel)
	if err != nil {
		return "", 0, err
	}

	// O prazo da busca sai do orçamento do job: o backend expira o trabalho
	// se o candidato não chegar a tempo.
	searchCtx, cancel := context.WithTimeout(ctx, searchBudget(job.TimeBudgetMs))
	defer cancel()

	res, err := search.Minimax(searchCtx, pos, depth, multiPV)
	if err != nil {
		return "", 0, err
	}
	return res.Move.UCI(), res.Score, nil
}
