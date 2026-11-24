package eval

import (
	"strings"
	"testing"

	"github.com/CallMePeterD/chess-ia/rules"
)

func posicao(t *testing.T, fen string) rules.Position {
	t.Helper()
	p, err := rules.FromFEN(fen)
	if err != nil {
		t.Fatalf("FEN inválida %q: %v", fen, err)
	}
	return p
}

// espelha troca as cores e inverte as linhas de uma FEN. O que era vantagem das
// brancas vira a mesma vantagem das pretas, então a avaliação tem de inverter o
// sinal e manter o módulo.
func espelha(fen string) string {
	campos := strings.Fields(fen)
	linhas := strings.Split(campos[0], "/")
	for i, j := 0, len(linhas)-1; i < j; i, j = i+1, j-1 {
		linhas[i], linhas[j] = linhas[j], linhas[i]
	}
	for i, linha := range linhas {
		var sb strings.Builder
		for _, c := range linha {
			switch {
			case c >= 'a' && c <= 'z':
				sb.WriteRune(c - 32)
			case c >= 'A' && c <= 'Z':
				sb.WriteRune(c + 32)
			default:
				sb.WriteRune(c)
			}
		}
		linhas[i] = sb.String()
	}
	campos[0] = strings.Join(linhas, "/")
	if len(campos) > 1 {
		if campos[1] == "w" {
			campos[1] = "b"
		} else {
			campos[1] = "w"
		}
	}
	// Roque e en passant não sobrevivem ao espelhamento e não entram na avaliação.
	if len(campos) > 2 {
		campos[2] = "-"
	}
	if len(campos) > 3 {
		campos[3] = "-"
	}
	return strings.Join(campos, " ")
}

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
		pos := posicao(t, tc.fen)
		if got := Material(pos); got != tc.want {
			t.Errorf("%s: %d, esperado %d", tc.name, got, tc.want)
		}
	}
}

// Material conta peças, não casas: mover a mesma peça não altera a soma.
func TestMaterialIgnoraAPosicaoDasPecas(t *testing.T) {
	canto := posicao(t, "4k3/8/8/8/8/8/8/N3K3 w - - 0 1")
	centro := posicao(t, "4k3/8/8/8/4N3/8/8/4K3 w - - 0 1")

	if Material(canto) != Material(centro) {
		t.Errorf("mesma peça em casas diferentes deu %d e %d", Material(canto), Material(centro))
	}
	if Material(canto) != PieceValue[rules.Knight] {
		t.Errorf("um cavalo a mais deveria valer %d, deu %d", PieceValue[rules.Knight], Material(canto))
	}
}

func TestEvaluateNaPosicaoInicialEstaEquilibrada(t *testing.T) {
	if got := Evaluate(posicao(t, rules.StartFEN)); got != 0 {
		t.Errorf("posição inicial é simétrica, deveria avaliar 0, deu %d", got)
	}
}

// A avaliação é do ponto de vista das brancas. Espelhar a posição tem de inverter
// exatamente o sinal — é o que garante que as tabelas de posicionamento estão
// sendo espelhadas junto, e não aplicadas iguais para os dois lados.
func TestEvaluateEspelhadaInverteOSinal(t *testing.T) {
	fens := []string{
		rules.StartFEN,
		"4k3/8/8/8/4N3/8/8/4K3 w - - 0 1",
		"4k3/8/8/8/8/8/8/N3K3 w - - 0 1",
		"4k3/4p3/8/8/8/8/4P3/4K3 w - - 0 1",
		"r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
		"8/2k5/8/8/3Q4/8/5PPP/6K1 w - - 0 1",
		"2r3k1/5ppp/8/8/8/8/5PPP/2R3K1 w - - 0 1",
	}
	for _, fen := range fens {
		direta := Evaluate(posicao(t, fen))
		invertida := Evaluate(posicao(t, espelha(fen)))
		if direta != -invertida {
			t.Errorf("%s: avaliou %d, espelhada avaliou %d (esperado %d)", fen, direta, invertida, -direta)
		}
	}
}

// As tabelas existem para premiar o centro: a mesma peça vale mais centralizada.
func TestEvaluatePremiaCentralizacao(t *testing.T) {
	cases := []struct {
		peca    string
		canto   string
		central string
	}{
		{"cavalo", "4k3/8/8/8/8/8/8/N3K3 w - - 0 1", "4k3/8/8/8/4N3/8/8/4K3 w - - 0 1"},
		{"bispo", "4k3/8/8/8/8/8/8/B3K3 w - - 0 1", "4k3/8/8/8/3B4/8/8/4K3 w - - 0 1"},
		{"dama", "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1", "4k3/8/8/3Q4/8/8/8/4K3 w - - 0 1"},
	}
	for _, tc := range cases {
		canto := Evaluate(posicao(t, tc.canto))
		central := Evaluate(posicao(t, tc.central))
		if central <= canto {
			t.Errorf("%s: no centro avaliou %d, no canto %d — o centro deveria valer mais",
				tc.peca, central, canto)
		}
	}
}

// O peão avança para promover: quanto mais perto da oitava, melhor.
func TestEvaluatePremiaPeaoAvancado(t *testing.T) {
	recuado := Evaluate(posicao(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"))
	avancado := Evaluate(posicao(t, "4k3/4P3/8/8/8/8/8/4K3 w - - 0 1"))

	if avancado <= recuado {
		t.Errorf("peão na sétima avaliou %d e na segunda %d — avançar deveria valer mais",
			avancado, recuado)
	}
}

// Uma peça a mais domina qualquer diferença de posicionamento.
func TestEvaluateRefleteVantagemMaterial(t *testing.T) {
	semDama := Evaluate(posicao(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNB1KBNR w KQkq - 0 1"))
	comDama := Evaluate(posicao(t, rules.StartFEN))

	if semDama >= comDama {
		t.Errorf("brancas sem dama avaliaram %d, contra %d com dama", semDama, comDama)
	}
	if diferenca := comDama - semDama; diferenca < PieceValue[rules.Queen]/2 {
		t.Errorf("perder a dama custou só %d centipeões", diferenca)
	}
}

// O rei não entra no material: capturá-lo não é um lance possível, e dar valor
// a ele distorceria toda a conta.
func TestReiNaoTemValorMaterial(t *testing.T) {
	if PieceValue[rules.King] != 0 {
		t.Errorf("rei deveria valer 0 no material, vale %d", PieceValue[rules.King])
	}
	soReis := posicao(t, "4k3/8/8/8/8/8/8/4K3 w - - 0 1")
	if got := Material(soReis); got != 0 {
		t.Errorf("só com os dois reis o material deveria ser 0, deu %d", got)
	}
}
