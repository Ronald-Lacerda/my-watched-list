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

## Handoff

Fase 2 implementada e validada na branch feat/fase-2-app. PR #3 separado, baseado em feat/fase-1-netflix (PR #2 ainda aberto); PR #1/#2 não foram mesclados. T1–T8, T9/F1, T10/F3b e T11 concluídas com commits atômicos. Revisão independente PASS dos 24 critérios em a2326b2 e do delta de teste T11 em 0bb53ab: Quick 38 e Full 130 casos, vet/build/web/gofmt/configuração Compose e smoke HTTP aprovados. Mutações de autenticação, expiração, Origin, ownership, zero, cursor, cache, validação de corpos e campos/orientação de conexões foram detectadas após os reforços de cobertura. O CI do PR é acompanhado antes de tirá-lo do rascunho; nenhuma mudança de produção ocorreu após 266a19e.

App local atualizado em http://127.0.0.1:8080, com histórico/filtros, notas próprias 0–10/0,5, média, pendentes, conexões e PWA pública sem cache privado. Backup PostgreSQL validado antes da migration 3; R=557 e Gabriela=691 eventos preservados, todos únicos e com identidade da obra. Nenhum dado fictício foi inserido no banco pessoal. Docker está saudável e os demais containers foram preservados. Não remover volumes nem substituir SESSION_ENCRYPTION_KEY.

Convites de uma hora estão nos arquivos ignorados .local/primeiro-acesso-{r,gabriela}-fase2.html. O usuário ainda precisa escolher a própria senha, abrindo o HTML no Chrome pelo Explorador. O navegador integrado bloqueou a abertura file://; sua aba foi atualizada para a tela de login da Fase 2 e mantida disponível. Não imprimir links/tokens, cookies, .env ou histórico pessoal. Para regenerar convites expirados, seguir docs/app.md; gerar outro arquivo novo revoga o convite anterior.

Integração usa somente TEST_DATABASE_URL descartável em 55432; banco pessoal em 55433 fica restrito a loopback pelo compose.admin.yaml. No Windows, rodar Go em sequência com GOMAXPROCS=2, GOFLAGS=-p=1 e GOMEMLIMIT=512MiB; build web precede testes Chromium. TMDB ainda não tem token e não bloqueia histórico/notas. VPS depende de SO/recursos/domínio; nenhuma implantação remota foi feita. Próximo trabalho de produto é Fase 3, somente quando solicitado.
