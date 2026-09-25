// Package search - Arquivo tt.go
package search

import "github.com/CallMePeterD/chess-ia/rules"

// Flags para saber que tipo de nota foi salva pela Poda Alfa-Beta
const (
	FlagExact      = iota // Encontramos a nota exata
	FlagUpperBound        // Falhou baixo (pior que o Alpha), nota é no máximo X
	FlagLowerBound        // Falhou alto (cortou no Beta), nota é no mínimo X
)

// TTEntry tem 32 bytes de tamanho total (ótimo para alinhamento de memória em 64-bits).
type TTEntry struct {
	Hash  uint64
	Score int
	Depth int
	Flag  int
	Move  rules.Move
}

// TT é a nossa Tabela de Transposição baseada em um array de tamanho fixo.
type TT struct {
	entries []TTEntry
	size    uint64
}

// NewTT aloca a Tabela de Transposição. mbSize define o uso de RAM (ex: 32 = 32 MB).
func NewTT(mbSize int) *TT {
	// 1 MB = 1024 * 1024 bytes. Dividido por 32 bytes por entrada.
	numEntries := (mbSize * 1024 * 1024) / 32
	return &TT{
		entries: make([]TTEntry, numEntries),
		size:    uint64(numEntries),
	}
}

// Probe busca uma posição na tabela.
func (tt *TT) Probe(hash uint64, depth, alpha, beta int) (int, rules.Move, bool) {
	index := hash % tt.size
	entry := tt.entries[index]

	// Colisão de Hash: garante que a posição na memória é realmente a que estamos buscando
	if entry.Hash == hash {
		// Só podemos usar a nota direta se a busca salva for de profundidade igual ou maior
		if entry.Depth >= depth {
			if entry.Flag == FlagExact {
				return entry.Score, entry.Move, true
			}
			if entry.Flag == FlagUpperBound && entry.Score <= alpha {
				return entry.Score, entry.Move, true // Não adianta buscar, será pior que o alpha
			}
			if entry.Flag == FlagLowerBound && entry.Score >= beta {
				return entry.Score, entry.Move, true // Oponente vai cortar de qualquer jeito (Poda Beta)
			}
		}
		// Mesmo se a profundidade for menor, podemos devolver o Move salvo
		// para a Ordenação de Lances testá-lo primeiro!
		return 0, entry.Move, false
	}
	return 0, 0, false
}

// Store salva o cálculo de uma posição na tabela.
func (tt *TT) Store(hash uint64, depth, score, flag int, move rules.Move) {
	index := hash % tt.size
	// Substituição "Always Replace" (sempre sobrescreve). É a estratégia mais rápida e simples em Go.
	tt.entries[index] = TTEntry{
		Hash:  hash,
		Score: score,
		Depth: depth,
		Flag:  flag,
		Move:  move,
	}
}