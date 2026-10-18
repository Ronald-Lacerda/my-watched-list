# Estado do projeto

## Decisions

- AD-001: usar Go + chi + pgx + goose, PostgreSQL e React + Vite + TypeScript.
- AD-002: manter um módulo Go na raiz, com API e worker como processos separados.
- AD-003: notas 0–10 com passos de 0,5; datas sem horário permanecem sem horário.
- AD-004: deduplicar eventos por conexão e chave estável da origem; preservar eventos sem TMDB.
- AD-005: sessões futuras em envelope AES-256-GCM versionado; chave externa ao banco. Nenhum cookie será persistido nesta fase.
- AD-006: concluir somente a Fase 0 nesta entrega e apresentar estrutura antes da Fase 1.

## Handoff

Fase 0 implementada na branch feat/fase-0, aguardando revisão independente e execução do CI. Usuário autorizou criar repositório GitHub novo; padrão escolhido: privado, my-watched-list. Não implantar na VPS nem iniciar Fase 1 nesta entrega. Docker Desktop local falha no socket dockerInference; testes de banco e smoke test executados com PostgreSQL/Caddy nativos temporários.
