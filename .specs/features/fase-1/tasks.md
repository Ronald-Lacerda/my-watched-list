# Tarefas da Fase 1

## Test Coverage Matrix

| Camada | Verificação |
| --- | --- |
| Sessão | Testes unitários de cifragem, nonce, contexto, adulteração e configuração |
| Normalização | Fixtures sintéticas, datas, ordem, tradução, vazio e erros |
| Navegador | Integração com site local controlado em Chromium, perfil e paginação |
| Persistência | PostgreSQL real; replay, concorrência, rollback e status |
| TMDB | Servidor HTTP controlado, match, ambiguidade e falhas |
| Agenda/CLI | Testes de orquestração e comandos sem rede real |
| Deploy | Compose/build/smoke e CI |

## Gate Check Commands

Quick: `go test ./...` e `go vet ./...`.
Full: `go test -tags=integration -count=1 ./...` com TEST_DATABASE_URL descartável; testes browser usam Chromium instalado.
Build: Quick + Full + `go build ./...`, `npm --prefix web run build`, Compose build/up e checks HTTP.

## Execution Plan

Uma entrega por componente, executada localmente em sequência; sete tarefas, sem delegação de implementação.

```text
T1 -> T2 -> T3 -> T4 -> T5 -> T6 -> T7
```

## Task Breakdown

### T1: sessões criptografadas
Where: internal/session
Depends on: none
Tests: unit
Gate: Quick
- [x] Cifrar, autenticar contexto da conexão e validar chave; NFLX-01. go test/vet aprovados; internal/session/cipher_test.go verifica igualdade do plaintext recuperado, nonce distinto e rejeição de contexto, chave, versão e ciphertext alterados. Integração da configuração no processo em T6.

### T2: normalizador Netflix
Where: worker/netflix
Depends on: T1
Tests: unit
Gate: Quick
- [x] Lote estrito, chave estável e datas corretas; NFLX-02. TestNormalize preserva campos e confere virada UTC/dia local, tradução e ordem. TestInvalidPayloadIsAtomic rejeita lote parcial, IDs/datas inválidos e schema ausente; vazio explícito aceito. go test/vet aprovados. Corrigido literal horário UTC da fixture sintética durante criação do teste, mantendo comparação exata e adicionando caso de virada do dia.

### T3: adaptador Playwright
Where: worker/netflix
Depends on: T2
Tests: integration browser
Gate: Full
- [x] Login assistido, perfil, paginação e falhas categorizadas; NFLX-03. TestBrowserCapture passou em Chromium nos cinco cenários: duas páginas, perfil errado, schema alterado, página repetida e redirecionamento ao login. go test/vet aprovados. Contrato real confirmado em 18/10/2026: vaModel.data inicial e Falcor na paginação, botão desabilitado ao terminar, vhSize conferido. R: 557 eventos; Gabriela: 691. TestBrowserFalcor acrescenta sucesso, vazio, perfil incorreto na paginação, parser, limite e total incompleto. TestUnavailableTitle cobre decisão explícita de preservar título em branco. A fixture continua sintética; nenhum histórico/cookie real está versionado.

### T4: repositório de coleta
Where: worker/store
Depends on: T3
Tests: integration postgres
Gate: Full
- [x] Migration de pistas de enriquecimento, lock, transação, status e replay concorrente; NFLX-04. TestPersistence em PostgreSQL Docker aprovado: round-trip cifrado, replay count=1, lock concorrente, revisita count=2, rollback de lote parcial e last_sync_at inalterado na falha. Suite completa com integration aprovada.

### T5: enriquecimento TMDB
Where: worker/tmdb
Depends on: T4
Tests: unit HTTP e integração postgres
Gate: Full
- [x] Match conservador, metadados, episódios e pendências; NFLX-05. TestMatch cobre filme, série/episódio, ambiguidade inclusive em segunda página, vazio e HTTP 429. TestPersistence verifica título/sinopse/pôster/gêneros vinculados, episódio e retry durável sem vínculo indevido. go test e vet aprovados.

