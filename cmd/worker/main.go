// Comando worker: laço de polling contra o backend (ou o mock).
//
//	IA_TOKEN=segredo go run ./cmd/worker -backend http://localhost:8080
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/CallMePeterD/chess-ia/book"
	"github.com/CallMePeterD/chess-ia/search"
	"github.com/CallMePeterD/chess-ia/worker"
)

func main() {
	backend := flag.String("backend", envOr("IA_BACKEND_URL", "http://localhost:8080"), "URL base do backend")
	interval := flag.Duration("interval", 750*time.Millisecond, "intervalo de polling quando não há trabalho")
	bookPath := flag.String("book", envOr("IA_BOOK", "book.bin"), "livro de aberturas no formato PolyGlot (opcional)")
	flag.Parse()

	logger := log.New(os.Stderr, "[ia] ", log.LstdFlags)

	// O livro é opcional: só é erro se o caminho foi pedido explicitamente.
	b, err := book.Load(*bookPath, flagWasSet("book") || os.Getenv("IA_BOOK") != "")
	if err != nil {
		logger.Fatalf("livro de aberturas %q: %v", *bookPath, err)
	}
	if b != nil {
		search.OpeningBook = b
		logger.Printf("livro de aberturas carregado: %s (%d lances)", *bookPath, b.Len())
	}

	token := os.Getenv("IA_TOKEN")
	if token == "" {
		log.Fatal("defina IA_TOKEN com o token compartilhado do canal interno")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger.Printf("conectando em %s", *backend)
	err = worker.Run(ctx, worker.NewClient(*backend, token), worker.SearchSolver,
		worker.Config{PollInterval: *interval, Logger: logger})
	if errors.Is(err, worker.ErrUnauthorized) {
		logger.Fatal("token recusado pelo backend; confira IA_TOKEN")
	}
	if err != nil {
		logger.Fatal(err)
	}
	logger.Print("encerrado")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// flagWasSet informa se a flag foi escrita na linha de comando (e não apenas
// deixada no valor padrão).
func flagWasSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
