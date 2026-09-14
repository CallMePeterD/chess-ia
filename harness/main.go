// Comando harness: recebe uma FEN e imprime o lance escolhido pela IA.
//
//	go run ./harness -fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
//	go run ./harness -dificuldade media
//	go run ./harness -perft 4
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/CallMePeterD/chess-ia/eval"
	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/search"
)

func main() {
	fen := flag.String("fen", rules.StartFEN, "posição em FEN")
	depth := flag.Int("depth", 2, "profundidade da busca (ignorada se -dificuldade for usada)")
	dificuldade := flag.String("dificuldade", "", "facil | media | dificil")
	perft := flag.Int("perft", 0, "em vez de buscar, roda perft até esta profundidade")
	flag.Parse()

	if err := run(*fen, *depth, *dificuldade, *perft); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(fen string, depth int, dificuldade string, perft int) error {
	pos, err := rules.FromFEN(fen)
	if err != nil {
		return err
	}

	if perft > 0 {
		for d := 1; d <= perft; d++ {
			start := time.Now()
			n := rules.Perft(pos, d)
			fmt.Printf("perft(%d) = %d  (%s)\n", d, n, time.Since(start).Round(time.Millisecond))
		}
		return nil
	}

	if dificuldade != "" {
		if depth, err = search.DepthFor(dificuldade); err != nil {
			return err
		}
	}

	fmt.Printf("posição:    %s\n", pos.FEN())
	fmt.Printf("avaliação:  %d (estática)\n", eval.Evaluate(pos))

	start := time.Now()
	res, err := search.Minimax(pos, depth)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)

	fmt.Printf("lance:      %s\n", res.Move.UCI())
	fmt.Printf("score:      %d (profundidade %d)\n", res.Score, depth)
	fmt.Printf("nós:        %d em %s\n", res.Nodes, elapsed.Round(time.Millisecond))
	return nil
}
