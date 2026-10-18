# My Watched List

Registro pessoal do que duas pessoas assistem, com notas individuais e favoritos do ano. A coleta será própria, começando pela Netflix. O TMDB será usado somente para metadados. Nome provisório.

**Estado atual: Fase 2 implementada.** App com histórico dos dois perfis, filtros, notas individuais, média, pendentes e status das conexões. Login pessoal por convite local, sem senha padrão. Consulte o [guia de primeiro acesso e uso](docs/app.md). O [coletor Netflix](docs/netflix.md) continua operando com sessões cifradas e coleta idempotente. O token TMDB ainda é opcional para usar histórico e notas; sem ele, metadados permanecem pendentes.

## Estrutura

```text
api/
  cmd/api/                 Servidor HTTP autenticado
  cmd/admin/               Convites locais de primeiro acesso
  cmd/migrate/             Executor de migrations incorporadas ao binário
  internal/httpapi/        Rotas e testes HTTP
  migrations/              SQL goose e testes de integração
worker/
  cmd/worker/              CLI administrativa e worker diário
  collector/               Contratos de coleta e extração assistida
  netflix/                 Captura Playwright, perfil e normalização
  store/                   Transações, locks e pendências de metadados
  tmdb/                    Correspondência conservadora e metadados
  syncer/                  Orquestração e agenda diária
internal/auth/             Senhas, convites, sessões e limite de tentativas
internal/config/           Configuração compartilhada dos processos Go
web/                       PWA React + Vite + TypeScript, histórico e notas
deploy/
  Dockerfile.backend       API, worker e executor de migrations
  Dockerfile.web           Frontend estático servido por Caddy
  compose.yaml             Ambiente local
  compose.production.yaml  Portas públicas e domínio HTTPS
  compose.test.yaml        PostgreSQL descartável dos testes
  Caddyfile                Arquivos estáticos e proxy /api/*
.github/workflows/         Build, testes, integração e smoke test do Compose
.specs/                    Escopo, decisões e relatório de validação
.env.example               Variáveis sem credenciais reais
go.mod / go.sum            Um módulo compartilhado por API e worker
```

## Decisões desta fase

- Go 1.26, chi, pgx v5 e goose. Dependências Go em go.sum e dependências web fixadas em package-lock.json.
- PostgreSQL 17. Todas as identidades de domínio usam UUID; IDs do TMDB são externos e separados.
- Um processo de API e um de worker, com sync na inicialização e cron diário às 04:00 em America/Sao_Paulo. Advisory lock PostgreSQL evita sync concorrente na mesma conexão. River fica para quando houver necessidade de fila.
- Coletor Go com playwright-go e Chromium. A imagem worker usa a base oficial Playwright 1.62.1; versão do driver alinhada ao módulo Go. API/migrations continuam em imagem Alpine menor.
- Uma `streaming_account` representa serviço, referência da conta, perfil e pessoa responsável. Cada perfil fica atribuído a uma pessoa. Sincronizações sempre registram essa pessoa explicitamente.
- Uma conexão nova começa em `needs_login`. Há estados `active`, `needs_login`, `error` e `disabled`, com último sucesso, última tentativa e código de erro. O coletor marca sessões expiradas como needs_login.
- Sessões são envelopes AES-256-GCM versão 1, com nonce aleatório e chave de 32 bytes em base64 fora do banco. O UUID da conexão é autenticado no envelope. Login local não grava storage state em arquivo aberto. Não altere a chave sem recifrar as sessões.
- Eventos usam `UNIQUE (streaming_account_id, source_event_key)`. Na Netflix a chave é `netflix:movieID:epoch_ms`, sem índice de página ou título traduzido. O upsert idempotente é exercitado em banco real. Reassistir com uma nova chave é um novo evento. Alterar o dono de uma conexão com eventos é impedido pela chave estrangeira; uma transferência exige fluxo próprio.
- `watch_events.source_title` e `source_title_id` preservam a identidade original. `title_id` e `episode_id` ficam nulos até o enriquecimento. Falhar no TMDB não deverá perder o histórico coletado.
- `watched_on` preserva o dia informado pelo serviço. `watched_at` é opcional e só será preenchido se a fonte informar um instante real. Não inventamos meia-noite. Instantes usam `timestamptz`; a agenda e a apresentação usarão `America/Sao_Paulo` inicialmente.
- Notas de 0 a 10 em passos de 0,5; uma por pessoa e obra do streaming, inclusive para séries e sem depender do TMDB. `source_work_id` preserva o ID da obra; `ratings.source_key` mantém a nota estável após enriquecimento. Notas antigas por `title_id` continuam legíveis. A média usa as notas disponíveis, sem persistência duplicada.
- Episódios pertencem somente a séries e devem coincidir com o título do evento. Temporada 0 permite especiais. Nenhuma duração assistida é inferida por enquanto; duração de catálogo poderá servir a estatísticas estimadas, identificadas como tal.
- Sem senhas padrão. A CLI cria identidades desabilitadas até o primeiro acesso por convite de uma hora. Auth usa Argon2id, sessões opacas revogáveis de 14 dias, cookie HttpOnly/SameSite e Secure em HTTPS. Histórico e notas exigem sessão; mutações também exigem Origin autorizado. As duas pessoas pertencem à mesma casa e compartilham leitura.
- VPS assumida como Linux com Docker Compose v2. Specs, domínio, estratégia de backup e outros streamings serão preenchidos quando forem informados.

