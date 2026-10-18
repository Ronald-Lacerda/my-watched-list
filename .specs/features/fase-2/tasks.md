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
T1 -> T2 -> T3 -> T4 -> T5 -> T6 -> T7 -> T8 -> T9 -> T10 -> T11 -> T12
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
- [x] Ativação, login/logout, cliente HTTP e shell adaptável, com E2E do fluxo de acesso. APP-06. Build web, Full/Quick/vet aprovados. TestBrowserAuthentication usa Chromium com API/PostgreSQL reais: login errado em português, login correto, persistência após reload, nome R, logout, convite, confirmação de senha e ativação. Asserta localStorage/sessionStorage vazios, cookie invisível ao JavaScript e remoção do token do endereço. O teste revelou navegação por fragmento sem reload; hashchange agora também ativa o convite. Screenshot de login revisado visualmente; CI prepara o build web antes da integração.

### T7: histórico e avaliações na interface
Where: web/src
Depends on: T6
Tests: integration browser
Gate: Full
- [x] Filtros, paginação, pendentes, diálogo de nota, média e conexões. APP-07. Build web e Full/Quick/vet/build Go passaram. TestBrowserJournalAndIndividualRatings verifica filtros combinados/vazio (linhas 66–80), zero/reload/duas pessoas e média 5/4,25/0 (90–127), pendentes e conexões (129–141), Escape/foco (82–89) e 390 px sem overflow (142–150). TestBrowserPaginationAndRecovery exige 61 IDs únicos, reset para um resultado, erro/retry e preservação do formulário até sucesso (154–207). Screenshots de desktop, celular e diálogo revisados. Os testes revelaram rótulos ambíguos e arredondamento da média para uma casa; ambos corrigidos na UI. Nenhuma asserção anterior da Fase 1 ou T1–T6 foi alterada.

### T8: PWA e ambiente de uso
Where: deploy
Depends on: T7
Tests: integration browser e smoke
Gate: Build
- [x] Manifest/SW/ícones, offline sem dados privados, CI, Compose e guia de primeiro acesso. APP-08. Build web e Full/Quick/vet/build Go passaram. TestBrowserPWAOfflinePrivacy exige manifest standalone, ícones 192/512, cache exclusivo do shell, ausência de notas gravadas offline, shell após reload sem histórico privado e recuperação online. O teste revelou redirecionamento de index.html no servidor de teste e sinal onLine impreciso após reload: fallback público normaliza a resposta e falha real de rede também atualiza o aviso. Compose build/up e configurações local/HTTPS passaram; readyz=200, histórico sem sessão=401 e Chromium no worker renderizou o app. Backup PostgreSQL local de 116976 bytes validado por pg_restore; migration/startup preservaram R=557 e Gabriela=691 eventos únicos, todos com source_work_id preenchido. Convites individuais locais gerados sem senha padrão ou token no chat. A aba antiga foi atualizada para o login da Fase 2; abertura automática do HTML do convite foi bloqueada pela política file:// do navegador integrado, e o usuário recebeu instrução para abrir o arquivo pelo Explorador/Chrome. docs/app.md descreve uso, operação e rollback protegido. PR separado será aberto após este commit e só ficará pronto após revisão independente e CI.

### T9: fechar lacunas da revisão independente (F1)
Where: api/internal/httpapi
Depends on: T8
Tests: integration HTTP postgres e browser
Gate: Full
- [x] Corrigir validação de corpos opcionais em logout/DELETE e reforçar orientação needs_login. AC9/14/21. TestOptionalMutationBodies: :50 exige 400 para sintaxe, ator, null, array e objeto extra; 415 para MIME e 413 para excesso, tanto com ContentLength conhecido quanto chunked/-1. :53 exige sessão ainda válida e :58 exige count=2/soma=11 após rejeições. :77 aceita objeto vazio válido. TestBrowserJournalAndIndividualRatings: :138 exige a orientação de entrar novamente para continuar a coleta. Os testes falharam antes das correções; a extensão também revelou null aceito como corpo de DELETE, agora rejeitado pelo decoder que exige objeto. Full/Quick/vet/build aprovados após correção, sem remover/enfraquecer testes. Revalidação independente PASS registrada no relatório; os mutantes de regressão foram detectados.

### T10: conferir os campos das conexões no navegador (F3b)
Where: api/internal/httpapi
Depends on: T9
Tests: integration browser
Gate: Full
- [x] AC21: o browser agora exige três cartões, serviço e perfil identificáveis, pessoa correta, estado e último sync em São Paulo (18/10/2026, 09:00) nas três conexões sintéticas, além da orientação needs_login existente. TestBrowserJournalAndIndividualRatings confere a conjunção em :139–157; a revisão anterior demonstrou mutante que substituía o perfil passando pelo teste. Full/Quick/vet/build aprovados; nenhum código de produção mudou nesta correção. Revalidação independente PASS: mutantes de perfil, pessoa, serviço e horário/fuso detectados.

### T11: aguardar a conclusão da saída antes de navegar no E2E
Where: api/internal/httpapi
Depends on: T10
Tests: integration browser
Gate: Full
- [x] AC18/20: TestBrowserJournalAndIndividualRatings agora espera o heading de login após clicar Sair e antes de loginBrowser navegar. O CI pull_request de 845b472 falhou na espera do heading após navegação prematura; o job push do mesmo código passou. A nova espera mantém todas as asserções e impede cancelar a requisição de logout em andamento. Fluxo Chromium aprovado em cinco repetições consecutivas; Full/Quick/vet/build aprovados. Nenhum código de produção ou timeout foi alterado. Verifier confirmou com atraso controlado de 500 ms: com a espera passa, sem a espera falha, restaurando passa. CI do novo commit é obrigatório antes de tirar o PR do rascunho.

### T12: autenticar por username
Where: internal/auth
Depends on: T11
Tests: unit, integration PostgreSQL/HTTP/admin e browser
Gate: Build
- [x] APP-09/AC25: migration 4, login/DTO/normalização/throttle por username, atribuição administrativa, convites e tela Usuário. TestUsernameMigrationPreservesCredentials exige token anterior válido, mesmo hash/UUID/evento/nota zero e login de ambas as pessoas; rejeita colisões/sintaxe no banco. TestUsernameCanonicalThrottle exige 11ª tentativa bloqueada com caixa/espaços e um único identificador persistido. TestUsernameLoginContract exige ronald/gabriela corretos, ausência de email no DTO e rejeição do contrato antigo. TestUsernameAdministration exige colisão recusada, estado desabilitado preservado e convite ligado à pessoa correta. E2Es existentes mantêm asserções e usam o contrato solicitado. Build web, Full/Quick/vet/build Go e Compose build/up aprovados. Backup validado antes da migration; senhas reais comparadas sem expor hashes, UUIDs e vínculos de eventos/notas preservados; logins ronald/gabriela atribuídos às contas existentes. Revisão independente e CI do delta pendentes antes de liberar o PR novamente.

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
