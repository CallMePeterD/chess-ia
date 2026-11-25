package worker_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CallMePeterD/chess-ia/mock"
	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/worker"
)

const (
	token    = "segredo"
	workerID = "worker-teste"
)

var fastCfg = worker.Config{PollInterval: 5 * time.Millisecond, MaxBackoff: 20 * time.Millisecond}

// job monta um Job do contrato v1 com os campos usados nos testes.
func job(jobID, gameID string, rev int64, fen string, budgetMs int) worker.Job {
	return worker.Job{
		JobID:        jobID,
		GameID:       gameID,
		Revision:     rev,
		Position:     worker.Position{FEN: fen, SideToMove: "WHITE"},
		TimeBudgetMs: budgetMs,
	}
}

// runUntil roda o worker até cond ser verdadeira (ou estourar o prazo).
func runUntil(t *testing.T, c *worker.Client, solve worker.Solver, cond func() bool) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx, c, solve, fastCfg) }()
	for !cond() {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			t.Fatal("prazo estourado esperando o worker")
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	return <-done
}

func TestWorkerEndToEnd(t *testing.T) {
	m := mock.New(token)
	srv := httptest.NewServer(m)
	defer srv.Close()

	m.Enqueue(
		job("j1", "g1", 1, "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1", 1000),
		job("j2", "g2", 1, rules.StartFEN, 300),
	)

	c := worker.NewClient(srv.URL, workerID, token)
	if err := runUntil(t, c, worker.SearchSolver, func() bool { return len(m.Results()) == 2 }); err != nil {
		t.Fatal(err)
	}

	got := m.Results()
	if got[0].JobID != "j1" || got[0].UCI != "a1a8" {
		t.Errorf("j1: esperava mate a1a8, veio %+v", got[0])
	}
	if got[1].JobID != "j2" || got[1].UCI == "" {
		t.Errorf("j2: resultado inesperado %+v", got[1])
	}
	for _, r := range got {
		if r.WorkerID != workerID {
			t.Errorf("workerId não foi enviado: %+v", r)
		}
	}
}

func TestSubmitStatuses(t *testing.T) {
	m := mock.New(token)
	srv := httptest.NewServer(m)
	defer srv.Close()
	ctx := context.Background()
	c := worker.NewClient(srv.URL, workerID, token)

	if j, err := c.Next(ctx); j != nil || err != nil {
		t.Fatalf("fila vazia: esperava (nil, nil), veio (%v, %v)", j, err)
	}

	m.Enqueue(job("j1", "g", 7, rules.StartFEN, 300))
	j, err := c.Next(ctx)
	if err != nil || j == nil {
		t.Fatalf("esperava job, veio (%v, %v)", j, err)
	}
	if j.Revision != 7 || j.Position.FEN != rules.StartFEN || j.TimeBudgetMs != 300 {
		t.Errorf("job veio incompleto: %+v", j)
	}

	base := worker.Result{JobID: "j1", GameID: "g", Revision: 7}

	ilegal := base
	ilegal.UCI = "e2e5"
	if _, err := c.Submit(ctx, ilegal); !errors.Is(err, worker.ErrInvalidCandidate) {
		t.Errorf("lance ilegal: esperava ErrInvalidCandidate, veio %v", err)
	} else if code := codeOf(err); code != worker.CodeIllegalMove {
		t.Errorf("lance ilegal: esperava %s, veio %q", worker.CodeIllegalMove, code)
	}

	legal := base
	legal.UCI = "e2e4"
	ack, err := c.Submit(ctx, legal)
	if err != nil {
		t.Fatalf("lance legal: %v", err)
	}
	if ack.Status != "APPLIED" || ack.Revision != 8 {
		t.Errorf("ack inesperado: %+v", ack)
	}

	// Repetição idêntica é idempotente e devolve o mesmo resultado.
	if ack2, err := c.Submit(ctx, legal); err != nil || ack2.Revision != ack.Revision {
		t.Errorf("reenvio idêntico: esperava mesmo ack, veio (%+v, %v)", ack2, err)
	}

	// Candidato diferente para trabalho concluído é recusado.
	outro := base
	outro.UCI = "d2d4"
	if _, err := c.Submit(ctx, outro); !errors.Is(err, worker.ErrJobGone) {
		t.Errorf("trabalho concluído: esperava ErrJobGone, veio %v", err)
	} else if code := codeOf(err); code != worker.CodeJobAlreadyDone {
		t.Errorf("trabalho concluído: esperava %s, veio %q", worker.CodeJobAlreadyDone, code)
	}

	bad := worker.NewClient(srv.URL, workerID, "errado")
	if _, err := bad.Next(ctx); !errors.Is(err, worker.ErrUnauthorized) {
		t.Errorf("token errado: esperava ErrUnauthorized, veio %v", err)
	}
}

