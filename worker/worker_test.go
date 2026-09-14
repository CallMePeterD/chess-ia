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

const token = "segredo"

var fastCfg = worker.Config{PollInterval: 5 * time.Millisecond, MaxBackoff: 20 * time.Millisecond}

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
		worker.Job{GameID: "g1", RequestID: "r1", FEN: "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1", Dificuldade: "media"},
		worker.Job{GameID: "g2", RequestID: "r2", FEN: rules.StartFEN, Dificuldade: "facil"},
	)

	c := worker.NewClient(srv.URL, token)
	if err := runUntil(t, c, worker.SearchSolver, func() bool { return len(m.Results()) == 2 }); err != nil {
		t.Fatal(err)
	}

	got := m.Results()
	if got[0].RequestID != "r1" || got[0].Move != "a1a8" {
		t.Errorf("r1: esperava mate a1a8, veio %+v", got[0])
	}
	if got[1].RequestID != "r2" || got[1].Move == "" {
		t.Errorf("r2: resultado inesperado %+v", got[1])
	}
}

func TestSubmitStatuses(t *testing.T) {
	m := mock.New(token)
	srv := httptest.NewServer(m)
	defer srv.Close()
	ctx := context.Background()
	c := worker.NewClient(srv.URL, token)

	if job, err := c.Next(ctx); job != nil || err != nil {
		t.Fatalf("fila vazia: esperava (nil, nil), veio (%v, %v)", job, err)
	}

	m.Enqueue(worker.Job{GameID: "g", RequestID: "r", FEN: rules.StartFEN, Dificuldade: "facil"})
	job, err := c.Next(ctx)
	if err != nil || job == nil {
		t.Fatalf("esperava job, veio (%v, %v)", job, err)
	}

	if err := c.Submit(ctx, worker.Result{GameID: "g", RequestID: "r", Move: "e2e5"}); !errors.Is(err, worker.ErrInvalidMove) {
		t.Errorf("lance ilegal: esperava ErrInvalidMove, veio %v", err)
	}
	if err := c.Submit(ctx, worker.Result{GameID: "g", RequestID: "r", Move: "e2e4"}); err != nil {
		t.Errorf("lance legal: %v", err)
	}
	if err := c.Submit(ctx, worker.Result{GameID: "g", RequestID: "r", Move: "e2e4"}); !errors.Is(err, worker.ErrConflict) {
		t.Errorf("reenvio: esperava ErrConflict, veio %v", err)
	}

	bad := worker.NewClient(srv.URL, "errado")
	if _, err := bad.Next(ctx); !errors.Is(err, worker.ErrUnauthorized) {
		t.Errorf("token errado: esperava ErrUnauthorized, veio %v", err)
	}
}

func TestRunStopsOnBadToken(t *testing.T) {
	srv := httptest.NewServer(mock.New(token))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := worker.Run(ctx, worker.NewClient(srv.URL, "errado"), worker.SearchSolver, fastCfg)
	if !errors.Is(err, worker.ErrUnauthorized) {
		t.Errorf("esperava ErrUnauthorized, veio %v", err)
	}
}

// O backend oscila: /next e /result falham algumas vezes antes de funcionar.
func TestRunRetriesTransientErrors(t *testing.T) {
	m := mock.New(token)
	m.Enqueue(worker.Job{GameID: "g", RequestID: "r", FEN: rules.StartFEN, Dificuldade: "facil"})

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

	c := worker.NewClient(srv.URL, token)
	if err := runUntil(t, c, worker.SearchSolver, func() bool { return len(m.Results()) == 1 }); err != nil {
		t.Fatal(err)
	}
}
