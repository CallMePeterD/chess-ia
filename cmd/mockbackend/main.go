// Comando mockbackend: sobe os endpoints do contrato Backend-IA v1
// (/v1/ai/jobs/next e /v1/ai/jobs/result) simulados, com uma fila de
// posições de exemplo. Com -selfplay, cada lance aceito gera o
// próximo pedido da mesma partida, então dá para ver a IA jogando sozinha.
//
//	IA_TOKEN=segredo go run ./cmd/mockbackend -selfplay
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CallMePeterD/chess-ia/mock"
	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/worker"
)

func main() {
	addr := flag.String("addr", ":8080", "endereço de escuta")
	selfplay := flag.Bool("selfplay", false, "a partir da posição inicial, a IA joga contra si mesma")
	budget := flag.Int("budget", 1000, "timeBudgetMs enviado nos trabalhos gerados")
	maxPlies := flag.Int("plies", 40, "limite de meios-lances no selfplay")
	flag.Parse()

	token := os.Getenv("IA_TOKEN")
	if token == "" {
		log.Fatal("defina IA_TOKEN (o mesmo usado pelo worker)")
	}

	m := mock.New(token)
	var seq atomic.Int64
	revisoes := map[string]int64{}
	var revMu sync.Mutex
	newJob := func(game, fen string) worker.Job {
		revMu.Lock()
		rev := revisoes[game] + 1
		revisoes[game] = rev
		revMu.Unlock()
		return worker.Job{
			JobID:        fmt.Sprintf("job-%d", seq.Add(1)),
			GameID:       game,
			Revision:     rev,
			Position:     worker.Position{FEN: fen, SideToMove: sideToMove(fen)},
			TimeBudgetMs: *budget,
			ExpiresAt:    time.Now().Add(30 * time.Second),
		}
	}

	if *selfplay {
		m.Enqueue(newJob("selfplay", rules.StartFEN))
	} else {
		m.Enqueue(
			newJob("demo-inicial", rules.StartFEN),
			newJob("demo-mate", "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1"),
			newJob("demo-captura", "4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1"),
		)
	}

	var plies atomic.Int64
	m.OnResult = func(job worker.Job, res worker.Result) {
		log.Printf("%s %s (rev %d): %s", job.GameID, job.JobID, job.Revision, res.UCI)
		if !*selfplay {
			return
		}

		pos, _ := rules.FromFEN(job.Position.FEN)

		// Converte a string UCI para a nossa struct Move de 16 bits
		var mv rules.Move
		for _, m := range pos.ValidMoves() {
			if m.UCI() == res.UCI {
				mv = m
				break
			}
		}

		next := pos.Apply(mv)
		validMoves := next.ValidMoves()

		// Verifica o final do jogo da mesma forma que a nossa Poda Alfa-Beta
		if len(validMoves) == 0 {
			if next.InCheck(next.SideToMove) {
				log.Printf("xeque-mate! fen final: %s", next.FEN())
			} else {
				log.Printf("afogamento. fen final: %s", next.FEN())
			}
		} else if plies.Add(1) >= int64(*maxPlies) {
			log.Printf("limite de %d meios-lances. fen final: %s", *maxPlies, next.FEN())
		} else {
			m.Enqueue(newJob(job.GameID, next.FEN()))
		}
	}

	log.Printf("mock do backend em %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, m))
}

// sideToMove extrai de quem é a vez a partir da FEN, no vocabulário do
// contrato (WHITE/BLACK).
func sideToMove(fen string) string {
	campos := strings.Fields(fen)
	if len(campos) > 1 && campos[1] == "b" {
		return "BLACK"
	}
	return "WHITE"
}
