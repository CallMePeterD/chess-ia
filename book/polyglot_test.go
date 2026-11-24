package book

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/CallMePeterD/chess-ia/rules"
)

// escreveLivro grava um .bin no formato Polyglot: registos de 16 bytes em
// big-endian, ordenados por chave, que é o que a busca binária de GetMove exige.
func escreveLivro(t *testing.T, entradas []Entry) string {
	t.Helper()
	sort.SliceStable(entradas, func(i, j int) bool { return entradas[i].Key < entradas[j].Key })

	caminho := filepath.Join(t.TempDir(), "livro.bin")
	f, err := os.Create(caminho)
	if err != nil {
		t.Fatalf("não consegui criar o livro: %v", err)
	}
	defer f.Close()
	if err := binary.Write(f, binary.BigEndian, entradas); err != nil {
		t.Fatalf("não consegui gravar o livro: %v", err)
	}
	return caminho
}

func hashDe(t *testing.T, fen string) uint64 {
	t.Helper()
	p, err := rules.FromFEN(fen)
	if err != nil {
		t.Fatalf("FEN inválida %q: %v", fen, err)
	}
	return PolyglotHash(p)
}

func posicaoDe(t *testing.T, fen string) rules.Position {
	t.Helper()
	p, err := rules.FromFEN(fen)
	if err != nil {
		t.Fatalf("FEN inválida %q: %v", fen, err)
	}
	return p
}

// codifica monta os 16 bits de um lance no formato Polyglot.
func codifica(fromFile, fromRank, toFile, toRank, promo uint16) uint16 {
	return toFile | toRank<<3 | fromFile<<6 | fromRank<<9 | promo<<12
}

func TestDecodeMove(t *testing.T) {
	cases := []struct {
		nome string
		bits uint16
		want string
	}{
		{"e2e4", codifica(4, 1, 4, 3, 0), "e2e4"},
		{"a1h8", codifica(0, 0, 7, 7, 0), "a1h8"},
		{"promoção a cavalo", codifica(0, 6, 0, 7, 1), "a7a8n"},
		{"promoção a bispo", codifica(1, 6, 1, 7, 2), "b7b8b"},
		{"promoção a torre", codifica(2, 6, 2, 7, 3), "c7c8r"},
		{"promoção a dama", codifica(3, 6, 3, 7, 4), "d7d8q"},
	}
	for _, tc := range cases {
		if got := decodeMove(tc.bits); got != tc.want {
			t.Errorf("%s: decodeMove deu %q, esperado %q", tc.nome, got, tc.want)
		}
	}
}

func TestLenDeLivroNuloEhZero(t *testing.T) {
	var b *Book
	if got := b.Len(); got != 0 {
		t.Errorf("livro nulo deveria ter 0 lances, tem %d", got)
	}
}

func TestOpenCarregaAsEntradas(t *testing.T) {
	chave := hashDe(t, rules.StartFEN)
	caminho := escreveLivro(t, []Entry{
		{Key: chave, Move: codifica(4, 1, 4, 3, 0), Weight: 10},
		{Key: chave, Move: codifica(3, 1, 3, 3, 0), Weight: 90},
	})

	b, err := Open(caminho)
	if err != nil {
		t.Fatalf("Open falhou: %v", err)
	}
	if got := b.Len(); got != 2 {
		t.Errorf("livro deveria ter 2 lances, tem %d", got)
	}
}

func TestOpenFalhaEmArquivoInexistente(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "nao-existe.bin"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("esperava erro de arquivo inexistente, veio %v", err)
	}
}

// Jogar sem teoria de abertura é aceitável: o caminho padrão ausente não é erro.
func TestLoadSemLivroNaoEhErroQuandoNaoExigido(t *testing.T) {
	b, err := Load(filepath.Join(t.TempDir(), "nao-existe.bin"), false)
	if err != nil {
		t.Errorf("livro opcional ausente não deveria dar erro, deu %v", err)
	}
	if b != nil {
		t.Errorf("sem livro, Load deveria devolver nil, devolveu %v", b)
	}
}

