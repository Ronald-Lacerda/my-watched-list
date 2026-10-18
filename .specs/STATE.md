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

## Handoff

Fase 2 na branch feat/fase-2-app, PR #3 baseado em feat/fase-1-netflix (PR #2 aberto). T1–T11 estavam validadas e com CI verde em 1cf4591. Usuário solicitou login por username, implementado em T12/AC25: ronald (conta R) e gabriela. API e tela já foram atualizadas no Docker local, com migration 4 e atribuição administrativa dos dois usernames. Backups validados; senhas comparadas antes/depois sem exposição dos hashes, UUIDs e vínculos de histórico/notas preservados. Não criar/resetar senhas pessoais nem ler cookies. A senha de R já foi definida pelo usuário. Quem ainda precisa de primeiro acesso usa convite; admin invite agora aceita --username.

T12 passou em build web, Full/Quick/vet/build Go e Compose build/up. Revisão independente e CI do delta de username são o próximo passo; PR #3 foi convertido temporariamente para rascunho durante atualização. Não mesclar PRs #1/#2/#3. app http://127.0.0.1:8080; usuário deve recarregar para obter o campo Usuário. Banco pessoal em 55433 apenas loopback; testes usam exclusivamente TEST_DATABASE_URL em 55432 e schemas sintéticos. Não remover volumes ou trocar SESSION_ENCRYPTION_KEY. Demais containers preservados. No Windows, Go serial com GOMAXPROCS=2, GOFLAGS=-p=1 e GOMEMLIMIT=512MiB; build web antes de integração Chromium. TMDB sem token, sem impedir histórico/notas. Nenhuma implantação VPS remota. Fase 3 ainda depende de pedido do usuário.
