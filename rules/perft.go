package rules

// Perft viaja pela árvore de lances e conta os nós finais para testar a exatidão e velocidade do MoveGen.
func Perft(p Position, depth int) int {
	if depth == 0 {
		return 1
	}

	moves := p.ValidMoves()
	
	// Otimização: se for a última profundidade, não precisamos aplicar os lances, 
	// basta contar quantos lances legais existem
	if depth == 1 {
		return len(moves)
	}

	nodes := 0
	for _, m := range moves {
		newP := p.Apply(m)
		nodes += Perft(newP, depth-1)
	}

	return nodes
}