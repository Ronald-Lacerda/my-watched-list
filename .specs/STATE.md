# Estado do projeto

## Decisions

- AD-001: usar Go + chi + pgx + goose, PostgreSQL e React + Vite + TypeScript.
- AD-002: manter um módulo Go na raiz, com API e worker como processos separados.
- AD-003: notas 0–10 com passos de 0,5; datas sem horário permanecem sem horário.
- AD-004: deduplicar eventos por conexão e chave estável da origem; preservar eventos sem TMDB.
- AD-005: sessões futuras em envelope AES-256-GCM versionado; chave externa ao banco. Nenhum cookie será persistido nesta fase.
- AD-006: concluir somente a Fase 0 nesta entrega e apresentar estrutura antes da Fase 1.
- AD-007: usuário aprovou estrutura e autorizou Fase 1, PR separado e reparo/atualização do Docker. Supera o limite temporal de AD-006. Perfis Netflix: Gabriela e R. Usar um contexto criptografado por perfil, sem senha padrão de usuário web.

## Handoff

Fase 1 validada na branch feat/fase-1-netflix, PR #2 separado e baseado em feat/fase-0 enquanto PR #1 permanece aberto. T1–T7 e correções F1/F2 concluídas; revisão independente PASS em 9fbbd0f, 75 testes/subtestes e 11/11 mutações detectadas após reforçar cobertura CLI, Serve, sessão Chromium e rejeições TMDB. CI Go/web/containers passou em 9fbbd0f (run 39170364886). O status do commit mais recente fica no PR #2. Coleta autorizada real em 18/10/2026: R 557 eventos e Gabriela 691, replay e startup no Docker sem duplicação. GUID canônico vem da atividade; Netflix fornece vaModel inicial, Falcor paginado e botão desabilitado ao terminar. Usuário autorizou preservar títulos em branco como indisponíveis, sem busca TMDB. Sessões cifradas ficam no PostgreSQL local e chave no .env ignorado; não remover volume nem substituir chave. Docker recuperado por atualização oficial 4.95.0, Engine 29.8.2, demais containers preservados. Stack e Chromium no worker validados. Usuário ainda não tem token TMDB; integração HTTP controlada aprovada, chamada real pendente. VPS depende de SO, recursos e domínio. Nenhuma implantação remota foi feita. Próximo trabalho de produto é Fase 2, quando solicitado; não foi iniciado.
