# Tarefas da Fase 2

## Test Coverage Matrix

Diretrizes: RTK.md, escala e invariantes do schema, testes Go/PostgreSQL/Chromium existentes. Nenhum teste existente será removido ou enfraquecido.

| Camada | Testes exigidos | Evidência esperada |
| --- | --- | --- |
| Schema/identidade | unit + PostgreSQL | Preservação, chave da série, notas sem TMDB, replay enriquecido |
| Auth | unit + PostgreSQL | Hash/salt/bounds, convite concorrente, expiração/revogação/throttle |
| HTTP | integração real | Cada rota: sucesso, entrada inválida, sessão ausente e Origin incorreto |
| Consultas/notas | PostgreSQL | Filtros inclusivos, cursor, zero, média, independência e concorrência |
| React | Chromium + API/PostgreSQL reais | Entrada, filtros, salvar/editar/excluir, outra pessoa, recarga, mobile e teclado |
| PWA/deploy | browser e smoke | Manifest, SW sem dados privados, offline, Compose e CI |

## Gate Check Commands

Quick: `go test ./...` e `go vet ./...`.
Full: `go test -tags=integration -count=1 ./...`, com TEST_DATABASE_URL descartável; após T6 também executar `npm --prefix web run build` antes do Full para disponibilizar a UI nos testes Chromium.
Build: Quick + Full + `go build ./...`, `npm --prefix web run build`, gofmt e Compose build/up/HTTP/browser.

## Execution Plan

Oito entregas coesas, implementadas em sequência local. O usuário já autorizou a implementação e as decisões simples; ferramentas normais do projeto são mantidas. Verifier independente obrigatório após o último commit.

```text
T1 -> T2 -> T3 -> T4 -> T5 -> T6 -> T7 -> T8
```

## Task Breakdown

### T1: identidade própria para avaliações
Where: api/migrations
Depends on: none
Tests: unit e integration postgres
Gate: Full
- [x] Migration aditiva e source_work_id no coletor; preservar eventos/notas e identidade de série no replay. APP-01. Full/Quick/vet passaram com GOFLAGS=-p=1 e GOMAXPROCS=2 após limite de memória virtual do Windows na tentativa paralela. TestMigrationPreservesData mantém nota 8.5/comentário e evento antigo e aceita nota zero sem TMDB; TestWorkIdentity compara IDs 101/200/200; TestWorkReplayPreservesRating exige count=1, work=200, title_id preservado e nota 9 intacta após replay enriquecido. Nenhuma asserção anterior foi alterada.

### T2: ciclo de credenciais e sessões
Where: internal/auth
Depends on: T1
Tests: unit e integration postgres
Gate: Full
- [x] Hash, sessões, convite de primeiro acesso/CLI e throttle persistido com testes de expiração, concorrência e revogação. APP-02. Full/Quick/vet aprovados: TestPasswords compara hash/salt, senha errada, formato e limites Unicode; TestCredentialLifecycle confere TTL, hash persistido, revogação/logout/reset; TestInvitationConcurrencyAndExpiry exige exatamente um sucesso e uma rejeição; TestLoginThrottle exige 401 até a décima tentativa, 429 na 11ª e reset de janela/sucesso. TestInvitationCommand usa o link gerado, confere saída sem token e impede sobrescrita de arquivo.

### T3: fronteira HTTP autenticada
Where: api/internal/httpapi
Depends on: T2
Tests: integration HTTP postgres
Gate: Full
- [x] Rotas de auth, cookie, Origin e JSON limitado; endpoints privados não vazam dados sem sessão. APP-03. Full/Quick/vet passaram. TestAuthHTTP verifica cookie HttpOnly/SameSite/Path/TTL, identidade, logout e 401 sem sessão/revogada/expirada; TestHTTPOriginAndJSON exige 403 em login/ativação/logout cross-origin, 400/415/413 na decodificação e 429/Retry-After no limite; TestHTTPSCookieAndOriginConfiguration exige Secure e rejeita configuração inválida.

### T4: leitura do histórico filtrado
Where: api/internal/library
Depends on: T3
Tests: integration HTTP postgres
Gate: Full
- [x] Histórico, filtros e conexões com paginação estável, período inclusivo e payload sanitizado. APP-04. Full/Quick/vet passaram. TestHistoryFiltersAndPagination exige seis eventos, filtros combinados, limites inclusivos, vazio e seis páginas sem repetição/perda em datas iguais; query inválida exige 400. TestReadEndpointsPrivacy exige 401 sem sessão, nomes/serviços/needs_login corretos e ausência de ciphertext, hashes e e-mails em todos os endpoints de leitura.

### T5: notas individuais por obra
Where: api/internal/library
Depends on: T4
Tests: integration HTTP postgres
Gate: Full
- [x] Upsert/delete por pessoa autenticada, validação, zero, média e pendentes individuais sem TMDB. APP-05. Full/Quick/vet aprovados. TestIndividualRatingsAndPending exige média 5 com 0/10, média 9.25 após editar, nota da parceira intacta, uma nota por série e pendentes pessoais. TestRatingValidationAndAuthorization exige 400/401/403/404/409 e preservação da nota 6; TestRatingConcurrentUpsertAndLegacy verifica concorrência e compatibilidade sem ressuscitar nota removida. TestRatingRollbackOnFailure provoca erro depois do upsert e exige rollback integral e 500 sanitizado. TestRatingBounds cobre 0/0.5/10, NaN/Inf e comentários Unicode nos limites.

### T6: entrada e shell do app
Where: web/src
Depends on: T5
Tests: integration browser
Gate: Full
- [ ] Ativação, login/logout, cliente HTTP e shell adaptável, com E2E do fluxo de acesso. APP-06.

### T7: histórico e avaliações na interface
Where: web/src
Depends on: T6
Tests: integration browser
Gate: Full
- [ ] Filtros, paginação, pendentes, diálogo de nota, média e conexões; testes de recarga, duas pessoas, mobile e teclado. APP-07.

### T8: PWA e ambiente de uso
Where: deploy
Depends on: T7
Tests: integration browser e smoke
Gate: Build
- [ ] Manifest/SW/ícones, offline sem dados privados, CI, Compose e guia de primeiro acesso; gerar convites locais e PR separado. APP-08.

## Conferência estrutural

| Tarefa | Dependência/diagrama | Testes na própria entrega |
| --- | --- | --- |
| T1 | none | Unit + PostgreSQL |
| T2 | T1 | Unit + PostgreSQL + CLI |
| T3 | T2 | HTTP/DB |
| T4 | T3 | HTTP/DB |
| T5 | T4 | HTTP/DB |
| T6 | T5 | Chromium |
| T7 | T6 | Chromium |
| T8 | T7 | Chromium/smoke |
