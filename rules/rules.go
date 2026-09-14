// Package rules é o adaptador de regras: o único pacote que conhece a
// biblioteca de xadrez (github.com/notnil/chess). O resto da IA fala apenas
// com os tipos daqui, então trocar de biblioteca afeta só este arquivo.
package rules

import (
	"fmt"

	"github.com/notnil/chess"
)

// Color indica o lado.
type Color int8

const (
	White Color = iota
	Black
)

// PieceType é o tipo de peça, independente de cor.
type PieceType int8

const (
	NoPiece PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// Piece é uma peça no tabuleiro. Square vai de 0 (a1) a 63 (h8),
// crescendo por coluna e depois por fileira (b1 = 1, a2 = 8).
type Piece struct {
	Type   PieceType
	Color  Color
	Square int
}

// Status descreve se a partida acabou na posição.
type Status int8

const (
	Ongoing Status = iota
	Checkmate
	Stalemate
)

// Move é um lance legal numa posição específica.
type Move struct {
	m   *chess.Move
	uci string
}

// UCI devolve o lance em notação UCI (e2e4, e7e8q).
func (mv Move) UCI() string { return mv.uci }

// IsCapture informa se o lance captura uma peça (inclui en passant).
func (mv Move) IsCapture() bool { return mv.m.HasTag(chess.Capture) }

// IsPromotion informa se o lance promove um peão.
func (mv Move) IsPromotion() bool { return mv.m.Promo() != chess.NoPieceType }

func (mv Move) String() string { return mv.uci }

// Position é uma posição imutável: Apply devolve uma nova posição e a
// original continua válida (é assim que a busca "desfaz" lances).
type Position struct {
	pos *chess.Position
}

// StartFEN é a posição inicial padrão.
const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// FromFEN interpreta uma FEN padrão.
func FromFEN(fen string) (Position, error) {
	p := &chess.Position{}
	if err := p.UnmarshalText([]byte(fen)); err != nil {
		return Position{}, fmt.Errorf("fen inválida %q: %w", fen, err)
	}
	return Position{pos: p}, nil
}

// Start devolve a posição inicial.
func Start() Position {
	return Position{pos: chess.StartingPosition()}
}

// FEN devolve a posição em FEN.
func (p Position) FEN() string { return p.pos.String() }

// SideToMove devolve quem joga.
func (p Position) SideToMove() Color {
	if p.pos.Turn() == chess.White {
		return White
	}
	return Black
}

// LegalMoves lista os lances legais do lado que joga.
func (p Position) LegalMoves() []Move {
	ms := p.pos.ValidMoves()
	out := make([]Move, len(ms))
	for i, m := range ms {
		out[i] = Move{m: m, uci: chess.UCINotation{}.Encode(p.pos, m)}
	}
	return out
}

// Apply aplica um lance (obtido de LegalMoves desta posição) e devolve a
// posição resultante.
func (p Position) Apply(mv Move) Position {
	return Position{pos: p.pos.Update(mv.m)}
}

// ParseUCI converte uma string UCI num lance legal desta posição.
func (p Position) ParseUCI(s string) (Move, error) {
	for _, mv := range p.LegalMoves() {
		if mv.uci == s {
			return mv, nil
		}
	}
	return Move{}, fmt.Errorf("lance %q não é legal em %s", s, p.FEN())
}

// Status informa xeque-mate, afogamento ou partida em andamento.
func (p Position) Status() Status {
	switch p.pos.Status() {
	case chess.Checkmate:
		return Checkmate
	case chess.Stalemate:
		return Stalemate
	}
	return Ongoing
}

// Pieces lista todas as peças no tabuleiro.
func (p Position) Pieces() []Piece {
	board := p.pos.Board()
	out := make([]Piece, 0, 32)
	for sq := chess.A1; sq <= chess.H8; sq++ {
		cp := board.Piece(sq)
		if cp == chess.NoPiece {
			continue
		}
		c := White
		if cp.Color() == chess.Black {
			c = Black
		}
		out = append(out, Piece{Type: pieceType(cp.Type()), Color: c, Square: int(sq)})
	}
	return out
}

func pieceType(t chess.PieceType) PieceType {
	switch t {
	case chess.Pawn:
		return Pawn
	case chess.Knight:
		return Knight
	case chess.Bishop:
		return Bishop
	case chess.Rook:
		return Rook
	case chess.Queen:
		return Queen
	case chess.King:
		return King
	}
	return NoPiece
}

// Perft conta as posições folha até a profundidade dada. Serve para
// validar a geração de lances contra valores conhecidos.
func Perft(p Position, depth int) int64 {
	if depth == 0 {
		return 1
	}
	moves := p.pos.ValidMoves()
	if depth == 1 {
		return int64(len(moves))
	}
	var n int64
	for _, m := range moves {
		n += Perft(Position{pos: p.pos.Update(m)}, depth-1)
	}
	return n
}
