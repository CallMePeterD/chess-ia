package rules

import (
	"errors"
	"strconv"
	"strings"
)

// StartFEN é a posição inicial padrão do xadrez.
const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// FromFEN analisa uma string FEN e converte-a na nossa estrutura de Bitboards.
func FromFEN(fen string) (Position, error) {
	var p Position
	parts := strings.Split(strings.TrimSpace(fen), " ")
	if len(parts) < 4 {
		return p, errors.New("FEN inválido: não tem as partes suficientes")
	}

	// 1. Posicionamento das Peças
	rank, file := 7, 0 // O FEN começa na linha 8 (índice 7) e coluna A (índice 0)
	for _, char := range parts[0] {
		if char == '/' {
			rank--
			file = 0
			continue
		}
		if char >= '1' && char <= '8' {
			file += int(char - '0') // Casas vazias: avança a coluna
			continue
		}

		sq := rank*8 + file
		var color Color = Black
		if char >= 'A' && char <= 'Z' {
			color = White
		}

		var pt PieceType
		switch strings.ToLower(string(char)) {
		case "p":
			pt = Pawn
		case "n":
			pt = Knight
		case "b":
			pt = Bishop
		case "r":
			pt = Rook
		case "q":
			pt = Queen
		case "k":
			pt = King
		default:
			return p, errors.New("peça inválida detetada no FEN")
		}

		// Ativa o bit 1 na casa exata do tabuleiro da peça e da cor
		p.Pieces[pt].Set(sq)
		p.Colors[color].Set(sq)
		file++
	}

	// 2. Lado a jogar
	if parts[1] == "w" {
		p.SideToMove = White
	} else {
		p.SideToMove = Black
	}

	// 3. Direitos de Roque (Castling)
	p.Castling = 0
	if parts[2] != "-" {
		if strings.ContainsRune(parts[2], 'K') {
			p.Castling |= 1
		}
		if strings.ContainsRune(parts[2], 'Q') {
			p.Castling |= 2
		}
		if strings.ContainsRune(parts[2], 'k') {
			p.Castling |= 4
		}
		if strings.ContainsRune(parts[2], 'q') {
			p.Castling |= 8
		}
	}

	// 4. Casa de En Passant
	p.EnPassant = -1
	if parts[3] != "-" && len(parts[3]) == 2 {
		f := int(parts[3][0] - 'a')
		r := int(parts[3][1] - '1')
		p.EnPassant = r*8 + f
	}

	// 5 e 6. Regra dos 50 lances e número do lance
	if len(parts) >= 6 {
		p.Halfmove, _ = strconv.Atoi(parts[4])
		p.Fullmove, _ = strconv.Atoi(parts[5])
	} else {
		p.Fullmove = 1
	}

	// Gera a chave Hash instantaneamente com a nova configuração
	p.HashKey = p.ComputeHash()

	return p, nil
}

// fenChars mapeia o tipo da peça para a letra usada no FEN (minúscula = pretas).
var fenChars = [6]byte{Pawn: 'p', Knight: 'n', Bishop: 'b', Rook: 'r', Queen: 'q', King: 'k'}

// PieceAt devolve o tipo e a cor da peça numa casa. Se a casa estiver vazia,
// devolve NoPiece (a cor não tem significado nesse caso).
func (p *Position) PieceAt(sq int) (PieceType, Color) {
	color := White
	if p.Colors[Black].Has(sq) {
		color = Black
	} else if !p.Colors[White].Has(sq) {
		return NoPiece, White
	}
	for pt := Pawn; pt <= King; pt++ {
		if p.Pieces[pt].Has(sq) {
			return pt, color
		}
	}
	return NoPiece, White
}

// FEN reconverte os Bitboards de volta para uma string FEN padrão.
// É o inverso de FromFEN: FromFEN(p.FEN()) devolve a mesma posição.
func (p *Position) FEN() string {
	var sb strings.Builder

	// 1. Posicionamento das peças, da linha 8 (índice 7) para a linha 1.
	for rank := 7; rank >= 0; rank-- {
		empty := 0
		for file := 0; file < 8; file++ {
			pt, color := p.PieceAt(rank*8 + file)
			if pt == NoPiece {
				empty++
				continue
			}
			if empty > 0 {
				sb.WriteString(strconv.Itoa(empty))
				empty = 0
			}
			char := fenChars[pt]
			if color == White {
				char -= 'a' - 'A' // maiúscula para as brancas
			}
			sb.WriteByte(char)
		}
		if empty > 0 {
			sb.WriteString(strconv.Itoa(empty))
		}
		if rank > 0 {
			sb.WriteByte('/')
		}
	}

	// 2. Lado a jogar.
	if p.SideToMove == White {
		sb.WriteString(" w ")
	} else {
		sb.WriteString(" b ")
	}

	// 3. Direitos de roque, sempre na ordem KQkq.
	if p.Castling == 0 {
		sb.WriteByte('-')
	} else {
		for i, flag := range []uint8{1, 2, 4, 8} {
			if p.Castling&flag != 0 {
				sb.WriteByte("KQkq"[i])
			}
		}
	}

	// 4. Casa de en passant.
	sb.WriteByte(' ')
	if p.EnPassant < 0 || p.EnPassant > 63 {
		sb.WriteByte('-')
	} else {
		sb.WriteByte(byte('a' + p.EnPassant%8))
		sb.WriteByte(byte('1' + p.EnPassant/8))
	}

	// 5 e 6. Relógio dos 50 lances e número do lance.
	fullmove := p.Fullmove
	if fullmove < 1 {
		fullmove = 1
	}
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(p.Halfmove))
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(fullmove))

	return sb.String()
}