// Um jobId de outro worker é recusado com WORKER_OWNERSHIP_CONFLICT.
func TestSubmitOwnershipConflict(t *testing.T) {
	m := mock.New(token)
	srv := httptest.NewServer(m)
	defer srv.Close()
	ctx := context.Background()

	m.Enqueue(job("j1", "g", 1, rules.StartFEN, 300))
	dono := worker.NewClient(srv.URL, "worker-1", token)
	if _, err := dono.Next(ctx); err != nil {
		t.Fatal(err)
	}

	intruso := worker.NewClient(srv.URL, "worker-2", token)
	_, err := intruso.Submit(ctx, worker.Result{JobID: "j1", GameID: "g", Revision: 1, UCI: "e2e4"})
	if !errors.Is(err, worker.ErrJobGone) {
		t.Fatalf("esperava ErrJobGone, veio %v", err)
	}
	if code := codeOf(err); code != worker.CodeOwnershipConflict {
		t.Errorf("esperava %s, veio %q", worker.CodeOwnershipConflict, code)
	}
}

// Revisão vencida é recusada com STALE_REVISION e o trabalho é descartado.
func TestSubmitStaleRevision(t *testing.T) {
	m := mock.New(token)
	srv := httptest.NewServer(m)
	defer srv.Close()
	ctx := context.Background()
	c := worker.NewClient(srv.URL, workerID, token)

	m.Enqueue(job("j1", "g", 3, rules.StartFEN, 300))
	if _, err := c.Next(ctx); err != nil {
		t.Fatal(err)
	}
	m.SetRevision("g", 4) // a partida andou enquanto a IA pensava

	_, err := c.Submit(ctx, worker.Result{JobID: "j1", GameID: "g", Revision: 3, UCI: "e2e4"})
	if !errors.Is(err, worker.ErrJobGone) {
		t.Fatalf("esperava ErrJobGone, veio %v", err)
	}
	if code := codeOf(err); code != worker.CodeStaleRevision {
		t.Errorf("esperava %s, veio %q", worker.CodeStaleRevision, code)
	}
}

func TestRunStopsOnBadToken(t *testing.T) {
	srv := httptest.NewServer(mock.New(token))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := worker.Run(ctx, worker.NewClient(srv.URL, workerID, "errado"), worker.SearchSolver, fastCfg)
	if !errors.Is(err, worker.ErrUnauthorized) {
		t.Errorf("esperava ErrUnauthorized, veio %v", err)
	}
}

// O backend oscila: /next e /result falham algumas vezes antes de funcionar.
func TestRunRetriesTransientErrors(t *testing.T) {
	m := mock.New(token)
	m.Enqueue(job("j1", "g", 1, rules.StartFEN, 300))

	var nextFails, resultFails atomic.Int32
	flaky := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == worker.PathNext && nextFails.Add(1) <= 2 {
			http.Error(w, "fora do ar", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == worker.PathResult && resultFails.Add(1) <= 2 {
			http.Error(w, "fora do ar", http.StatusBadGateway)
			return
		}
		m.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(flaky)
	defer srv.Close()

	c := worker.NewClient(srv.URL, workerID, token)
	if err := runUntil(t, c, worker.SearchSolver, func() bool { return len(m.Results()) == 1 }); err != nil {
		t.Fatal(err)
	}
}

func TestLevelFor(t *testing.T) {
	casos := []struct {
		budgetMs int
		quer     string
	}{
		{0, "media"}, {200, "facil"}, {1000, "media"}, {2000, "dificil"},
		{2999, "dificil"}, {3000, "mestre"}, {10000, "mestre"},
	}
	for _, c := range casos {
		if got := worker.LevelFor(c.budgetMs); got != c.quer {
			t.Errorf("LevelFor(%d) = %q, queria %q", c.budgetMs, got, c.quer)
		}
	}
}

// codeOf extrai o código do contrato de um erro devolvido pelo cliente.
func codeOf(err error) string {
	var api *worker.APIError
	if errors.As(err, &api) {
		return api.Code
	}
	return ""
}
