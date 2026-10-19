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

## Handoff

Redesign Sessão na branch codex/redesign-sessao, baseado em feat/fase-2-app em 29aea94 (PR #3 pronto e CI verde). T1–T3 implementadas: shell, tema/fontes locais/PWA, histórico com destaque e fallback, documentação e atualização local. Build web, Full/Quick/vet/build Go passaram; quatro testes Chromium novos cobrem destaque/filtros/notas/paginação, fallback/teclado e fontes offline. Capturas desktop/celular inspecionadas com dados sintéticos. Aguardar revisão independente após o último commit; publicar PR separado com base feat/fase-2-app e exigir CI verde antes de marcar pronto. Não mesclar PRs #1/#2/#3 nem o novo PR.

App http://127.0.0.1:8080 atualizado com Sessão: somente container web recriado, API pronta e assets/fontes iguais ao build validado. Comparação em memória antes/depois preservou usuários/credenciais, sessões e 1248 eventos; havia zero notas. IDs dos demais containers preservados. Nenhuma migration, coleta ou redefinição de senha. Usuário deve recarregar a página. Logins ronald (R) e gabriela mantidos; primeiro acesso por convite para quem ainda não definiu senha. Não criar/resetar senhas pessoais nem ler cookies. Banco pessoal em 55433 apenas loopback; testes usam exclusivamente TEST_DATABASE_URL em 55432 e schemas sintéticos. Não remover volumes ou trocar SESSION_ENCRYPTION_KEY. No Windows, Go serial com GOMAXPROCS=2, GOFLAGS=-p=1 e GOMEMLIMIT=512MiB; build web antes de integração Chromium. TMDB sem token, sem impedir histórico/notas. Nenhuma implantação VPS remota. Fase 3 ainda depende de pedido do usuário.
