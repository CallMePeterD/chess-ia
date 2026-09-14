package eval

import (
	"testing"

	"github.com/CallMePeterD/chess-ia/rules"
)

func TestMaterial(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		want int
	}{
		{"inicial é equilibrada", rules.StartFEN, 0},
		{"brancas sem dama", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNB1KBNR w KQkq - 0 1", -900},
		{"pretas sem torre e cavalo", "2bqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQk - 0 1", 800},
		{"só reis", "4k3/8/8/8/8/8/8/4K3 w - - 0 1", 0},
	}
	for _, tc := range cases {
		pos, err := rules.FromFEN(tc.fen)
		if err != nil {
			t.Fatal(err)
		}
		if got := Evaluate(pos); got != tc.want {
			t.Errorf("%s: %d, esperado %d", tc.name, got, tc.want)
		}
	}
}
