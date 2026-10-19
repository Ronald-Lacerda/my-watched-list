# Estado do projeto

## Decisions

- AD-001: usar Go + chi + pgx + goose, PostgreSQL e React + Vite + TypeScript.
- AD-002: manter um módulo Go na raiz, com API e worker como processos separados.
- AD-003: notas 0–10 com passos de 0,5; datas sem horário permanecem sem horário.
- AD-004: deduplicar eventos por conexão e chave estável da origem; preservar eventos sem TMDB.
- AD-005: sessões futuras em envelope AES-256-GCM versionado; chave externa ao banco. Nenhum cookie será persistido nesta fase.
- AD-006: concluir somente a Fase 0 nesta entrega e apresentar estrutura antes da Fase 1.
- AD-007: usuário aprovou estrutura e autorizou Fase 1, PR separado e reparo/atualização do Docker. Supera o limite temporal de AD-006. Perfis Netflix: Gabriela e R. Usar um contexto criptografado por perfil, sem senha padrão de usuário web.
- AD-008: usuário autorizou a Fase 2 e PR separado, baseado na Fase 1 ainda aberta. App de uma única casa, sem cadastro público. Sessões opacas revogáveis no PostgreSQL e primeiro acesso por convite local de uso único.
- AD-009: avaliação não exige TMDB. ratings aceita chave própria serviço/ID da obra e title_id opcional; episódios usam o ID da série Netflix, preservado pelo coletor. Enriquecimento posterior não altera a identidade da nota. Escala e média seguem AD-003.

- AD-010: usuário solicitou login por username, com ronald para a conta R e gabriela para Gabriela. Supera a escolha anterior de e-mail no login. Preservar credenciais/UUIDs/dados; coluna própria canônica única e CLI administrativa para atribuição.

- AD-011: usuário aprovou a proposta visual Sessão e autorizou sua implementação com PR separado. Tema escuro cinematográfico, fontes locais e primeiro evento real em destaque; preservar contratos da Fase 2, identidades dos eventos e notas. Sem nova migration ou dependência obrigatória do TMDB.

- AD-012: usuário solicitou uma entrada por série na lista principal. Agrupar por serviço/identidade da obra entre pessoas, com filtros antes do resumo e paginação depois. Supera a apresentação por episódio de AD-011; os eventos originais ficam preservados. Nota já é por série e segue independente por pessoa. Filmes continuam por evento; séries sem chave confiável não são agrupadas por nome.

- AD-013: usuário pediu capas para todos os itens e sinopses visíveis. TMDB continua prioritário; usuário autorizou material público oficial da Netflix como exceção. Usar aliases únicos e identificação confirmada por serviço/obra, durável entre pessoas e episódios. Não inventar sinopses nem buscar imagens no Google.

## Handoff

Capas e sinopses entregues na branch codex/capas-sinopses, PR #6 baseado em codex/agrupar-series (PR #5). T1–T6 concluídas: cliente TMDB compartilhado com aliases, busca/detalhes e fallback de idioma; fallback público Netflix sem sessão; migration 5 metadata_overrides; identificação durável na API/app; CLI enrich sem coleta; sinopses nos cards e correção acessível. Revisão independente PASS: 9/9 critérios e 10/10 mutantes detectados; relatório em .specs/features/capas-sinopses/validation.md. Os dois gaps da primeira rodada foram corrigidos em dfec221, com regressões: unicidade canônico/alias/cap20 e tipo de série conhecido pelo override Netflix. Full/Quick/vet/build, web e Compose passaram; API/worker locais atualizados. Exigir CI verde no HEAD antes de marcar o PR pronto. Não mesclar PRs. PRs #4 e #5 já estavam prontos com CI verde; Fase 3 depende de novo pedido.

App http://127.0.0.1:8080 com 193 itens (séries agrupadas), 1.248 eventos preservados e todos com imagem/sinopse. 1.246 usam TMDB; Cine Gibi e Harry Quer Casar: O Reencontro usam material oficial Netflix. 192 URLs distintas de imagens conferidas HTTP 200. Naruto foi validado por alias como TMDB 20982; nomes truncados tiveram identidade confirmada por página pública completa. Sinopses não foram inventadas. A consulta pública Netflix usa transporte HTTP/1.1 com ALPN correspondente para evitar timeouts HTTP/2 observados; TLS/proxy/cancelamento continuam ativos. Backup local de 166.016 bytes foi restaurado e comparado em banco temporário antes da migration. Usuários, credenciais, sessões, eventos e notas preservados. Há zero notas pessoais na última conferência; não criar avaliações de demonstração no banco pessoal.

Token TMDB somente no .env ignorado, injetado em API/worker. Nunca imprimir token ou ler cookies. Logins ronald (R) e gabriela mantidos; senhas pessoais não foram redefinidas. Banco pessoal apenas loopback 55433; testes somente TEST_DATABASE_URL em 55432 e schemas sintéticos. Não remover volumes nem trocar SESSION_ENCRYPTION_KEY. Go serial no Windows com GOMAXPROCS=2 GOFLAGS=-p=1 GOMEMLIMIT=512MiB; build web antes dos testes Chromium. Nenhuma implantação remota na VPS. Operações e pesquisas pessoais ficam em .local ignorado; não commitar dumps ou payloads pessoais.
