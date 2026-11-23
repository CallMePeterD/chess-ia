// Package mock simula os endpoints do contrato Backend-IA v1
// (/v1/ai/jobs/next e /v1/ai/jobs/result), para desenvolver e testar o
// worker sem o backend real.
package mock

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CallMePeterD/chess-ia/rules"
	"github.com/CallMePeterD/chess-ia/worker"
)

// atribuicao guarda um trabalho entregue em /next e o worker dono dele.
type atribuicao struct {
	job      worker.Job
	workerID string
	ack      *worker.Ack // preenchido depois do primeiro resultado aceito
	uci      string
}

// Server guarda uma fila de trabalhos e os resultados recebidos.
// Implementa http.Handler.
type Server struct {
	Token string
	// OnResult, se definido, é chamado a cada resultado aceito (fora do lock).
	OnResult func(worker.Job, worker.Result)

	mu       sync.Mutex
	queue    []worker.Job
	atribui  map[string]*atribuicao // jobId -> atribuição
	results  []worker.Result
	revisoes map[string]int64 // gameId -> revisão autoritativa
	mux      *http.ServeMux
}

// New cria um mock que exige o token dado.
func New(token string) *Server {
	s := &Server{
		Token:    token,
		atribui:  map[string]*atribuicao{},
		revisoes: map[string]int64{},
	}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("POST "+worker.PathNext, s.next)
	s.mux.HandleFunc("POST "+worker.PathResult, s.result)
	return s
}

// Enqueue adiciona trabalhos à fila. Jobs sem ExpiresAt ganham um prazo
// folgado, e a revisão de cada partida passa a ser a do último job enfileirado.
func (s *Server) Enqueue(jobs ...worker.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range jobs {
		if j.ExpiresAt.IsZero() {
			j.ExpiresAt = time.Now().Add(time.Minute)
		}
		s.revisoes[j.GameID] = j.Revision
		s.queue = append(s.queue, j)
	}
}

// SetRevision força a revisão autoritativa de uma partida, para testar
// STALE_REVISION.
func (s *Server) SetRevision(gameID string, rev int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revisoes[gameID] = rev
}

// Results devolve uma cópia dos resultados aceitos.
func (s *Server) Results() []worker.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]worker.Result(nil), s.results...)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+s.Token {
		fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "token inválido")
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) next(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkerID string `json:"workerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.WorkerID) == "" {
		fail(w, http.StatusBadRequest, worker.CodeValidationError, "workerId obrigatório")
		return
	}

	s.mu.Lock()
	if len(s.queue) == 0 {
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	job := s.queue[0]
	s.queue = s.queue[1:]
	s.atribui[job.JobID] = &atribuicao{job: job, workerID: req.WorkerID}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func (s *Server) result(w http.ResponseWriter, r *http.Request) {
	var res worker.Result
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		fail(w, http.StatusBadRequest, worker.CodeValidationError, "json inválido")
		return
	}
	if res.WorkerID == "" || res.JobID == "" || res.UCI == "" {
		fail(w, http.StatusBadRequest, worker.CodeValidationError, "workerId, jobId e uci são obrigatórios")
		return
	}

	s.mu.Lock()
	a, ok := s.atribui[res.JobID]
	if !ok {
		s.mu.Unlock()
		fail(w, http.StatusNotFound, worker.CodeJobNotFound, "jobId desconhecido")
		return
	}
	if a.workerID != res.WorkerID {
		s.mu.Unlock()
		fail(w, http.StatusConflict, worker.CodeOwnershipConflict, "trabalho pertence a outro worker")
		return
	}
	if a.ack != nil {
		// Repetição idêntica devolve o mesmo resultado; diferente é recusada.
		ack, mesmo := *a.ack, a.uci == res.UCI
		s.mu.Unlock()
		if mesmo {
			ok200(w, ack)
			return
		}
		fail(w, http.StatusConflict, worker.CodeJobAlreadyDone, "trabalho já concluído com outro candidato")
		return
	}
	if !a.job.ExpiresAt.IsZero() && time.Now().After(a.job.ExpiresAt) {
		s.mu.Unlock()
		fail(w, http.StatusConflict, worker.CodeJobExpired, "trabalho expirado")
		return
	}
	if rev, temRev := s.revisoes[res.GameID]; temRev && res.Revision != rev {
		s.mu.Unlock()
		fail(w, http.StatusConflict, worker.CodeStaleRevision, "revisão vencida")
		return
	}

	pos, err := rules.FromFEN(a.job.Position.FEN)
	if err != nil {
		s.mu.Unlock()
		fail(w, http.StatusUnprocessableEntity, worker.CodeInvalidGameState, err.Error())
		return
	}
	uci := strings.TrimSpace(res.UCI)
	legal := false
	for _, m := range pos.ValidMoves() {
		if m.UCI() == uci {
			legal = true
			break
		}
	}
	if !legal {
		s.mu.Unlock()
		fail(w, http.StatusUnprocessableEntity, worker.CodeIllegalMove, "lance ilegal ou formato UCI inválido")
		return
	}

	rev := s.revisoes[res.GameID] + 1
	s.revisoes[res.GameID] = rev
	ack := worker.Ack{Status: "APPLIED", GameID: res.GameID, Revision: rev}
	a.ack, a.uci = &ack, uci
	s.results = append(s.results, res)
	job, cb := a.job, s.OnResult
	s.mu.Unlock()

	if cb != nil {
		cb(job, res)
	}
	ok200(w, ack)
}

func ok200(w http.ResponseWriter, ack worker.Ack) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ack)
}

func fail(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(worker.APIError{Code: code, Message: msg})
}
