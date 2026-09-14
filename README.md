# chess-ia

Módulo de **IA** do jogo de xadrez contra o computador (trabalho de Engenharia de
Software). Recebe uma posição em [FEN](https://www.chessprogramming.org/Forsyth-Edwards_Notation)
e devolve um lance em notação UCI (`e2e4`, `e7e8q`).

A IA roda como **worker**: ela mesma consulta o backend perguntando se há lance a
calcular, calcula e devolve o resultado. Assim não é preciso hospedar um serviço público
para a IA.

- **Algoritmo:** minimax (busca clássica, não é machine learning).
- **Avaliação:** material (P=100, C=B=300, T=500, D=900, em centipeões).
- **Regras:** biblioteca [`notnil/chess`](https://github.com/notnil/chess), isolada no pacote `rules/`.

> Estado atual: demo parcial. A IA joga lances legais e acha táticas curtas, mas ainda
> é fraca (só enxerga material). Alfa-beta, ordenação de lances e tabelas de posição vêm a seguir.

---

## Pré-requisitos

- [Go](https://go.dev/dl/) **1.22 ou superior** (`go version` para conferir)
- Git

## Instalação

```sh
git clone https://github.com/CallMePeterD/chess-ia.git
cd chess-ia
go mod download
```

---

## 1. Rodar os testes

```sh
go test ./...
```

Todos os pacotes devem mostrar `ok`. Variações:

```sh
go test -v ./rules                  # mostra cada teste
go test -short ./...                # pula os perft mais pesados
go test -v -run TestPerft ./rules   # roda um teste específico
```

O que é testado:

| Pacote | Testes |
|---|---|
| `rules` | **perft** em 5 posições de referência (ex.: posição inicial, profundidade 4 = 197.281), promoção, xeque-mate, afogamento, FEN inválida |
| `eval` | contagem de material |
| `search` | captura grátis, mate em 1 (brancas e pretas), profundidade 2 enxerga recaptura |
| `worker` | fluxo completo contra o mock, respostas 204/409/422/401, retry com backend instável |

## 2. Harness: FEN → lance

Linha de comando para testar a IA sem backend.

```sh
# posição inicial, profundidade 2
go run ./harness

# mate em 1 (a IA deve responder a1a8)
go run ./harness -fen "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1" -dificuldade media

# validar o gerador de lances (última linha deve ser 197281)
go run ./harness -perft 4
```

Saída de exemplo:

```
posição:    6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1
avaliação:  200 (estática)
lance:      a1a8
score:      99999 (profundidade 2)
nós:        145 em 3ms
```

| Flag | Padrão | Descrição |
|---|---|---|
| `-fen` | posição inicial | posição a analisar (use aspas: a FEN tem espaços) |
| `-dificuldade` | — | `facil` (prof. 1), `media` (prof. 2), `dificil` (prof. 3) |
| `-depth` | `2` | profundidade manual, ignorada se `-dificuldade` for usada |
| `-perft` | `0` | em vez de buscar, conta posições até essa profundidade |

Dica: monte qualquer posição no [editor do lichess](https://lichess.org/editor) e copie a FEN.

## 3. Worker + mock do backend

O mock simula os endpoints do backend, então o ciclo completo roda sem o backend real.
Abra **dois terminais** e use o mesmo token nos dois.

**Terminal 1: mock do backend**

```sh
# Linux / macOS / Git Bash
export IA_TOKEN=segredo
go run ./cmd/mockbackend -selfplay -plies 20
```

```powershell
# Windows PowerShell
$env:IA_TOKEN = "segredo"
go run ./cmd/mockbackend -selfplay -plies 20
```

**Terminal 2: IA (worker)**

```sh
# Linux / macOS / Git Bash
export IA_TOKEN=segredo
go run ./cmd/worker -backend http://localhost:8080
```

```powershell
# Windows PowerShell
$env:IA_TOKEN = "segredo"
go run ./cmd/worker -backend http://localhost:8080
```

Com `-selfplay`, a IA joga contra si mesma e o mock mostra cada lance aceito:

```
mock do backend em :8080
selfplay req-1: b1a3  (avaliação 0)
selfplay req-2: b8a6  (avaliação 0)
...
```

Encerre com `Ctrl+C`.

**Flags do mock (`cmd/mockbackend`)**

| Flag | Padrão | Descrição |
|---|---|---|
| `-addr` | `:8080` | endereço de escuta |
| `-selfplay` | `false` | a IA joga uma partida contra si mesma; sem a flag, entrega 3 posições de exemplo |
| `-dificuldade` | `media` | dificuldade dos pedidos gerados |
| `-plies` | `40` | limite de meios-lances no selfplay |

**Worker (`cmd/worker`)**

| Flag / variável | Padrão | Descrição |
|---|---|---|
| `IA_TOKEN` (variável) | obrigatória | token compartilhado do canal interno |
| `-backend` ou `IA_BACKEND_URL` | `http://localhost:8080` | URL base do backend |
| `-interval` | `750ms` | intervalo de polling quando não há trabalho |

Para usar o **backend real**, basta apontar `-backend` para ele e usar o token combinado.

### Gerar executáveis

```sh
go build -o bin/ ./cmd/worker ./cmd/mockbackend ./harness
```

---

## Contrato com o backend

A IA consome dois endpoints, sempre com `Authorization: Bearer <token>`.

**`GET /internal/v1/moves/next`**: pega trabalho pendente

| Status | Significado |
|---|---|
| `204` | nada a calcular (a IA espera e tenta de novo) |
| `200` | `{ "gameId", "requestId", "fen", "dificuldade" }` |

**`POST /internal/v1/moves/result`**: entrega o lance

Corpo: `{ "gameId", "requestId", "move", "avaliacao" }`

| Status | Significado | O que a IA faz |
|---|---|---|
| `200` | aceito | segue para o próximo |
| `409` | `requestId` já resolvido/expirado | descarta |
| `422` | lance inválido | registra no log |
| `401`/`403` | token recusado | encerra o worker |
| `5xx` / erro de rede | falha transitória | tenta de novo, esperando cada vez mais |

Convenções:

- `move` em UCI; `dificuldade` ∈ `facil`, `media`, `dificil`.
- `avaliacao` em centipeões, do ponto de vista das **brancas** (positivo = brancas melhor).
  Mate vale ±(100000 − distância em meios-lances).
- Se a posição não tem lance legal (mate/afogamento) ou a dificuldade é desconhecida,
  a IA **não envia resultado**; o backend deve expirar o `requestId`.

O [`mock/mock.go`](mock/mock.go) é uma implementação de referência desse comportamento.

---

## Estrutura

```
rules/            adaptador de regras (único pacote que usa notnil/chess) + perft
eval/             função de avaliação
search/           minimax, mapeamento dificuldade → profundidade
worker/           cliente do contrato: polling, POST, token, retry
mock/             endpoints simulados do backend
harness/          CLI: FEN → lance, perft
cmd/worker/       executável do worker
cmd/mockbackend/  executável do mock
```

## Roadmap

- [x] Adaptador de regras + perft
- [x] Avaliação v1: material
- [x] Minimax
- [x] Mock dos endpoints e cliente worker
- [ ] Poda alfa-beta
- [ ] Ordenação de lances
- [ ] Avaliação v2: piece-square tables
- [ ] Avaliação v3: mobilidade
- [ ] Iterative deepening com limite de tempo
- [ ] Integração com o backend real
- [ ] Calibração (puzzles e comparação com Stockfish)

## Licença

[MIT](LICENSE)
