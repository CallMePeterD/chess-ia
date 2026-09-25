package rules

// Apply executa o lance, lida com regras especiais e atualiza o Zobrist Hash progressivamente.
func (p Position) Apply(m Move) Position {
	newP := p
	from := m.From()
	to := m.To()
	flags := m.Flags()
	us := p.SideToMove
	them := 1 ^ us

	// 1. Descobre a peça que se vai mover
	var movingPiece PieceType = NoPiece
	for pt := Pawn; pt <= King; pt++ {
		if newP.Pieces[pt].Has(from) && newP.Colors[us].Has(from) {
			movingPiece = pt
			break
		}
	}

	// 2. Resolve Capturas (incluindo a peculiaridade do En Passant)
	var capturedPiece PieceType = NoPiece
	var capSquare int = to

	if m.IsCapture() {
		if flags == FlagEP {
			capturedPiece = Pawn
			if us == White { capSquare = to - 8 } else { capSquare = to + 8 }
		} else {
			for pt := Pawn; pt <= King; pt++ {
				if newP.Pieces[pt].Has(to) && newP.Colors[them].Has(to) {
					capturedPiece = pt
					break
				}
			}
		}
	}

	// 3. Levanta a peça original
	newP.Pieces[movingPiece].Clear(from)
	newP.Colors[us].Clear(from)
	newP.HashKey ^= ZobristPieces[us][movingPiece][from]

	// 4. Remove a peça capturada do tabuleiro e do Hash
	if capturedPiece != NoPiece {
		newP.Pieces[capturedPiece].Clear(capSquare)
		newP.Colors[them].Clear(capSquare)
		newP.HashKey ^= ZobristPieces[them][capturedPiece][capSquare]
	}

	// 5. Pousa a peça no destino (lidando com Promoções)
	if m.IsPromotion() {
		var promoPiece PieceType
		switch flags & 3 {
		case 0: promoPiece = Knight
		case 1: promoPiece = Bishop
		case 2: promoPiece = Rook
		case 3: promoPiece = Queen
		}
		newP.Pieces[promoPiece].Set(to)
		newP.Colors[us].Set(to)
		newP.HashKey ^= ZobristPieces[us][promoPiece][to]
	} else {
		newP.Pieces[movingPiece].Set(to)
		newP.Colors[us].Set(to)
		newP.HashKey ^= ZobristPieces[us][movingPiece][to]
	}

	// 6. Atualiza o relógio da Regra dos 50 lances
	if movingPiece == Pawn || capturedPiece != NoPiece {
		newP.Halfmove = 0
	} else {
		newP.Halfmove++
	}

	// 7. Atualiza os metadados (En Passant e Direitos de Roque)
	if newP.EnPassant != -1 {
		newP.HashKey ^= ZobristEnPassant[newP.EnPassant]
		newP.EnPassant = -1
	}
	
	if flags == FlagDoublePawn {
		if us == White { newP.EnPassant = to - 8 } else { newP.EnPassant = to + 8 }
		newP.HashKey ^= ZobristEnPassant[newP.EnPassant] // Grava o novo alvo no Hash
	}

	// Invalida os direitos de Roque se Rei ou Torre moverem/forem capturados
	oldCastling := newP.Castling
	removeCastle := func(sq int) {
		if sq == 0 { newP.Castling &= ^uint8(2) }   // Torre A1
		if sq == 7 { newP.Castling &= ^uint8(1) }   // Torre H1
		if sq == 4 { newP.Castling &= ^uint8(3) }   // Rei E1
		if sq == 56 { newP.Castling &= ^uint8(8) }  // Torre A8
		if sq == 63 { newP.Castling &= ^uint8(4) }  // Torre H8
		if sq == 60 { newP.Castling &= ^uint8(12) } // Rei E8
	}
	removeCastle(from)
	removeCastle(to)
	if oldCastling != newP.Castling {
		newP.HashKey ^= ZobristCastling[oldCastling]
		newP.HashKey ^= ZobristCastling[newP.Castling]
	}

	// 8. Se for Roque, o Rei já se moveu. Falta arrastar a Torre!
	if flags == FlagKingCastle {
		var rookFrom, rookTo int
		if us == White { rookFrom, rookTo = 7, 5 } else { rookFrom, rookTo = 63, 61 }
		newP.Pieces[Rook].Clear(rookFrom)
		newP.Colors[us].Clear(rookFrom)
		newP.HashKey ^= ZobristPieces[us][Rook][rookFrom]

		newP.Pieces[Rook].Set(rookTo)
		newP.Colors[us].Set(rookTo)
		newP.HashKey ^= ZobristPieces[us][Rook][rookTo]
	} else if flags == FlagQueenCastle {
		var rookFrom, rookTo int
		if us == White { rookFrom, rookTo = 0, 3 } else { rookFrom, rookTo = 56, 59 }
		newP.Pieces[Rook].Clear(rookFrom)
		newP.Colors[us].Clear(rookFrom)
		newP.HashKey ^= ZobristPieces[us][Rook][rookFrom]

		newP.Pieces[Rook].Set(rookTo)
		newP.Colors[us].Set(rookTo)
		newP.HashKey ^= ZobristPieces[us][Rook][rookTo]
	}

	// 9. Troca a vez
	if us == Black {
		newP.Fullmove++
	}
	newP.SideToMove = them
	newP.HashKey ^= ZobristSide

	return newP
}