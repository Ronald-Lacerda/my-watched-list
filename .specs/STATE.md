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

## Handoff

Séries agrupadas na branch codex/agrupar-series, baseada em codex/redesign-sessao em 2b98eda (PR #4 pronto e CI verde). T1–T3 concluídas: API group_series=true opcional, agrupamento antes do cursor, participantes/quantidade do recorte, uma entrada por série e texto explícito de avaliação da série inteira. Filmes e API bruta preservados. Build web e Full/Quick/vet/build Go passaram; cinco testes novos e expectativas dos browsers adaptadas ao contrato aprovado. Revisão independente PASS até 0b88e89: 7/7 critérios, oito mutantes compilados/mortos e nenhum gap; relatório em .specs/features/series-agrupadas/validation.md. Conferência no app real: 193 itens agrupados, sessão preservada. Publicar PR separado baseado em codex/redesign-sessao e exigir CI verde no HEAD antes de marcar pronto. Novo pedido do usuário: resolver capas faltantes e mostrar sinopse automaticamente; iniciar feature separada após esta entrega, usando aliases TMDB e identificação confirmada. Não mesclar PRs.

App http://127.0.0.1:8080 atualizado com séries agrupadas: somente API/web recriadas, prontidão e assets conferidos. Comparação em memória preservou usuários/credenciais, sessões, 1248 eventos e 142 títulos; havia zero notas. Demais containers preservados. Nenhuma migration, coleta ou redefinição de senha nesta entrega. Usuário deve recarregar. TMDB foi configurado pelo usuário no ambiente local e validado: 844 eventos associados a 142 obras com pôster/sinopse; 186 ambíguos e 218 sem correspondência, zero pendências imediatas após os lotes. Não imprimir/commitar token nem ler cookies. Logins ronald (R) e gabriela mantidos; primeiro acesso por convite para quem ainda não definiu senha. Não criar/resetar senhas pessoais. Banco pessoal em 55433 apenas loopback; testes exclusivamente TEST_DATABASE_URL em 55432 e schemas sintéticos. Não remover volumes nem trocar SESSION_ENCRYPTION_KEY. Windows: Go serial GOMAXPROCS=2 GOFLAGS=-p=1 GOMEMLIMIT=512MiB; build web antes de integração Chromium. Nenhuma implantação VPS remota. Fase 3 depende de pedido.
