# Design da Fase 2

## Architecture Overview

React conversa somente com a API Go na mesma origem. chi expõe auth, histórico e notas; pgx usa PostgreSQL para sessões, convites, throttle, histórico e notas. O frontend não conhece a chave das sessões Netflix. Não há cadastro público: este deploy representa uma única casa com usuários provisionados administrativamente.

Foram consideradas sessão opaca no PostgreSQL (escolhida), JWT no navegador (revogação exige mais estado) e serviço externo (fora da solução própria solicitada). Para avaliar sem TMDB, escolhemos identidade própria da origem em ratings; inventar IDs TMDB seria incorreto e bloquear avaliações até obter token não atende o uso imediato. O usuário autorizou decisões simples documentadas, portanto não há novo bloqueio de aprovação.

## Code Reuse Analysis

Reutilizar migrations goose incorporadas, pgxpool, router de health/readiness, normalizador e store do worker, escala/restrições de ratings, imagens Caddy/Go e Playwright já instalado. Preservar migrations 1 e 2 e testes existentes. Nova migration adiciona source_work_id nos eventos, ratings.source_key opcional e permite title_id nulo com constraint de identidade obrigatória; índices por chave própria e data. Metadados continuam em titles e podem ser vinculados posteriormente sem mudar a nota.

## Components

- `internal/auth`: hash Argon2id, tokens aleatórios, sessões/convites/throttle em PostgreSQL; CLI `api/cmd/admin` gera arquivo local de convite, sem imprimir token/senha.
- `api/internal/httpapi`: NewApp integra rotas e mantém New para os testes health existentes. Auth via cookie, validação explícita de Origin e JSON limitado. Erros de SQL nunca chegam ao cliente.
- `api/internal/library`: consultas com parâmetros, paginação por cursor e agregação das notas. Nenhuma query SELECT de API inclui ciphertext ou password_hash.
- `web/src`: cliente HTTP sem armazenamento de token, tela de entrada/ativação, shell, histórico filtrado, diálogo de nota e conexões. Português, sem fontes/CDNs externos; somente pôster TMDB quando houver.
- `web/public`: manifest, ícones locais e SW limitado a assets/shell, sem cache de API. API sempre Cache-Control no-store.

## Data Models

source_work_id é o movieID de filme ou series da Netflix, independente de título traduzido. Eventos antigos sem ID de série ficam visíveis; a sincronização atualizada preenche antes do uso local. Chave de avaliação `service:work:<source_work_id>`; fallback de filme usa source_title_id. Não criar chave de série a partir do nome. A chave estável é mantida após enriquecimento. Notas antigas ligadas a title_id continuam válidas e são lidas como fallback quando não houver nota pela chave própria.

Sessões e convites usam 32 bytes aleatórios codificados em base64url; banco guarda SHA-256 do token. Sessões expiram em 14 dias, convites em 1 hora e são consumidos sob lock de linha. Senhas usam Argon2id 19 MiB, t=2, p=1, salt aleatório 16 bytes. Hash PHC admite somente parâmetros suportados para impedir custo arbitrário. Throttle de login persistido: 10 tentativas/15 min por identificador normalizado, zerado no sucesso.

## API

| Método/rota | Contrato |
| --- | --- |
| POST /api/auth/login | username/senha, sessão cookie; 401/429 |
| POST /api/auth/activate | token de convite/senha; consumo único e nova sessão |
| GET /api/auth/me | pessoa autenticada ou 401 |
| POST /api/auth/logout | revoga sessão e expira cookie |
| GET /api/filters | pessoas e serviços disponíveis, autenticado |
| GET /api/history | person/service/from/to/pending/cursor/limit; registros, total e próximo cursor |
| GET /api/connections | estados sanitizados |
| PUT /api/events/{id}/rating | nota/comentário da pessoa autenticada para a obra do evento |
| DELETE /api/events/{id}/rating | remove somente a nota da pessoa autenticada |

Origin permitido é configurável em APP_ORIGINS, com localhost e 127.0.0.1:8080 no desenvolvimento; produção limita ao domínio HTTPS. Cookie Secure é derivado de origens HTTPS. Convites levam token no fragmento, removido pelo app; nunca em querystring ou logs. JSON até 64 KiB, campos desconhecidos rejeitados. Sem CORS aberto.

## Risks & Concerns

| Local | Risco | Mitigação |
| --- | --- | --- |
| ratings.title_id obrigatório | Sem token TMDB ninguém consegue avaliar | Chave própria opcional e constraint de identidade; testar com title_id null |
| normalizador não preserva series | Notas diferentes por episódio | source_work_id e sync real antes de entregar a tela |
| upsert atual atualiza apenas não enriquecidos | ID da obra fica ausente em enriquecidos | Atualizar identidade própria também no replay enriquecido e testar |
| API atual sem auth | Exposição de histórico ao adicionar endpoints | Todas as rotas de domínio protegidas, testes 401/Origin/ownership |
| Dois perfis e futuro produto | Globalização inadvertida de histórico entre casais | Documentar deploy de única casa, sem cadastro público; tenants fora desta fase |
| Cache PWA | Dados pessoais persistidos após logout | SW ignora /api, sem token em storage e teste de cache |
| Sessões reais locais | Testes podem poluir dados do casal | Testdb isolado; convites locais sem senha padrão; nunca fixtures pessoais |

## Research

[Argon2 Go](https://pkg.go.dev/golang.org/x/crypto/argon2), [parâmetros OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html), [cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie) e [PWA instalável](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Making_PWAs_installable) foram conferidos nas fontes primárias. Context7 não está disponível nesta sessão. As decisões continuam documentadas na spec; não usar auth gerenciada nem integração Supabase.


A alteração solicitada após a entrega usa users.username canônico (3–40 caracteres ASCII, primeiro letra, demais letras/números/_), índice único e validação equivalente no serviço. Login e hash do throttle compartilham a normalização trim/lower. A migration 4 dá usernames provisórios únicos às contas antigas; admin username atribui os logins amigáveis explicitamente sem trocar IDs/credenciais. O e-mail permanece somente como identificador administrativo legado do coletor, fora dos DTOs públicos de auth. Convites aceitam --username; --email permanece compatível na CLI.
