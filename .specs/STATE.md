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

- AD-014: usuário autorizou mesclar todos os PRs e implantar no EasyPanel, levando histórico e contas com as senhas existentes. Supera a restrição anterior de não mesclar/implantar. A branch principal é main. O usuário tornou o repositório público, permitindo clone HTTPS sem chave de deploy. Stack exclusiva em schedule/my-watched-list porque a licença gratuita atingiu três projetos; preservar todos os serviços existentes.

## Handoff

PRs #1–#7 mesclados em main. Primeiro deploy EasyPanel comprovado no commit 3afd88c2bb103a7b1cf22194b21b6dcd03eab539, CI 39189983616 aprovado. API/web/PostgreSQL/migrations/worker na stack schedule/my-watched-list. HTTPS do painel aponta para web:80; sem portas publicadas. Configuração em deploy/compose.easypanel.yaml; Source/Git usa Build Path /deploy, Compose File compose.easypanel.yaml e branch main. Endereço público e variáveis privadas em .local/easypanel.env ignorado; nunca imprimir valores. O usuário tornou o repositório público; não foi criada chave SSH nem mudada sua visibilidade pelo agente.

Backup fresco de 184.154 bytes restaurado localmente em banco temporário e depois no PostgreSQL novo da VPS, com SHA-256/tamanho conferidos e comparação idêntica das 11 tabelas antes de ativar collector. Usuários ronald e gabriela mantêm suas senhas; sessões Netflix e chave originais preservadas. Após ativar collector, os dois perfis sincronizaram sem erro, com 557 e 691 registros recebidos, conexões active e total inalterado de 1.248 eventos. Conteúdo completo de users, ratings e watch_events conferido idêntico após sync. Nenhuma nota de demonstração criada. Cron diário às 4h America/Sao_Paulo. /api/readyz, / e manifest 200 em HTTPS; /api/history sem sessão 401. Tela de login pública aberta para o usuário. Não foi digitada senha pessoal para testar login.

Capas/sinopses, séries agrupadas, redesign Sessão e Fase 2 concluídos; revisões independentes e sensores aprovados. Revisão da configuração EasyPanel: 22 assertivas, 6/6 mutantes detectados; relatório com operação real em .specs/features/deploy-easypanel/validation.md. O registro documental pós-deploy corrige o diretório de trabalho no guia, sem alterar código ou imagens. Exigir CI verde no commit exato de qualquer reimplantação posterior. Fase 3 não autorizada nesta entrega.

Evidências operacionais privadas em .local/production-import-verified.json e capturas producao-*.png; dump e configuração fora do Git. Não remover volumes nem trocar SESSION_ENCRYPTION_KEY. Ambiente local continua preservado; ainda pode executar o cron local em paralelo. Banco pessoal local somente loopback 55433; testes só em 55432 com dados sintéticos. Go serial no Windows: GOMAXPROCS=2, GOFLAGS=-p=1, GOMEMLIMIT=512MiB; build web antes de Chromium. Para novos pushes usar temporariamente credential.helper do gh, pois o Git Credential Manager devolveu um 403 e o gh autenticado funcionou; não alterar credenciais globais.
