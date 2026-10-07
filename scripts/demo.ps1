# Demonstracao da IA para a apresentacao.
#   .\scripts\demo.ps1          roteiro completo, passo a passo (ENTER entre cada etapa)
#   .\scripts\demo.ps1 -Passo 3 executa so a etapa 3
param([int]$Passo = 0)

$ErrorActionPreference = "Stop"
$raiz = Split-Path -Parent $PSScriptRoot
$bin = Join-Path $raiz "bin"
$env:IA_TOKEN = "segredo"

function Titulo($n, $texto) {
    Write-Host ""
    Write-Host ("  [$n] $texto") -ForegroundColor Cyan
    Write-Host ("  " + ("-" * 70)) -ForegroundColor DarkGray
}

function Comando($cmd) {
    Write-Host "  > $cmd" -ForegroundColor Yellow
    Write-Host ""
}

function Pausa() {
    if ($Passo -ne 0) { return }
    Write-Host ""
    Write-Host "  [ENTER para a proxima etapa]" -ForegroundColor DarkGray
    Read-Host | Out-Null
}

# Garante os binarios prontos (evita compilar na frente da turma)
if (-not (Test-Path "$bin\harness.exe")) {
    Write-Host "Compilando..." -ForegroundColor DarkGray
    & go build -o "$bin\" ./harness ./cmd/worker ./cmd/mockbackend
}

Clear-Host
Write-Host ""
Write-Host "  IA DE XADREZ - Grupo 7 - demonstracao" -ForegroundColor White
Write-Host "  Go 1.22 | busca classica (minimax com poda alfa-beta)" -ForegroundColor DarkGray

# ---------------------------------------------------------------- 1
if ($Passo -in 0, 1) {
    Titulo 1 "As regras estao corretas: teste de perft"
    Comando "harness -perft 5"
    & "$bin\harness.exe" -perft 5
    Write-Host ""
    Write-Host "  Conta todas as sequencias de lances possiveis e compara com o" -ForegroundColor DarkGray
    Write-Host "  gabarito publicado: 4.865.609 posicoes. Se bate, roque, en passant" -ForegroundColor DarkGray
    Write-Host "  e promocao estao corretos." -ForegroundColor DarkGray
    Pausa
}

# ---------------------------------------------------------------- 2
if ($Passo -in 0, 2) {
    Titulo 2 "Dada uma posicao, a IA devolve o lance: mate em 1"
    Comando 'harness -fen "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1" -dificuldade media'
    & "$bin\harness.exe" -fen "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1" -dificuldade media
    Write-Host ""
    Write-Host "  a1a8 e o mate na ultima fileira. Score perto de 100000 = mate." -ForegroundColor DarkGray
    Pausa
}

# ---------------------------------------------------------------- 3
if ($Passo -in 0, 3) {
    Titulo 3 "Tatica: a IA sacrifica a torre para dar mate"
    Comando 'harness -fen "6k1/pp4p1/2p5/2bp4/8/P5Pb/1P3rrP/2BRRN1K b - - 0 1" -depth 4'
    & "$bin\harness.exe" -fen "6k1/pp4p1/2p5/2bp4/8/P5Pb/1P3rrP/2BRRN1K b - - 0 1" -depth 4
    Write-Host ""
    Write-Host "  g2g1 entrega a torre de graca. A IA ve que a recaptura e forcada" -ForegroundColor DarkGray
    Write-Host "  e que o mate vem no lance seguinte. Score -99997 = mate das pretas." -ForegroundColor DarkGray
    Pausa
}

# ---------------------------------------------------------------- 4
if ($Passo -in 0, 4) {
    Titulo 4 "A IA conversa com o backend: partida completa contra o simulador"
    Comando "mockbackend -selfplay   |   worker"
    Write-Host "  Abrindo duas janelas: o simulador do backend e a IA." -ForegroundColor DarkGray
    Write-Host "  Acompanhe a janela do simulador: cada linha e um lance aceito." -ForegroundColor DarkGray

    $mock = Start-Process -FilePath "$bin\mockbackend.exe" `
        -ArgumentList "-selfplay", "-plies", "24" -PassThru
    Start-Sleep -Milliseconds 800
    $worker = Start-Process -FilePath "$bin\worker.exe" -PassThru

    Write-Host ""
    Write-Host "  [ENTER para encerrar as duas janelas]" -ForegroundColor DarkGray
    Read-Host | Out-Null
    foreach ($p in @($mock, $worker)) {
        if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
    }
}

# ---------------------------------------------------------------- 5
if ($Passo -in 0, 5) {
    Titulo 5 "Rede de protecao: a suite de testes"
    Comando "go test ./..."
    & go test ./...
    Write-Host ""
    Write-Host "  19 testes: regras (perft), avaliacao, busca e comunicacao." -ForegroundColor DarkGray
    Write-Host "  Os mesmos que rodam no GitHub Actions a cada push e cada PR." -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "  Fim da demonstracao." -ForegroundColor White
Write-Host ""
