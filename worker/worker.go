// Package worker é o cliente que conversa com o backend (contrato Backend-IA
// v1, documentado em docs/contracts/backend-ai.md do repositório do backend):
// pede trabalho em POST /v1/ai/jobs/next, calcula o lance e entrega o
// candidato em POST /v1/ai/jobs/result.
//
// O worker fornece apenas um movimento candidato. Estado, relógio, revisão,
// legalidade e conclusão da partida continuam sendo responsabilidade do
// backend.
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	PathNext   = "/v1/ai/jobs/next"
	PathResult = "/v1/ai/jobs/result"
)

// Position é a posição enviada pelo backend.
type Position struct {
	FEN        string `json:"fen"`
	SideToMove string `json:"sideToMove"`
}

// Job é um pedido de lance vindo do backend.
type Job struct {
	JobID        string    `json:"jobId"`
	GameID       string    `json:"gameId"`
	Revision     int64     `json:"revision"`
	Position     Position  `json:"position"`
	TimeBudgetMs int       `json:"timeBudgetMs"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// nextRequest é o corpo enviado a /v1/ai/jobs/next.
type nextRequest struct {
	WorkerID string `json:"workerId"`
}

// Result é o candidato entregue ao backend.
type Result struct {
	WorkerID string `json:"workerId"`
	JobID    string `json:"jobId"`
	GameID   string `json:"gameId"`
	Revision int64  `json:"revision"`
	UCI      string `json:"uci"`
}

// Ack é a resposta 200 de /v1/ai/jobs/result.
type Ack struct {
	Status   string `json:"status"`
	GameID   string `json:"gameId"`
	Revision int64  `json:"revision"`
}

// Códigos de erro do contrato.
const (
	CodeValidationError   = "VALIDATION_ERROR"
	CodeInvalidCandidate  = "INVALID_CANDIDATE"
	CodeJobNotFound       = "JOB_NOT_FOUND"
	CodeJobExpired        = "JOB_EXPIRED"
	CodeStaleRevision     = "STALE_REVISION"
	CodeJobAlreadyDone    = "JOB_ALREADY_COMPLETED"
	CodeOwnershipConflict = "WORKER_OWNERSHIP_CONFLICT"
	CodeIllegalMove       = "ILLEGAL_MOVE"
	CodeInvalidGameState  = "INVALID_GAME_STATE"
)

// APIError carrega o corpo {"code","message"} devolvido pelo backend.
// Use errors.As para ler o código.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("status %d: %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("status %d: %s", e.StatusCode, e.Code)
}

var (
	// ErrJobGone: o trabalho não vale mais (404, ou 409 por expiração,
	// revisão vencida, conclusão anterior ou posse de outro worker).
	// O worker descarta o trabalho local e volta a pedir em /next.
	ErrJobGone = errors.New("trabalho não é mais válido")
	// ErrInvalidCandidate: o backend recusou o candidato (400 ou 422).
	ErrInvalidCandidate = errors.New("candidato recusado pelo backend")
	// ErrUnauthorized: token ausente ou inválido (401/403). Não adianta repetir.
	ErrUnauthorized = errors.New("token recusado pelo backend")
)

// Client fala HTTP com o backend.
type Client struct {
	BaseURL  string
	WorkerID string
	Token    string
	HTTP     *http.Client
}

// NewClient cria um cliente com timeout razoável.
func NewClient(baseURL, workerID, token string) *Client {
	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		WorkerID: workerID,
		Token:    token,
		HTTP:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Next pede trabalho pendente. Devolve (nil, nil) quando não há nada (204).
func (c *Client) Next(ctx context.Context) (*Job, error) {
	body, err := json.Marshal(nextRequest{WorkerID: c.WorkerID})
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, PathNext, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusOK:
		var job Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			return nil, fmt.Errorf("resposta de %s inválida: %w", PathNext, err)
		}
		return &job, nil
	}
	return nil, statusError(resp)
}

// Submit entrega o candidato. Erros: ErrJobGone, ErrInvalidCandidate,
// ErrUnauthorized ou falha transitória (rede / 5xx), que vale repetir.
func (c *Client) Submit(ctx context.Context, r Result) (*Ack, error) {
	if r.WorkerID == "" {
		r.WorkerID = c.WorkerID
	}
	body, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, PathResult, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var ack Ack
		if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
			return nil, fmt.Errorf("resposta de %s inválida: %w", PathResult, err)
		}
		return &ack, nil
	}
	return nil, statusError(resp)
}

func (c *Client) do(ctx context.Context, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	return c.HTTP.Do(req)
}

// statusError traduz a resposta de erro do backend. Mantém o APIError
// embrulhado, para que errors.As recupere o código do contrato.
func statusError(resp *http.Response) error {
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrUnauthorized
	}

	apiErr := &APIError{StatusCode: resp.StatusCode}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	_ = json.Unmarshal(raw, apiErr) // corpo fora do contrato: segue com Code vazio
	if apiErr.Code == "" {
		apiErr.Message = string(bytes.TrimSpace(raw))
	}

	switch resp.StatusCode {
	case http.StatusNotFound, http.StatusConflict:
		return fmt.Errorf("%w: %w", ErrJobGone, apiErr)
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %w", ErrInvalidCandidate, apiErr)
	}
	return apiErr
}

// Solver calcula o lance (UCI) e a avaliação para um pedido.
type Solver func(ctx context.Context, job Job) (uci string, avaliacao int, err error)

// Config controla o laço de polling.
type Config struct {
	PollInterval  time.Duration // espera quando não há trabalho (padrão 750ms)
	MaxBackoff    time.Duration // teto da espera após erros (padrão 10s)
	SubmitRetries int           // tentativas extras de POST em falha transitória (padrão 3)
	Logger        *log.Logger
}

func (cfg *Config) defaults() {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 750 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 10 * time.Second
	}
	if cfg.SubmitRetries < 0 {
		cfg.SubmitRetries = 0
	} else if cfg.SubmitRetries == 0 {
		cfg.SubmitRetries = 3
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(io.Discard, "", 0)
	}
}

// Run executa o laço até ctx ser cancelado (devolve nil) ou o token ser
// recusado (devolve ErrUnauthorized).
func Run(ctx context.Context, c *Client, solve Solver, cfg Config) error {
	cfg.defaults()
	lg := cfg.Logger
	backoff := time.Duration(0)

	for {
		wait := cfg.PollInterval
		job, err := c.Next(ctx)
		switch {
		case ctx.Err() != nil:
			return nil
		case errors.Is(err, ErrUnauthorized):
			return err
		case err != nil:
			backoff = nextBackoff(backoff, cfg.PollInterval, cfg.MaxBackoff)
			wait = backoff
			lg.Printf("erro em %s: %v (nova tentativa em %s)", PathNext, err, wait)
		case job != nil:
			backoff = 0
			if err := handle(ctx, c, solve, cfg, *job); errors.Is(err, ErrUnauthorized) {
				return err
			}
			continue // pode haver mais trabalho: pergunta de novo sem esperar
		default:
			backoff = 0
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}
	}
}

func handle(ctx context.Context, c *Client, solve Solver, cfg Config, job Job) error {
	lg := cfg.Logger
	start := time.Now()
	uci, score, err := solve(ctx, job)
	if err != nil {
		// Sem lance para entregar; o backend expira o trabalho.
		lg.Printf("game=%s job=%s: não foi possível calcular: %v", job.GameID, job.JobID, err)
		return nil
	}
	lg.Printf("game=%s job=%s rev=%d: %s (%d) em %s",
		job.GameID, job.JobID, job.Revision, uci, score, time.Since(start).Round(time.Millisecond))

	res := Result{
		WorkerID: c.WorkerID,
		JobID:    job.JobID,
		GameID:   job.GameID,
		Revision: job.Revision,
		UCI:      uci,
	}
	backoff := time.Duration(0)
	for attempt := 0; ; attempt++ {
		ack, err := c.Submit(ctx, res)
		switch {
		case err == nil:
			lg.Printf("job=%s: %s (revisão %d)", job.JobID, ack.Status, ack.Revision)
			return nil
		case errors.Is(err, ErrJobGone):
			// Expirado, revisão vencida, já concluído ou de outro worker:
			// descarta e volta a pedir trabalho.
			lg.Printf("job=%s: descartado (%v)", job.JobID, err)
			return nil
		case errors.Is(err, ErrInvalidCandidate):
			lg.Printf("job=%s: backend recusou %s (%v)", job.JobID, uci, err)
			return nil
		case errors.Is(err, ErrUnauthorized), ctx.Err() != nil:
			return err
		case attempt >= cfg.SubmitRetries:
			lg.Printf("job=%s: desistindo após %d tentativas: %v", job.JobID, attempt+1, err)
			return err
		}
		backoff = nextBackoff(backoff, cfg.PollInterval, cfg.MaxBackoff)
		lg.Printf("job=%s: erro no POST: %v (nova tentativa em %s)", job.JobID, err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
}

func nextBackoff(cur, base, max time.Duration) time.Duration {
	if cur == 0 {
		return base
	}
	if cur*2 > max {
		return max
	}
	return cur * 2
}
