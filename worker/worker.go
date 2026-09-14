// Package worker é o cliente que conversa com o backend (contrato worker):
// faz polling em GET /internal/v1/moves/next, calcula o lance e entrega em
// POST /internal/v1/moves/result.
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
	PathNext   = "/internal/v1/moves/next"
	PathResult = "/internal/v1/moves/result"
)

// Job é um pedido de lance vindo do backend.
type Job struct {
	GameID      string `json:"gameId"`
	RequestID   string `json:"requestId"`
	FEN         string `json:"fen"`
	Dificuldade string `json:"dificuldade"`
}

// Result é o lance calculado, entregue ao backend.
type Result struct {
	GameID    string `json:"gameId"`
	RequestID string `json:"requestId"`
	Move      string `json:"move"`
	Avaliacao int    `json:"avaliacao"`
}

var (
	// ErrConflict: requestId já resolvido ou expirado (409). Descartar.
	ErrConflict = errors.New("requestId já resolvido ou expirado")
	// ErrInvalidMove: o backend recusou o lance (422).
	ErrInvalidMove = errors.New("lance recusado pelo backend")
	// ErrUnauthorized: token ausente ou inválido (401/403). Não adianta repetir.
	ErrUnauthorized = errors.New("token recusado pelo backend")
)

// Client fala HTTP com o backend.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// NewClient cria um cliente com timeout razoável.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Next busca trabalho pendente. Devolve (nil, nil) quando não há nada (204).
func (c *Client) Next(ctx context.Context) (*Job, error) {
	resp, err := c.do(ctx, http.MethodGet, PathNext, nil)
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
			return nil, fmt.Errorf("resposta de /next inválida: %w", err)
		}
		return &job, nil
	}
	return nil, statusError(resp)
}

// Submit entrega o resultado. Erros: ErrConflict, ErrInvalidMove,
// ErrUnauthorized ou falha transitória (rede / 5xx), que vale repetir.
func (c *Client) Submit(ctx context.Context, r Result) error {
	body, err := json.Marshal(r)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPost, PathResult, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return ErrConflict
	case http.StatusUnprocessableEntity:
		return ErrInvalidMove
	}
	return statusError(resp)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.HTTP.Do(req)
}

func statusError(resp *http.Response) error {
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrUnauthorized
	}
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("%s: status %d: %s", resp.Request.URL.Path, resp.StatusCode, bytes.TrimSpace(msg))
}

// Solver calcula o lance (UCI) e a avaliação para um pedido.
type Solver func(ctx context.Context, job Job) (move string, avaliacao int, err error)

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
			lg.Printf("erro em /next: %v (nova tentativa em %s)", err, wait)
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
	move, score, err := solve(ctx, job)
	if err != nil {
		// Sem lance para entregar; o backend expira o requestId.
		lg.Printf("game=%s req=%s: não foi possível calcular: %v", job.GameID, job.RequestID, err)
		return nil
	}
	lg.Printf("game=%s req=%s: %s (%d) em %s", job.GameID, job.RequestID, move, score, time.Since(start).Round(time.Millisecond))

	res := Result{GameID: job.GameID, RequestID: job.RequestID, Move: move, Avaliacao: score}
	backoff := time.Duration(0)
	for attempt := 0; ; attempt++ {
		err := c.Submit(ctx, res)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, ErrConflict):
			lg.Printf("req=%s: 409, descartado", job.RequestID)
			return nil
		case errors.Is(err, ErrInvalidMove):
			lg.Printf("req=%s: 422, backend recusou %s", job.RequestID, move)
			return nil
		case errors.Is(err, ErrUnauthorized), ctx.Err() != nil:
			return err
		case attempt >= cfg.SubmitRetries:
			lg.Printf("req=%s: desistindo após %d tentativas: %v", job.RequestID, attempt+1, err)
			return err
		}
		backoff = nextBackoff(backoff, cfg.PollInterval, cfg.MaxBackoff)
		lg.Printf("req=%s: erro no POST: %v (nova tentativa em %s)", job.RequestID, err, backoff)
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
