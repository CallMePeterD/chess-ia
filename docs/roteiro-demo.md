# Roteiro da demo — IA (2ª apresentação, 07/10/2026)

Duração: cerca de 3 minutos. Tudo roda **local**, sem internet e sem o backend.

## Antes de sair de casa

- [ ] `go build -o bin/ ./harness ./cmd/worker ./cmd/mockbackend` (já feito; refazer se mudar o código)
- [ ] Rodar `.\scripts\demo.ps1` uma vez inteira, para conferir que tudo abre
- [ ] Aumentar a fonte do terminal (`Ctrl` + `Shift` + `+` no Windows Terminal)
- [ ] Gravar um vídeo curto da demo rodando — é o plano B se a máquina travar
- [ ] Deixar o PDF do deck aberto numa aba

## Como rodar

```powershell
.\scripts\demo.ps1            # roteiro completo, ENTER entre as etapas
.\scripts\demo.ps1 -Passo 3   # pula direto para uma etapa
```

## As cinco etapas

### 1. Perft — as regras estão corretas

```
perft(5) = 4865609  (300ms)
```

> "Antes de falar em inteligência, a base tem que estar certa. O perft conta todas as
> sequências de lances possíveis até certa profundidade e compara com valores publicados
> pela comunidade. Quase 5 milhões de posições, e bate exatamente — inclusive roque,
> en passant e promoção."

### 2. Mate em 1 — a IA devolve um lance

```
lance: a1a8   score: 99999   nós: 33 em 1ms
```

> "A interface da IA é essa: entra uma posição, sai um lance. Aqui ela achou o mate na
> última fileira olhando 33 posições, em 1 milissegundo."

### 3. Sacrifício de torre — o ponto alto

```
lance: g2g1   score: -99997   nós: 11955 em 19ms
```

> "Esta é a posição mais interessante. A IA joga a torre para g1, onde ela é capturada de
> graça. Materialmente é um desastre, mas a busca vê que a recaptura é forçada e que o
> mate vem no lance seguinte. Doze mil posições analisadas em 19 milésimos."

### 4. Worker contra o simulador — a integração

Abrem duas janelas: o simulador do backend e a IA. A partida roda sozinha.

> "A IA roda como worker: ela pergunta ao backend se há lance a calcular, calcula e
> devolve. Como o backend ainda não expõe esses endpoints, escrevi um simulador que
> implementa o mesmo contrato. Cada linha aqui é um lance que a IA calculou e o
> simulador aceitou, validando a legalidade."

### 5. Testes — a rede de proteção

```
ok  rules | eval | search | worker | book
```

> "São 19 testes automatizados, os mesmos que rodam no GitHub Actions a cada push e a
> cada pull request."

## Se der problema

| Problema | O que fazer |
|---|---|
| Porta 8080 ocupada (etapa 4) | `.\bin\mockbackend.exe -addr :9090 -selfplay` e `.\bin\worker.exe -backend http://localhost:9090` |
| Firewall pedindo permissão | Cancelar e seguir: a conexão é local (localhost) e funciona mesmo negando |
| Aviso de livro de aberturas | Não aparece mais; o livro é opcional e silencioso |
| Projetor com fonte pequena | `.\scripts\demo.ps1 -Passo N` mostra uma etapa por vez |
| Máquina travou | Usar o vídeo gravado e os números do slide 12 |

## Perguntas prováveis da banca

**"Isso é inteligência artificial de verdade? Usa machine learning?"**
Não usa aprendizado de máquina. É busca clássica — minimax com poda alfa-beta, a mesma
família do Deep Blue. A IA simula os lances, as respostas do adversário, e escolhe o
caminho cujo pior desfecho é o melhor possível.

**"Como vocês sabem que a IA não sugere lance ilegal?"**
Por dois caminhos independentes: o perft valida a geração de lances contra valores
conhecidos, e o backend revalida todo lance antes de aplicar. Se a IA errar, o lance é
recusado com o código `ILLEGAL_MOVE`.

**"Qual a força de jogo dela?"**
Ainda não medimos, e é a próxima tarefa da frente (EAP 4.6): montar um corpus de puzzles
e comparar com o Stockfish como oráculo. Hoje sabemos que ela acha táticas curtas e mates
forçados, mas sem corpus não dá para afirmar um nível.

**"Por que Go, se o backend é Java?"**
A IA é um processo separado que conversa por HTTP e JSON, então a linguagem de um lado não
aparece para o outro. Go dá bom desempenho na busca e tem concorrência e testes na
biblioteca padrão.

**"Por que a IA chama o backend, e não o contrário?"**
Para não precisar hospedar um serviço público. A IA roda na máquina local e só faz
requisições de saída, como um navegador.
