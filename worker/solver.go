package worker

import (
	"context"

	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/search"
)

// SearchSolver é o Solver real: FEN → busca na profundidade da dificuldade.
func SearchSolver(_ context.Context, job Job) (string, int, error) {
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

	// Adicionada a variável multiPV na chamada
	res, err := search.Minimax(pos, depth, multiPV)
	if err != nil {
		return "", 0, err
	}
	return res.Move.UCI(), res.Score, nil
}