### T6: comandos e agenda
Where: worker/cmd/worker
Depends on: T5
Tests: unit e integração
Gate: Full
- [x] Bootstrap, login, status, sync e agenda com contexto; NFLX-06. TestPipeline aprovado em PostgreSQL: normalizador → cifragem → sync → TMDB/vínculo, replay, falha TMDB seguida de retry, expiração, parser e erro de driver sem segredo. TestDailySchedule confere 04:00 e TestInvalidKeyStopsBeforeNetwork exige falha antes da rede. Suite integration e vet aprovados. Login real concluído em Chrome para R e Gabriela; GUID canônico lido na atividade.

### T7: ambiente e operação
Where: deploy
Depends on: T6
Tests: build e smoke
Gate: Build
- [x] Imagem Chromium, variáveis, CI, documentação e Docker recuperado; NFLX-07. Docker Desktop atualizado de 4.68.0 para 4.95.0 pelo instalador oficial assinado; Engine 29.8.2 e hello-world aprovados. Build Go/web e stack Compose aprovados, API healthz/readyz HTTP 200. Screenshot Chromium executado dentro do worker como pwuser. Sessões reais funcionaram também no worker Docker. Sem credencial TMDB do usuário: testes HTTP/PostgreSQL aprovados; chamada real permanece pendente.

### Revisão de adequação da entrega

| Critério | Evidência e resultado exigido | Resultado |
| --- | --- | --- |
| AC 3–4 | worker/netflix/normalize_test.go:69 compara SourceTitle, SourceTitleID, SourceKey, MatchTitle e instante exatos; teste de null continua exigindo ErrParserChanged/lote nil | Aprovado; título em branco alterado por decisão expressa do usuário |
| AC 5–6 | worker/netflix/browser_integration_test.go:107 exige dois IDs distintos e estado; :122 exige erro categorizado e lote nil em perfil, parser, limite e total incompleto | Aprovado; Chromium com SSR/Falcor sintético |
| AC 13 | .github/workflows/checks.yml executa stack/readyz e screenshot Chromium como usuário da imagem; deploy/Dockerfile.backend fixa versão compatível | Smoke local passou com exit 0 e HTTP 200 |

Os testes novos verificam AC 3–6 e os comandos de smoke verificam AC 13; nenhum teste foi removido ou ignorado. A normalização de título em branco mudou após confirmação do usuário, preservando teste de campo ausente e acrescentando teste de null. Verificação independente é o próximo gate.

Dependências do diagrama coincidem com cada Depends on. Todos os componentes possuem testes na mesma tarefa. Usuário autorizou implementação, testes e PR; decisões simples seguem os padrões documentados, sem nova aprovação de ferramentas.

## Correções da revisão independente

- [x] F1, AC 5/9–10: testes Chromium exigem cookie sintético por conexão e rejeitam estado ausente ou de outro perfil. Testes TMDB rejeitam nome inexato, ID/temporada/episódio incompatíveis e ambiguidade entre mídias; aceitam título original e match único entre tipos. A célula de decisão da spec foi alinhada ao comportamento já documentado para tipo desconhecido. Gate completo de integração e vet aprovado; revalidação independente registrada abaixo.
- [x] F2, AC 11–12: CLI usa fronteira browser substituível no teste, mantendo PostgreSQL, cifragem e orquestração reais. TestAdministrativeCLI exige user-add com !disabled, login/reconexão cifrados, status sem cookies e sync idempotente. Serve testa startup, seleção active/error, exclusão needs_login/disabled, ciclos recorrentes, cancelamento e liberação do advisory lock. A agenda cron real continua coberta pelo próximo horário diário; o loop interno recebe cron.Schedule para executar recorrência controlada sem aguardar um dia. Gate de integração e vet aprovado; revalidação independente registrada abaixo.

Revalidação independente em 9fbbd0f: PASS, 75 testes/subtestes sem falha ou teste ignorado e 11/11 mutações detectadas, incluindo as cinco que sobreviveram na primeira rodada. Isolamento confirmado, sem mudanças no código/testes reais pelo verificador. Rebuild da stack final, readyz HTTP 200 e Chromium no worker aprovados. Coleta real do worker final preservou 557 eventos de R e 691 de Gabriela. O relatório mantém o histórico das lacunas e das correções. A chamada TMDB real permanece pendente do token do usuário.