// Mas se o utilizador apontou um caminho, o arquivo ausente é erro.
func TestLoadFalhaQuandoOLivroEhExigido(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nao-existe.bin"), true); err == nil {
		t.Error("livro exigido e ausente deveria dar erro")
	}
}

func TestLoadDevolveOLivroQuandoEleExiste(t *testing.T) {
	caminho := escreveLivro(t, []Entry{{Key: hashDe(t, rules.StartFEN), Move: codifica(4, 1, 4, 3, 0), Weight: 1}})

	b, err := Load(caminho, false)
	if err != nil {
		t.Fatalf("Load falhou: %v", err)
	}
	if b.Len() != 1 {
		t.Errorf("livro deveria ter 1 lance, tem %d", b.Len())
	}
}

// Havendo várias respostas para a mesma posição, vale a de maior peso.
func TestGetMoveEscolheOLanceDeMaiorPeso(t *testing.T) {
	chave := hashDe(t, rules.StartFEN)
	caminho := escreveLivro(t, []Entry{
		{Key: chave, Move: codifica(4, 1, 4, 3, 0), Weight: 10},  // e2e4
		{Key: chave, Move: codifica(3, 1, 3, 3, 0), Weight: 250}, // d2d4
		{Key: chave, Move: codifica(2, 1, 2, 3, 0), Weight: 70},  // c2c4
	})
	b, err := Open(caminho)
	if err != nil {
		t.Fatalf("Open falhou: %v", err)
	}

	lance, err := b.GetMove(posicaoDe(t, rules.StartFEN))
	if err != nil {
		t.Fatalf("GetMove falhou: %v", err)
	}
	if lance != "d2d4" {
		t.Errorf("deveria escolher d2d4 (peso 250), escolheu %s", lance)
	}
}

func TestGetMoveEncontraAPosicaoEntreVariasOutras(t *testing.T) {
	alvo := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	entradas := []Entry{
		{Key: hashDe(t, rules.StartFEN), Move: codifica(4, 1, 4, 3, 0), Weight: 1},
		{Key: hashDe(t, alvo), Move: codifica(4, 6, 4, 4, 0), Weight: 1}, // e7e5
		{Key: hashDe(t, "4k3/8/8/8/8/8/8/4K3 w - - 0 1"), Move: codifica(4, 0, 4, 1, 0), Weight: 1},
	}
	b, err := Open(escreveLivro(t, entradas))
	if err != nil {
		t.Fatalf("Open falhou: %v", err)
	}

	lance, err := b.GetMove(posicaoDe(t, alvo))
	if err != nil {
		t.Fatalf("GetMove falhou: %v", err)
	}
	if lance != "e7e5" {
		t.Errorf("deveria achar e7e5, achou %s", lance)
	}
}

func TestGetMoveFalhaQuandoAPosicaoNaoEstaNoLivro(t *testing.T) {
	b, err := Open(escreveLivro(t, []Entry{
		{Key: hashDe(t, rules.StartFEN), Move: codifica(4, 1, 4, 3, 0), Weight: 1},
	}))
	if err != nil {
		t.Fatalf("Open falhou: %v", err)
	}

	fora := "8/2k5/8/8/3Q4/8/5PPP/6K1 w - - 0 1"
	if lance, err := b.GetMove(posicaoDe(t, fora)); err == nil {
		t.Errorf("posição fora do livro deveria dar erro, devolveu %q", lance)
	}
}

func TestGetMoveFalhaEmLivroVazio(t *testing.T) {
	b, err := Open(escreveLivro(t, nil))
	if err != nil {
		t.Fatalf("Open falhou: %v", err)
	}
	if _, err := b.GetMove(posicaoDe(t, rules.StartFEN)); err == nil {
		t.Error("livro vazio deveria dar erro")
	}
}
