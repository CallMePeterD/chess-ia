// Package mock simula os dois endpoints internos do backend, para
// desenvolver e testar o worker sem o backend real.
package mock

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/worker"
)

// Server guarda uma fila de pedidos e os resultados recebidos.
// Implementa http.Handler.
type Server struct {
	Token string
	// OnResult, se definido, é chamado a cada resultado aceito (fora do lock).
	OnResult func(worker.Job, worker.Result)

	mu      sync.Mutex
	queue   []worker.Job
	pending map[string]worker.Job // entregues em /next, aguardando /result
	results []worker.Result
	mux     *http.ServeMux
}

// New cria um mock que exige o token dado.
func New(token string) *Server {
	s := &Server{Token: token, pending: map[string]worker.Job{}}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("GET "+worker.PathNext, s.next)
	s.mux.HandleFunc("POST "+worker.PathResult, s.result)
	return s
}

// Enqueue adiciona pedidos à fila.
func (s *Server) Enqueue(jobs ...worker.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, jobs...)
}

// Results devolve uma cópia dos resultados aceitos.
func (s *Server) Results() []worker.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]worker.Result(nil), s.results...)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+s.Token {
		http.Error(w, "token inválido", http.StatusUnauthorized)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) next(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	if len(s.queue) == 0 {
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	job := s.queue[0]
	s.queue = s.queue[1:]
	s.pending[job.RequestID] = job
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (s *Server) result(w http.ResponseWriter, r *http.Request) {
	var res worker.Result
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	job, ok := s.pending[res.RequestID]
	if !ok || job.GameID != res.GameID {
		s.mu.Unlock()
		http.Error(w, "requestId desconhecido ou já resolvido", http.StatusConflict)
		return
	}
	pos, err := rules.FromFEN(job.FEN)
	if err == nil {
		_, err = pos.ParseUCI(strings.TrimSpace(res.Move))
	}
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	delete(s.pending, res.RequestID)
	s.results = append(s.results, res)
	cb := s.OnResult
	s.mu.Unlock()

	if cb != nil {
		cb(job, res)
	}
	w.WriteHeader(http.StatusOK)
}
