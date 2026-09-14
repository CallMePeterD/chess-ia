package rules

import "testing"

// Valores de referência: https://www.chessprogramming.org/Perft_Results
var perftCases = []struct {
	name  string
	fen   string
	nodes []int64 // nodes[i] = perft(i+1)
}{
	{"inicial", StartFEN, []int64{20, 400, 8902, 197281}},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", []int64{48, 2039, 97862}},
	{"pos3", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", []int64{14, 191, 2812, 43238}},
	{"pos4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", []int64{6, 264, 9467}},
	{"pos5", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", []int64{44, 1486, 62379}},
}

func TestPerft(t *testing.T) {
	for _, tc := range perftCases {
		pos, err := FromFEN(tc.fen)
		if err != nil {
			t.Fatal(err)
		}
		for i, want := range tc.nodes {
			depth := i + 1
			if testing.Short() && want > 10000 {
				continue
			}
			if got := Perft(pos, depth); got != want {
				t.Errorf("%s perft(%d) = %d, esperado %d", tc.name, depth, got, want)
			}
		}
	}
}

func TestApplyIsImmutable(t *testing.T) {
	pos := Start()
	mv, err := pos.ParseUCI("e2e4")
	if err != nil {
		t.Fatal(err)
	}
	next := pos.Apply(mv)
	if pos.FEN() != StartFEN {
		t.Errorf("posição original mudou: %s", pos.FEN())
	}
	if next.SideToMove() != Black {
		t.Errorf("depois de e2e4 deveria ser a vez das pretas")
	}
}

func TestPromotionUCI(t *testing.T) {
	pos, err := FromFEN("8/4P3/8/8/8/8/k7/7K w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	mv, err := pos.ParseUCI("e7e8q")
	if err != nil {
		t.Fatal(err)
	}
	if !mv.IsPromotion() {
		t.Errorf("e7e8q deveria ser promoção")
	}
}

func TestStatus(t *testing.T) {
	cases := []struct {
		fen  string
		want Status
	}{
		{StartFEN, Ongoing},
		{"R5k1/5ppp/8/8/8/8/8/6K1 b - - 0 1", Checkmate},
		{"7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", Stalemate},
	}
	for _, tc := range cases {
		pos, err := FromFEN(tc.fen)
		if err != nil {
			t.Fatal(err)
		}
		if got := pos.Status(); got != tc.want {
			t.Errorf("%s: status %d, esperado %d", tc.fen, got, tc.want)
		}
	}
}

func TestInvalidFEN(t *testing.T) {
	if _, err := FromFEN("isso não é fen"); err == nil {
		t.Error("esperava erro para FEN inválida")
	}
}