## Executar com Docker

Pré-requisitos: Docker com engine Linux em execução, Docker Compose **2.24.4 ou superior** e porta local 8080 livre. Os builds usam Go e Node dentro dos containers.

Na raiz do projeto, copie `.env.example` para `.env`:

```powershell
Copy-Item .env.example .env
```

Em Linux/macOS: `cp .env.example .env`.

Preencha `POSTGRES_PASSWORD` no `.env` com uma senha aleatória de caracteres seguros para URL. Pode gerar com `openssl rand -hex 32`. No PowerShell 7, use `[Convert]::ToHexString([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLowerInvariant()`. Usuário e nome do banco também devem usar caracteres seguros para URL. Preencha `SESSION_ENCRYPTION_KEY` com 32 bytes em base64 (`openssl rand -base64 32`); ela é obrigatória no worker. TMDB é opcional para a coleta, mas necessário para enriquecer os eventos.

```sh
docker compose --env-file .env -f deploy/compose.yaml up --build -d --wait
docker compose --env-file .env -f deploy/compose.yaml ps
```

Abra [http://127.0.0.1:8080](http://127.0.0.1:8080). O Caddy serve a web e encaminha `/api/*` para a API. PostgreSQL e API não publicam portas no host. A API e o worker só iniciam após a migration terminar com sucesso.

```sh
curl http://127.0.0.1:8080/api/healthz
curl http://127.0.0.1:8080/api/readyz
docker compose --env-file .env -f deploy/compose.yaml logs --tail=50 api worker migrate
docker compose --env-file .env -f deploy/compose.yaml down
```

`healthz` responde `{"status":"ok"}` enquanto o processo está vivo. `readyz` responde `{"status":"ready"}` com HTTP 200 quando o banco responde, ou `{"status":"unavailable"}` com HTTP 503. Erros de conexão não são devolvidos ao navegador.

Os dados persistem no volume `postgres_data`. `down -v` remove os volumes e os dados; use somente para descartar ambientes de teste. Alterar `POSTGRES_PASSWORD` no arquivo não muda a senha de um PostgreSQL já inicializado.

## Desenvolvimento sem containers de aplicação

Pré-requisitos: Go 1.26+, Node 24 LTS e um PostgreSQL 17 acessível. Os processos Go não leem `.env` automaticamente: exporte as variáveis no shell. Dentro do Compose, elas são fornecidas pelos serviços.

PowerShell, usando um banco local de desenvolvimento:

```powershell
$env:DATABASE_URL = 'postgres://usuario:senha@127.0.0.1:5432/watched?sslmode=disable'
$env:HTTP_ADDR = '127.0.0.1:8080'
go run ./api/cmd/migrate up
go run ./api/cmd/api
```

Em Bash, use `export DATABASE_URL='...'` e `export HTTP_ADDR='127.0.0.1:8080'` antes dos comandos Go.

Em outro terminal, na pasta `web`:

```sh
npm ci
npm run dev
```

O Vite informa sua URL local e encaminha `/api` para `127.0.0.1:8080`. O worker exige DATABASE_URL e SESSION_ENCRYPTION_KEY. Seu comando padrão executa sync inicial e diário; o [guia Netflix](docs/netflix.md) documenta bootstrap, login, status e sync manual.

## Migrations

Os arquivos goose em `api/migrations` são incorporados no executável. A migration inicial cria `users`, `streaming_accounts`, `titles`, `episodes`, `watch_events` e `ratings` com índices e restrições.

```sh
go run ./api/cmd/migrate up
```

Para uma migration nova, adicione um arquivo SQL com numeração crescente e blocos `-- +goose Up` e `-- +goose Down`. Nunca altere uma migration já usada em produção. Em banco descartável, `go run ./api/cmd/migrate down` reverte a última migration; na migration inicial isso remove todas as tabelas do domínio.

No Compose, após mudar migrations, reconstrua a imagem e execute `docker compose --env-file .env -f deploy/compose.yaml run --rm --build migrate` antes de recriar API e worker. O bootstrap de um ambiente vazio já executa migrations automaticamente.

## Verificações

```sh
go test ./...
go vet ./...
go build ./...
npm --prefix web ci
npm --prefix web run build
```

Para testar migrations e idempotência em PostgreSQL descartável:

```sh
docker compose -f deploy/compose.test.yaml up -d --wait
```

PowerShell:

```powershell
$env:TEST_DATABASE_URL = 'postgres://test:test-local-only@127.0.0.1:55432/watched_test?sslmode=disable'
go test -tags=integration -count=1 -v ./api/migrations
```

Bash:

```sh
TEST_DATABASE_URL='postgres://test:test-local-only@127.0.0.1:55432/watched_test?sslmode=disable' go test -tags=integration -count=1 -v ./api/migrations
```

```sh
docker compose -f deploy/compose.test.yaml down -v
```

A tag `integration` exige `TEST_DATABASE_URL`; sua ausência falha explicitamente. Os testes criam e removem schemas aleatórios e cobrem migrations, replay, revisitas, locks, rollback, notas, vínculos e retry de metadados. A normalização usa fixture sintética identificada, sem histórico pessoal. Para rodar todos os testes, instale Chromium/driver com `go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6201.1 install --only-shell chromium` e execute `npm --prefix web run build
go test -tags=integration -count=1 ./...`; em Linux acrescente `--with-deps` à instalação. Os testes de browser usam um site local controlado e não substituem validação Netflix ao vivo.

O GitHub Actions executa testes Go com race detector, integração PostgreSQL, build web e smoke test da stack Docker. Os três jobs passaram na [validação inicial da Fase 0](https://github.com/Ronald-Lacerda/my-watched-list/actions/runs/39166847423). A revisão independente confirmou 16 critérios e detectou 10 falhas injetadas em cópia isolada; consulte `.specs/features/fase-0/validation.md`. Na Fase 1, o Docker Desktop local foi recuperado pela atualização de 4.68.0 para 4.95.0, usando o instalador oficial assinado. Engine 29.8.2, hello-world, build da stack, healthz/readyz e Chromium dentro do worker foram validados localmente. O volume da aplicação e os demais containers existentes foram preservados.

A revisão independente da Fase 1 aprovou os 13 critérios e 75 testes/subtestes, com 11/11 falhas injetadas detectadas. Consulte [.specs/features/fase-1/validation.md](.specs/features/fase-1/validation.md). Os jobs Go, web e containers passaram no [CI da implementação](https://github.com/Ronald-Lacerda/my-watched-list/actions/runs/39170364886).

## VPS e HTTPS

Quando houver domínio com DNS apontando para a VPS Linux e portas 80/443 livres, preencha `APP_DOMAIN` no `.env`, por exemplo `watched.seudominio.com`, sem protocolo nem caminho. O Caddy emitirá e renovará certificados, persistidos nos seus volumes. O frontend estático e o proxy HTTPS ficam no mesmo container.

Comando para a futura implantação, a executar na VPS após configurar o ambiente:

```sh
docker compose --env-file .env -f deploy/compose.yaml -f deploy/compose.production.yaml up --build -d --wait
```

O override substitui a porta local por 80/TCP e 443/TCP/UDP. O banco e a API continuam internos. A configuração atual é uma base de deploy; ainda não foi implantada na VPS. Definir backups de PostgreSQL e manter a chave de criptografia em backup separado antes de armazenar sessões reais.

## Segredos

- Somente variáveis de ambiente no servidor; `.env` é ignorado pelo Git e pelo build Docker.
- Cookies, `storage-state` e sessões também são ignorados. Nunca adicionar fixtures com dados reais.
- TMDB e chave de sessão ficam exclusivamente no backend/worker. Toda variável `VITE_*` entra no bundle público.
- O banco guarda somente o envelope criptografado da sessão. A chave ficará fora do banco; sua rotação precisa recriptografar os envelopes e preservar a versão.
- Não registrar cookies, páginas brutas, credenciais nem a URL de conexão nos logs.

## Próximas fases

| Fase | Entrega | Estado |
| --- | --- | --- |
| 0 | Monorepo, processos, Compose, schema e README | Estrutura aprovada |
| 1 | Netflix: sessão criptografada, API interna de histórico via Playwright, paginação, normalização, cron, idempotência e TMDB | Validada; TMDB real aguarda token |
| 2 | Auth para duas pessoas, timeline com filtros, notas, média do casal, pendentes e PWA | Planejada |
| 3 | Melhores do ano, rankings individuais/do casal e estatísticas | Planejada |
| 4 | Outros streamings e fallback assistido por LLM | Planejada |

O tráfego real de `/viewingactivity` foi inspecionado em sessão autorizada: vaModel inicial e Falcor na paginação, com identidade e total conferidos. Foram salvos 557 eventos de R e 691 de Gabriela, sem aumento ao repetir o sync. Sessão expirada marca `needs_login`; mudança de parser registra erro sem marcar sync bem-sucedido. O token TMDB ainda não foi configurado; eventos ficam pendentes de metadados, e os testes usam HTTP controlado. A interface `AssistedExtractor` fica preparada para injeção futura, com página sanitizada e saída validada no mesmo schema dos eventos. Não há dependência de Trakt, Simkl nem de serviço de terceiros para coleta.

Dados a informar antes do deploy na VPS: SO e recursos da VPS, instalação atual do Docker/proxy, domínio, outros streamings e perfis que cada pessoa usa. A coleta local funciona sem esses dados.

## Referências técnicas

- [Go 1.26](https://go.dev/blog/go1.26)
- [Vite](https://vite.dev/guide/)
- [Goose com migrations incorporadas](https://pressly.github.io/goose/documentation/provider/)
- [Caddy para SPA com API](https://caddyserver.com/docs/caddyfile/patterns)
- [Imagem oficial PostgreSQL](https://hub.docker.com/_/postgres)

Commits pequenos em Conventional Commits, com descrições em português. Um PR por fase ou funcionalidade. A branch inicial é `feat/fase-0`. O usuário solicitou a criação de um repositório novo; adotamos [my-watched-list](https://github.com/Ronald-Lacerda/my-watched-list), privado na conta GitHub autenticada. A estrutura da Fase 0 está no [PR #1](https://github.com/Ronald-Lacerda/my-watched-list/pull/1), com a estrutura aprovada. A Fase 1 usa PR separado baseado em feat/fase-0 enquanto o PR #1 estiver aberto.
