package search

import (
	"context"
	"testing"
)

// EAP 4.6.2 — validação com posições de lance conhecido.
//
// Cada caso tem uma única resposta certa, verificável por quem entende de
// xadrez sem precisar ler o código da busca. São o contrapeso dos testes de
// unidade: medem se a IA joga bem, não se as funções fazem o que dizem.

// puzzle é uma posição com resposta conhecida.
type puzzle struct {
	nome  string
	fen   string
	lance string // UCI esperado
	depth int
}

func resolve(t *testing.T, pz puzzle) Result {
	t.Helper()
	res, err := Minimax(context.Background(), mustFEN(t, pz.fen), pz.depth, 1)
	if err != nil {
		t.Fatalf("%s: busca falhou: %v", pz.nome, err)
	}
	return res
}

// Mates forçados: a IA tem de achar o lance e reconhecer que é mate.
func TestPuzzlesDeMate(t *testing.T) {
	puzzles := []puzzle{
		{"mate do corredor com torre", "6k1/5ppp/8/8/8/8/5PPP/R5K1 w - - 0 1", "a1a8", 3},
		{"mate do corredor com dama", "6k1/5ppp/8/8/8/8/5PPP/3QK3 w - - 0 1", "d1d8", 3},
		{"mate do pastor", "r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 4 4", "f3f7", 3},
		{"mate sufocado com o cavalo", "6rk/6pp/8/6N1/8/8/8/6K1 w - - 0 1", "g5f7", 3},
		{"mate em dois com torre", "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 1", "e1e8", 4},
	}
	for _, pz := range puzzles {
		res := resolve(t, pz)
		if got := res.Move.UCI(); got != pz.lance {
			t.Errorf("%s: jogou %s, esperado %s (score %d)", pz.nome, got, pz.lance, res.Score)
		}
		if res.Score < MateScore-100 {
			t.Errorf("%s: achou o lance mas não reconheceu o mate (score %d)", pz.nome, res.Score)
		}
	}
}

// Duas torres dão mate em escada por mais de um caminho; aqui o que importa é
// que a IA veja o mate, não qual das torres ela escolhe primeiro.
func TestPuzzleMateEmEscadaComDuasTorres(t *testing.T) {
	res := resolve(t, puzzle{"escada", "6k1/8/8/8/8/8/R7/1R4K1 w - - 0 1", "", 4})

	if res.Score < MateScore-100 {
		t.Errorf("deveria ver o mate em escada, jogou %s com score %d", res.Move.UCI(), res.Score)
	}
}

// Táticas sem mate: ganhar material à força.
func TestPuzzlesDeTatica(t *testing.T) {
	puzzles := []puzzle{
		{"captura a dama pendurada", "4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1", "e4d5", 2},
		{"garfo de cavalo no rei e na dama", "4k3/7q/8/3N4/8/8/8/4K3 w - - 0 1", "d5f6", 4},
		{"captura o cavalo com a dama", "4k3/8/8/8/8/5n2/8/3QK3 w - - 0 1", "d1f3", 4},
		{"promove o peão para dama", "8/4P3/8/8/8/8/8/4K2k w - - 0 1", "e7e8q", 4},
		{"pretas capturam o peão", "4k3/8/8/4q3/3P4/8/8/4K3 b - - 0 1", "e5d4", 3},
	}
	for _, pz := range puzzles {
		res := resolve(t, pz)
		if got := res.Move.UCI(); got != pz.lance {
			t.Errorf("%s: jogou %s, esperado %s (score %d)", pz.nome, got, pz.lance, res.Score)
		}
	}
}

// A avaliação é do ponto de vista das brancas, mas a busca devolve a nota do
// lado que joga: vantagem de quem move é sempre positiva.
func TestPuzzleScoreEhDoLadoQueJoga(t *testing.T) {
	brancas := resolve(t, puzzle{"brancas ganham a dama", "4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1", "e4d5", 2})
	pretas := resolve(t, puzzle{"pretas ganham a dama", "4k3/8/8/4q3/3P4/8/8/4K3 b - - 0 1", "e5d4", 3})

	if brancas.Score <= 0 {
		t.Errorf("brancas ganhando material deveriam ter score positivo, deu %d", brancas.Score)
	}
	if pretas.Score >= 0 {
		t.Errorf("pretas ganhando material deveriam ter score negativo nesta convenção, deu %d", pretas.Score)
	}
}

// Profundidade maior não pode piorar a escolha num mate forçado.
func TestPuzzleMateSeMantemComMaisProfundidade(t *testing.T) {
	fen := "6k1/5ppp/8/8/8/8/5PPP/R5K1 w - - 0 1"
	for _, depth := range []int{3, 4, 5} {
		res := resolve(t, puzzle{"mate do corredor", fen, "a1a8", depth})
		if got := res.Move.UCI(); got != "a1a8" {
			t.Errorf("profundidade %d: jogou %s, esperado a1a8", depth, got)
		}
	}
}
