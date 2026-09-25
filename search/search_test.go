package search

import (
	"errors"
	"testing"

	"github.com/CallMePeterD/chess-ia/rules"
)

func mustFEN(t *testing.T, fen string) rules.Position {
	t.Helper()
	p, err := rules.FromFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFindsBestMove(t *testing.T) {
	cases := []struct {
		name  string
		fen   string
		depth int
		want  string
	}{
		{"brancas capturam dama grátis", "4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1", 1, "e4d5"},
		{"brancas: mate no corredor", "6k1/5ppp/8/8/8/8/8/1R4K1 w - - 0 1", 2, "b1b8"},
		{"pretas: mate no corredor", "r5k1/8/8/8/8/8/5PPP/6K1 b - - 0 1", 2, "a8a1"},
	}
	for _, tc := range cases {
		// Adicionado o parâmetro multiPV = 1
		res, err := Minimax(mustFEN(t, tc.fen), tc.depth, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := res.Move.UCI(); got != tc.want {
			t.Errorf("%s: jogou %s (score %d), esperado %s", tc.name, got, res.Score, tc.want)
		}
	}
}

// Com a Busca de Quiescência ativa e os Bitboards perfeitos, a IA já consegue
// "ver" a recaptura mesmo na profundidade 1, evitando entregar a dama ingenuamente.
func TestQuiescenceSeesRecapture(t *testing.T) {
	pos := mustFEN(t, "4k3/8/2p5/3p4/8/8/8/3QK3 w - - 0 1")

	// Usamos profundidade 1 e MultiPV 1 (nível difícil)
	shallow, err := Minimax(pos, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	
	// A IA deve escolher um lance seguro (ex: Rei move) em vez de d1d5
	if shallow.Move.UCI() == "d1d5" {
		t.Fatalf("profundidade 1 não deveria ser ingênua com Quiescência ativada. Entregou a dama com d1d5")
	}
}

func TestMateScoreSign(t *testing.T) {
	// Adicionado o parâmetro multiPV = 1
	res, err := Minimax(mustFEN(t, "r5k1/8/8/8/8/8/5PPP/6K1 b - - 0 1"), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Score > -MateScore+10 {
		t.Errorf("mate das pretas deveria ter score muito negativo, veio %d", res.Score)
	}
}

func TestNoMoves(t *testing.T) {
	// Adicionado o parâmetro multiPV = 1
	_, err := Minimax(mustFEN(t, "R5k1/5ppp/8/8/8/8/8/6K1 b - - 0 1"), 2, 1)
	if !errors.Is(err, ErrNoMoves) {
		t.Errorf("esperava ErrNoMoves, veio %v", err)
	}
}

func TestDepthFor(t *testing.T) {
	for _, d := range []string{"facil", "media", "dificil"} {
		if _, err := DepthFor(d); err != nil {
			t.Error(err)
		}
	}
	if _, err := DepthFor("impossivel"); err == nil {
		t.Error("esperava erro para dificuldade desconhecida")
	}
}

func TestMultiPVFor(t *testing.T) {
	for _, d := range []string{"facil", "media", "dificil", ""} {
		if _, err := MultiPVFor(d); err != nil {
			t.Error(err)
		}
	}
	if _, err := MultiPVFor("impossivel"); err == nil {
		t.Error("esperava erro para dificuldade desconhecida")
	}
}