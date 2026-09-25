package book

import (
	"testing"

	"github.com/CallMePeterD/chess-ia/rules"
)

func TestPolyglotHash(t *testing.T) {
	cases := []struct {
		name     string
		fen      string
		expected uint64
	}{
		{
			name:     "Posição Inicial",
			fen:      rules.StartFEN,
			// Esta é a assinatura hexadecimal exata (semente oficial) do padrão PolyGlot 
			// desenhado pelo Fabien Letouzey para a posição inicial do xadrez.
			expected: 0x463b96181691fc9c, 
		},
		{
			name:     "Após e2e4 (Sem En Passant válido)",
			// O Polyglot ignora a casa de En Passant (e3) neste hash porque as pretas
			// não têm nenhum peão em d4 ou f4 para fazer a captura.
			fen:      "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			expected: 0x823c9b50fd114196,
		},
	}

	for _, tc := range cases {
		pos, err := rules.FromFEN(tc.fen)
		if err != nil {
			t.Fatalf("Erro ao ler FEN do teste '%s': %v", tc.name, err)
		}

		got := PolyglotHash(pos)
		if got != tc.expected {
			t.Errorf("%s: PolyglotHash() = %x, esperado %x", tc.name, got, tc.expected)
		}
	}
}