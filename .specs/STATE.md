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

Fase 2 implementada na branch feat/fase-2-app, baseada em feat/fase-1-netflix (PR #2 ainda aberto). T1–T8 concluídas com commits atômicos e Full/Quick/vet/build aprovados; PR #3 está aberto em rascunho, baseado no PR #2; CI inicial passou. Revisão independente encontrou gaps de validação de corpo HTTP e cobertura de reconexão, corrigidos em F1 com novos testes. Revalidação e CI do commit corrigido são o próximo passo. App local atualizado em http://127.0.0.1:8080: histórico/filtros, notas próprias 0–10/0,5, média, pendentes, conexões e PWA pública sem cache de dados privados. PostgreSQL teve backup local validado antes da migration 3; 557 eventos de R e 691 de Gabriela preservados, todos únicos e com identidade de obra. Convites de uma hora estão em arquivos ignorados .local/primeiro-acesso-{r,gabriela}-fase2.html; o usuário recebeu instrução para abrir o convite de R no Chrome após bloqueio de file:// no navegador integrado. A ativação pessoal ainda depende da escolha manual de senha. Não imprimir os links/tokens, cookies, .env ou histórico pessoal. Sessões Netflix e chave local preservadas; nenhum dado fictício foi inserido no banco pessoal. Integração usa somente TEST_DATABASE_URL descartável em 55432. O banco pessoal expõe 55433 somente em loopback pelo compose.admin.yaml. Docker funciona; demais containers preservados. TMDB ainda sem token, sem impedir histórico/notas. VPS/HTTPS remoto ainda depende de SO, recursos e domínio. PR #1/#2 não foram mesclados. No Windows, rodar Go em sequência com GOMAXPROCS=2, GOFLAGS=-p=1 e GOMEMLIMIT=512MiB; build web precede testes Chromium.
