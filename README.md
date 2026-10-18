# My Watched List

Registro pessoal do que duas pessoas assistem, com notas individuais e favoritos do ano. A coleta será própria, começando pela Netflix. O TMDB será usado somente para metadados. Nome provisório.

**Estado atual: Fase 0 implementada.** API de saúde/prontidão, worker de base, schema PostgreSQL, frontend inicial, Compose e CI. Ainda não há coleta, login, notas via app nem PWA instalável.

## Estrutura

```text
api/
  cmd/api/                 Servidor HTTP
  cmd/migrate/             Executor de migrations incorporadas ao binário
  internal/httpapi/        Rotas e testes HTTP
  migrations/              SQL goose e testes de integração
worker/
  cmd/worker/              Processo separado; coleta ainda desabilitada
  collector/               Contratos de coleta e extração assistida
internal/config/           Configuração compartilhada dos processos Go
web/                       React + Vite + TypeScript, base mobile-first
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
- Um processo de API e um de worker. Na Fase 1, o worker terá agenda diária com cron interno e exclusão de execução concorrente em PostgreSQL. River fica para quando houver necessidade comprovada de fila.
- O coletor Netflix será implementado em Go com playwright-go na Fase 1. A imagem do worker receberá os navegadores nessa etapa; a imagem atual não contém Playwright.
- Uma `streaming_account` representa serviço, referência da conta, perfil e pessoa responsável. Cada perfil fica atribuído a uma pessoa. Sincronizações sempre registram essa pessoa explicitamente.
- Uma conexão nova começa em `needs_login`. Há estados `active`, `needs_login`, `error` e `disabled`, com último sucesso, última tentativa e código de erro. Uma sessão expirada será marcada pelo coletor na Fase 1.
- Sessões futuras serão envelopes AES-256-GCM versionados, com nonce aleatório e chave de 32 bytes fora do banco. O schema aceita somente ciphertext e versão da chave em conjunto. A criptografia ainda será implementada; não grave cookies nesta base.
- Eventos usam `UNIQUE (streaming_account_id, source_event_key)`. A chave deve ser um identificador estável da origem; quando não houver ID, a regra de fingerprint será definida a partir do histórico real, sem índice de página. O `INSERT ... ON CONFLICT DO NOTHING` é exercitado em banco real. Reassistir com uma nova chave é um novo evento. Alterar o dono de uma conexão com eventos é impedido pela chave estrangeira; uma transferência exige fluxo próprio.
- `watch_events.source_title` e `source_title_id` preservam a identidade original. `title_id` e `episode_id` ficam nulos até o enriquecimento. Falhar no TMDB não deverá perder o histórico coletado.
- `watched_on` preserva o dia informado pelo serviço. `watched_at` é opcional e só será preenchido se a fonte informar um instante real. Não inventamos meia-noite. Instantes usam `timestamptz`; a agenda e a apresentação usarão `America/Sao_Paulo` inicialmente.
- Notas de 0 a 10 em passos de 0,5; uma por pessoa e título, inclusive para séries. A aplicação atualizará `updated_at` ao editar. Média do casal será calculada, sem persistência duplicada.
- Episódios pertencem somente a séries e devem coincidir com o título do evento. Temporada 0 permite especiais. Nenhuma duração assistida é inferida por enquanto; duração de catálogo poderá servir a estatísticas estimadas, identificadas como tal.
- Sem usuários ou senhas padrão. Cadastro das duas pessoas, hash de senha e sessão HTTP serão implementados na Fase 2, antes de expor dados pessoais. Nenhum endpoint de dados existe nesta fase.
- VPS assumida como Linux com Docker Compose v2. Specs, domínio, estratégia de backup e outros streamings serão preenchidos quando forem informados.

## Executar com Docker

Pré-requisitos: Docker com engine Linux em execução, Docker Compose **2.24.4 ou superior** e porta local 8080 livre. Os builds usam Go e Node dentro dos containers.

Na raiz do projeto, copie `.env.example` para `.env`:

```powershell
Copy-Item .env.example .env
```

Em Linux/macOS: `cp .env.example .env`.

Preencha `POSTGRES_PASSWORD` no `.env` com uma senha aleatória de caracteres seguros para URL. Pode gerar com `openssl rand -hex 32`. No PowerShell 7, use `[Convert]::ToHexString([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLowerInvariant()`. Usuário e nome do banco também devem usar caracteres seguros para URL. TMDB e chave de sessão ficam vazios nesta fase.

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

O Vite informa sua URL local e encaminha `/api` para `127.0.0.1:8080`. Para executar o worker, exporte `DATABASE_URL` nesse terminal e rode `go run ./worker/cmd/worker`. Ele aguarda encerramento e informa que a coleta ainda não foi implementada.

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

A tag `integration` exige `TEST_DATABASE_URL`; sua ausência falha explicitamente. Os testes criam e removem um schema aleatório, exercitam up/down/up, reaplicação sem mudanças, replay de eventos, revisitas, notas e chaves estrangeiras. Use sempre um banco de teste, pois o usuário precisa de permissão para criar/remover schemas. Testes de normalização do payload Netflix serão adicionados junto com o normalizador na Fase 1, usando fixtures sanitizadas.

O GitHub Actions executa testes Go com race detector, integração PostgreSQL, build web e smoke test da stack Docker. O workflow está preparado; sua execução remota depende da publicação no GitHub.

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
| 0 | Monorepo, processos, Compose, schema e README | Implementada; revisar estrutura antes de avançar |
| 1 | Netflix: sessão criptografada, API interna de histórico via Playwright, paginação, normalização, cron, idempotência e TMDB | Próxima |
| 2 | Auth para duas pessoas, timeline com filtros, notas, média do casal, pendentes e PWA | Planejada |
| 3 | Melhores do ano, rankings individuais/do casal e estatísticas | Planejada |
| 4 | Outros streamings e fallback assistido por LLM | Planejada |

Na Fase 1, inspecionar o tráfego real de `/viewingactivity` em sessão autorizada antes de fixar endpoints ou formatos internos. Sessão expirada marcará `needs_login`; mudança de parser registrará erro sem marcar sync bem-sucedido. A interface `AssistedExtractor` fica preparada para injeção futura, com página sanitizada e saída validada no mesmo schema dos eventos. Não há dependência de Trakt, Simkl nem de serviço de terceiros para coleta.

Dados a informar antes do deploy/coletor: SO e recursos da VPS, instalação atual do Docker/proxy, domínio, outros streamings e perfis que cada pessoa usa. Nenhum deles impede a revisão da Fase 0.

## Referências técnicas

- [Go 1.26](https://go.dev/blog/go1.26)
- [Vite](https://vite.dev/guide/)
- [Goose com migrations incorporadas](https://pressly.github.io/goose/documentation/provider/)
- [Caddy para SPA com API](https://caddyserver.com/docs/caddyfile/patterns)
- [Imagem oficial PostgreSQL](https://hub.docker.com/_/postgres)

Commits pequenos em Conventional Commits, com descrições em português. Um PR por fase ou funcionalidade. A branch inicial é `feat/fase-0`. O usuário solicitou a criação de um repositório novo; adotamos `my-watched-list`, privado na conta GitHub autenticada.
