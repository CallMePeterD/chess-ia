package search

import (
	"testing"
)

func TestProbeCloudTablebase_Success(t *testing.T) {
	// FEN de um final clássico de 3 peças: Rei e Torre Brancos vs Rei Preto.
	// O Lichess tem a solução exata para o Mate Forçado guardada na base de dados.
	fen := "8/8/8/8/8/8/R7/K5k1 w - - 0 1"

	move, ok := ProbeCloudTablebase(fen)

	if !ok {
		t.Fatalf("Acesso à API falhou. Verifique a sua ligação à internet ou se a API do Lichess (tablebase.lichess.ovh) está online.")
	}

	if len(move) < 4 {
		t.Errorf("A API reportou sucesso, mas o formato UCI recebido é inválido: %q", move)
	}

	// Imprime o lance exato que o supercomputador do Lichess escolheu
	t.Logf("API respondeu com sucesso! Para o FEN [%s], o lance perfeito é: %s", fen, move)
}

func TestProbeCloudTablebase_InvalidFEN(t *testing.T) {
	// Um FEN lixo formatado incorretamente para forçar a API a devolver um erro.
	// A nossa função tem de lidar com a rejeição e simplesmente devolver false.
	fen := "posicao_invalida_que_nao_existe"

	move, ok := ProbeCloudTablebase(fen)

	if ok {
		t.Errorf("A API devia ter rejeitado este FEN, mas retornou sucesso com o lance %q.", move)
	}
}