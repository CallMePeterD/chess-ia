package book

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	
	"github.com/CallMePeterD/chess-ia/rules"
)

// Entry representa um registo exato de 16 bytes do padrão Polyglot
type Entry struct {
	Key    uint64
	Move   uint16
	Weight uint16
	Learn  uint32
}

// Book guarda o ficheiro carregado em memória
type Book struct {
	entries []Entry
}

// Open lê o ficheiro binário (Big-Endian) e carrega-o para a memória.
// Como os livros costumam ter entre 1MB e 30MB, carregar num slice é super rápido.
func Open(path string) (*Book, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, _ := f.Stat()
	numEntries := info.Size() / 16

	b := &Book{
		entries: make([]Entry, numEntries),
	}

	// Polyglot guarda os dados no formato Big-Endian
	err = binary.Read(f, binary.BigEndian, &b.entries)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return b, nil
}

// GetMove calcula o Polyglot Hash da posição, procura no ficheiro .bin e devolve a UCI
func (b *Book) GetMove(p rules.Position) (string, error) {
	if len(b.entries) == 0 {
		return "", errors.New("livro vazio")
	}

	hash := PolyglotHash(p)

	l, r := 0, len(b.entries)-1
	firstIndex := -1

	// Busca Binária (encontra o registo na velocidade da luz)
	for l <= r {
		mid := l + (r-l)/2
		if b.entries[mid].Key == hash {
			firstIndex = mid
			r = mid - 1 // Continua para a esquerda para encontrar o primeiro lance desta posição
		} else if b.entries[mid].Key < hash {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}

	if firstIndex == -1 {
		return "", errors.New("posição não encontrada no livro")
	}

	// Se houver múltiplas respostas para a mesma posição, escolhemos a de maior peso (Weight)
	bestEntry := b.entries[firstIndex]
	for i := firstIndex; i < len(b.entries) && b.entries[i].Key == hash; i++ {
		if b.entries[i].Weight > bestEntry.Weight {
			bestEntry = b.entries[i]
		}
	}

	return decodeMove(bestEntry.Move), nil
}

// decodeMove traduz os 16 bits do formato Polyglot para a nossa string UCI ("e2e4")
func decodeMove(m uint16) string {
	toFile := m & 7
	toRank := (m >> 3) & 7
	fromFile := (m >> 6) & 7
	fromRank := (m >> 9) & 7
	promo := (m >> 12) & 7

	uci := string([]byte{
		byte('a' + fromFile),
		byte('1' + fromRank),
		byte('a' + toFile),
		byte('1' + toRank),
	})

	switch promo {
	case 1: uci += "n"
	case 2: uci += "b"
	case 3: uci += "r"
	case 4: uci += "q"
	}
	return uci
}
