# Entrega de séries agrupadas

## Execution Plan

```mermaid
flowchart LR
  T1 --> T2 --> T3
```

## Task Breakdown

### T1: agrupar séries na consulta
Where: api/internal/library e api/internal/httpapi
Depends on: none
Tests: integração PostgreSQL de agrupamento/filtros/paginação/notas e contratos brutos existentes
Gate: Full
- [x] Adicionar group_series opcional, resumir por identidade antes do cursor e retornar participantes/quantidade; testes com mesmo nome e identidades distintas, mais de uma página e série sem TMDB. SAG-01, AC1–3/6. TestSeriesGroupingAndFilters falhou com 400 antes da implementação; quatro testes novos passaram após implementar. Full Go passou com o frontend anterior, preservando contratos brutos e regressões.

### T2: apresentar série e avaliar obra inteira
Where: web e testes Chromium
Depends on: T1
Tests: Chromium de série única, filtros, nota zero/edição/remoção/pendentes, regressões existentes adaptadas ao novo contrato do app
Gate: Build
- [x] Solicitar group_series, apresentar participantes/última assistida/quantidade e explicar nota da série; atualizar expectativas de contagem afetadas sem enfraquecer regressões. SAG-02, AC4–5. TestBrowserGroupedSeriesAndRatings falhou antes da mudança por exibir os dois episódios; passou após implementar. Build web e Full/Quick/vet/build Go passaram. Regressões passam com 5 itens em vez de 6 e 60 em vez de 61, pois Série A agora ocupa uma entrada. Asserções de notas, privacidade, filtros e foco preservadas; captura de 320 px inspecionada.

### T3: documentar e atualizar app local
Where: documentação e operação local
Depends on: T2
Tests: build Compose API/web e smoke/preservação de dados; validadores de spec/tasks
Gate: Build
- [x] Documentar escolhas em docs/app.md e .specs/STATE.md, atualizar somente API/web sem migrations e validar preservação. SAG-03, AC7. Compose build/up --no-deps --wait passou, prontidão e assets conferidos. Snapshot em memória preservou contas/credenciais, sessões, 1248 eventos, 142 títulos e zero notas; demais containers mantidos. docs/netflix.md corrige o estado antigo do TMDB após validação real anterior. Revisão independente após este commit, PR/CI antes de marcar pronto.

## Test Coverage Matrix

| Camada | Resultado esperado |
| --- | --- |
| API/Postgres | Itens e totais exatos com filtros antes da agregação, paginação sem série repetida, filmes brutos preservados, nomes iguais separados |
| Notas | Mesma chave por série entre episódios/pessoas, zero/média/contador e remoção independentes |
| Browser | Uma série visível, pessoas reais, sem episódio, contrato de nota explícito e filtros/pendentes funcionais |
| Operação | Mesmos dados/identidades antes/depois; somente API/web recriadas |

## Gate Check Commands

- Full: testes Go de integração serial com banco sintético 55432, incluindo contratos brutos existentes.
- Build: npm --prefix web run build e .local/checks.py full quick vet build.
- Antes dos commits: validate_spec.py, validate_tasks.py, check_commit.py e git diff --check.
- Fechamento: Verifier independente após último commit, validate_state.py e CI verde no HEAD do PR.
