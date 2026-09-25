package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	// Descarrega o array diretamente do código-fonte do projeto python-chess
	resp, err := http.Get("https://raw.githubusercontent.com/niklasf/python-chess/master/chess/polyglot.py")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	code := string(body)
	start := strings.Index(code, "POLYGLOT_RANDOM_ARRAY = [")
	end := strings.Index(code[start:], "]") + start

	// Extrai apenas os números hexadecimais
	arrayStr := code[start+25 : end]

	// Cria o ficheiro de constantes na nossa pasta do livro
	out, _ := os.Create("book/polyglot_keys.go")
	defer out.Close()

	fmt.Fprintln(out, "package book")
	fmt.Fprintln(out, "// PolyglotRandoms contém as 781 sementes originais de Fabien Letouzey")
	fmt.Fprintln(out, "var PolyglotRandoms = [781]uint64{")
	fmt.Fprintln(out, arrayStr)
	fmt.Fprintln(out, "}")
	
	fmt.Println("Ficheiro book/polyglot_keys.go gerado com sucesso! Pode apagar o fetch.go.")
}