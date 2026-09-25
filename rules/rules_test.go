package rules

import "testing"

// Valores de referência: https://www.chessprogramming.org/Perft_Results
var perftCases = []struct {
	name  string
	fen   string
	nodes []int // nodes[i] = perft(i+1)
}{
	{"inicial", StartFEN, []int{20, 400, 8902, 197281}},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", []int{48, 2039, 97862}},
	{"pos3", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", []int{14, 191, 2812, 43238}},
	{"pos4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", []int{6, 264, 9467}},
	{"pos5", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", []int{44, 1486, 62379}},
}

// Função auxiliar para substituir o antigo ParseUCI nos testes
func parseUCI(p Position, uci string) (Move, bool) {
	for _, m := range p.ValidMoves() {
		if m.UCI() == uci {
			return m, true
		}
	}
	return 0, false
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
	pos, err := FromFEN(StartFEN)
	if err != nil {
		t.Fatal(err)
	}
	mv, ok := parseUCI(pos, "e2e4")
	if !ok {
		t.Fatal("lance e2e4 não encontrado")
	}
	
	next := pos.Apply(mv)
	
	// Como structs em Go são copiadas por valor, Apply nunca deve alterar o pos original
	if pos.SideToMove != White {
		t.Errorf("posição original mudou: o turno alterou-se de forma inesperada")
	}
	if next.SideToMove != Black {
		t.Errorf("depois de e2e4 deveria ser a vez das pretas")
	}
}

func TestPromotionUCI(t *testing.T) {
	pos, err := FromFEN("8/4P3/8/8/8/8/k7/7K w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	mv, ok := parseUCI(pos, "e7e8q")
	if !ok {
		t.Fatal("lance e7e8q não encontrado nos lances legais")
	}
	if !mv.IsPromotion() {
		t.Errorf("e7e8q deveria ser assinalado como promoção")
	}
}

func TestStatus(t *testing.T) {
	cases := []struct {
		fen       string
		isOver    bool
		isMate    bool
		isStale   bool
	}{
		{StartFEN, false, false, false},
		{"R5k1/5ppp/8/8/8/8/8/6K1 b - - 0 1", true, true, false}, // Checkmate
		{"7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", true, false, true},    // Stalemate
	}
	
	for _, tc := range cases {
		pos, err := FromFEN(tc.fen)
		if err != nil {
			t.Fatal(err)
		}
		
		moves := pos.ValidMoves()
		over := len(moves) == 0
		mate := over && pos.InCheck(pos.SideToMove)
		stale := over && !pos.InCheck(pos.SideToMove)
		
		if over != tc.isOver || mate != tc.isMate || stale != tc.isStale {
			t.Errorf("%s: status incorreto. esperado over=%v mate=%v stale=%v, obtido over=%v mate=%v stale=%v", 
				tc.fen, tc.isOver, tc.isMate, tc.isStale, over, mate, stale)
		}
	}
}

func TestInvalidFEN(t *testing.T) {
	if _, err := FromFEN("isso não é fen"); err == nil {
		t.Error("esperava erro para FEN inválida")
	}
}