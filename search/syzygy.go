package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// syzygyResponse mapeia apenas os campos que nos interessam do JSON do Lichess.
type syzygyResponse struct {
	Moves []struct {
		UCI string `json:"uci"`
	} `json:"moves"`
	Error string `json:"error"`
}

// ProbeCloudTablebase envia a string FEN para a base de dados do Lichess.
// Devolve o lance perfeito em UCI e uma flag indicando se teve sucesso.
func ProbeCloudTablebase(fen string) (string, bool) {
	// A API precisa do FEN no formato URL-encoded (ex: espaços viram %20)
	safeFEN := url.QueryEscape(fen)
	endpoint := fmt.Sprintf("http://tablebase.lichess.ovh/standard?fen=%s", safeFEN)

	// Definimos um timeout muito curto (2 segundos). Se a internet falhar, 
	// o nosso código ignora a API e cai graciosamente de volta no Minimax normal.
	client := &http.Client{Timeout: 2 * time.Second}
	
	resp, err := client.Get(endpoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", false
	}
	defer resp.Body.Close()

	var data syzygyResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", false
	}

	// Se a API devolver lances, o primeiro da lista é garantidamente o melhor caminho para a vitória/empate.
	if len(data.Moves) > 0 {
		return data.Moves[0].UCI, true
	}

	return "", false
}