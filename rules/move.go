package rules

// Move é um lance empacotado num único número de 16 bits.
// O layout binário é o seguinte:
// Bits 0-5:   Casa de origem (0 a 63)
// Bits 6-11:  Casa de destino (0 a 63)
// Bits 12-15: Flags (Captura, Promoção, Roque, etc.)
type Move uint16

// Flags padronizadas (bit 3 indica promoção, bit 2 indica captura)
const (
	FlagQuiet        uint16 = 0 // Lance normal
	FlagDoublePawn   uint16 = 1 // Peão anda duas casas
	FlagKingCastle   uint16 = 2 // Roque pequeno
	FlagQueenCastle  uint16 = 3 // Roque grande
	FlagCapture      uint16 = 4 // Captura normal
	FlagEP           uint16 = 5 // Captura En Passant
	
	// Promoções simples
	FlagPromoKnight  uint16 = 8
	FlagPromoBishop  uint16 = 9
	FlagPromoRook    uint16 = 10
	FlagPromoQueen   uint16 = 11
	
	// Promoções com captura simultânea
	FlagPromoCapKnight uint16 = 12
	FlagPromoCapBishop uint16 = 13
	FlagPromoCapRook   uint16 = 14
	FlagPromoCapQueen  uint16 = 15
)

// NewMove constrói o número final combinando origem, destino e flag através de shifts.
func NewMove(from, to int, flag uint16) Move {
	return Move(uint16(from) | (uint16(to) << 6) | (flag << 12))
}

// From extrai a casa de origem aplicando uma máscara aos 6 primeiros bits (0x3F).
func (m Move) From() int {
	return int(m & 0x3F)
}

// To extrai a casa de destino deslocando 6 bits para a direita.
func (m Move) To() int {
	return int((m >> 6) & 0x3F)
}

// Flags extrai o código do lance deslocando 12 bits para a direita.
func (m Move) Flags() uint16 {
	return uint16(m >> 12)
}

// IsCapture devolve true se o lance envolver qualquer tipo de captura.
// Magia binária: se o bit com valor 4 estiver ligado, o lance é uma captura.
func (m Move) IsCapture() bool {
	return (m.Flags() & 4) != 0
}

// IsPromotion devolve true se um peão chegar à última fileira.
// Magia binária: se o bit com valor 8 estiver ligado, é uma promoção.
func (m Move) IsPromotion() bool {
	return (m.Flags() & 8) != 0
}

// UCI converte a representação binária numa string Universal Chess Interface (ex: "e2e4", "e7e8q").
// Esta função é vital para comunicar com o seu backend em Java e para os testes unitários passarem.
func (m Move) UCI() string {
	if m == 0 {
		return "" // Lance nulo (usado na Tabela de Transposição antes de encontrar o melhor lance)
	}
	
	from := m.From()
	to := m.To()
	
	// A matemática simples do xadrez: Coluna (File) é o resto de 8, Linha (Rank) é a divisão por 8.
	fileFrom := byte('a' + (from % 8))
	rankFrom := byte('1' + (from / 8))
	fileTo := byte('a' + (to % 8))
	rankTo := byte('1' + (to / 8))
	
	uci := string([]byte{fileFrom, rankFrom, fileTo, rankTo})
	
	// Anexa o sufixo da peça caso seja uma promoção
	if m.IsPromotion() {
		switch m.Flags() & 3 { // Mascara os últimos 2 bits para descobrir qual é a peça promovida
		case 0: uci += "n"
		case 1: uci += "b"
		case 2: uci += "r"
		case 3: uci += "q"
		}
	}
	
	return uci
}