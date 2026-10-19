# Estado do projeto

## Decisions

- AD-001: usar Go + chi + pgx + goose, PostgreSQL e React + Vite + TypeScript.
- AD-002: manter um módulo Go na raiz, com API e worker como processos separados.
- AD-003: notas 0–10 com passos de 0,5; datas sem horário permanecem sem horário.
- AD-004: deduplicar eventos por conexão e chave estável da origem; preservar eventos sem TMDB.
- AD-005: sessões futuras em envelope AES-256-GCM versionado; chave externa ao banco. Nenhum cookie será persistido nesta fase.
- AD-006: concluir somente a Fase 0 nesta entrega e apresentar estrutura antes da Fase 1.

## Handoff

Fase 0 concluída na branch feat/fase-0. Revisão independente PASS: 16/16 critérios, 10/10 mutações detectadas. CI inicial passou nos jobs Go, web e containers; actions atualizadas para v7 e validadas com actionlint, com conferência do CI final antes da entrega do PR. Repositório privado criado por solicitação do usuário: Ronald-Lacerda/my-watched-list; PR #1. Próximo passo: revisão da estrutura pelo usuário, depois Fase 1 Netflix. VPS ainda depende de SO, recursos, domínio e Docker informados. Docker Desktop local falha no socket dockerInference; banco/smoke locais usaram PostgreSQL/Caddy nativos temporários, e containers passaram no CI Linux.
