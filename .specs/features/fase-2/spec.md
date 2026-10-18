# Fase 2: histórico e notas no app

## Problem Statement

Os dois perfis Netflix já têm histórico coletado, mas a interface não permite consultá-lo nem avaliar obras. Esta fase entrega acesso privado para R e Gabriela, histórico com filtros, notas individuais, média disponível do casal, pendentes e status das conexões no app React instalável.

## Out of Scope

Melhores do ano e estatísticas de duração/gênero (Fase 3), novos streamings, cadastro público, múltiplos casais/tenants, recuperação por e-mail, edição do histórico coletado e deploy na VPS. O token TMDB continua opcional. Não integrar autenticação externa.

## Assumptions & Open Questions

| Decisão | Padrão escolhido | Motivo |
| --- | --- | --- |
| Autorização | Usuário autorizou Fase 2 e PR separado; decisões ambíguas seguem o padrão simples documentado | Dispensa nova aprovação de spec/ferramentas, conforme instrução original |
| Acesso | E-mail/senha, sessões opacas em cookie HttpOnly/SameSite=Lax, 14 dias; Secure em HTTPS | Backend próprio e sessão revogável, sem token em localStorage |
| Primeiro acesso | CLI gera link local de uso único, válido por 1 hora, para definir senha | Sem senha padrão, envio de e-mail ou segredo no chat |
| Escopo dos dados | Uma casa, dois usuários provisionados por CLI; ambos veem histórico e notas de ambos | Não há cadastro público nem promessa de isolamento entre tenants |
| Notas | 0–10 em passos de 0,5, comentário opcional até 2000 caracteres; média das notas existentes, com contador | Preserva a decisão AD-003; zero é uma nota válida |
| Identidade de obra | Chave própria serviço + ID da obra Netflix; episódios compartilham ID da série | Notas funcionam sem TMDB e não dependem da tradução do título |
| Histórico antigo | Novo sync preenche source_work_id, inclusive em eventos já enriquecidos; série sem ID fica visível e pede sync antes de avaliar | Não adivinhar identidade de série nem perder eventos antigos |
| Metadados posteriores | ratings usa source_key e title_id opcional; o vínculo TMDB não muda a chave da nota | Não perder notas ao enriquecer depois. Unificação de obras entre serviços é futura |
| Período | Datas inclusivas em watched_on, o dia local preservado da origem | Não converter uma data em meia-noite nem alterar a semântica da Fase 1 |
| Paginação | Cursor watched_on/id decrescente; 50 padrão, limite 100 | Ordenação estável sem OFFSET |
| Pendentes | Uma linha por obra ainda sem nota da pessoa logada, respeitando filtros | Não pedir uma nota para cada episódio/revisita |
| Aparência | Diário editorial claro, verde escuro, cards compactos e navegação adaptada ao celular | Evolui a identidade existente sem fontes/serviços de terceiros |
| PWA | Manifest e shell instalável; nenhum cache de API, notas ou sessão; aviso offline | Não prometer escrita offline nem deixar dados privados no cache |
| PR | feat/fase-2-app baseada em feat/fase-1-netflix enquanto PR #2 estiver aberto | Diff exclusivo da fase; não fazer merge das fases anteriores automaticamente |

**Open questions:** none. As preferências já estabelecidas e a autorização para decidir o simples cobrem as escolhas acima.

## User Stories

**Acceptance Criteria**:
1. WHEN aplicar a migration, o sistema SHALL preservar eventos e notas anteriores e aceitar notas por identidade própria sem exigir um registro TMDB.
2. WHEN normalizar uma série, o coletor SHALL preservar o ID da série como source_work_id; filme usa movieID; replay com tradução diferente SHALL manter a identidade da obra.
3. WHEN sincronizar eventos existentes, o worker SHALL preencher source_work_id sem duplicar eventos ou apagar notas, inclusive para eventos já enriquecidos.
4. WHEN criar senha, o sistema SHALL aceitar 12–128 caracteres Unicode válidos e armazenar Argon2id com salt aleatório; senha inválida ou hash adulterado SHALL não autenticar.
5. WHEN usar um convite válido de primeiro acesso, o sistema SHALL consumi-lo uma única vez em transação, definir senha e revogar sessões anteriores; convite expirado, reutilizado ou inválido SHALL retornar 401 sem criar sessão.
6. WHEN autenticar com credenciais válidas, a API SHALL criar sessão de 14 dias, persistir somente hash do token e emitir cookie HttpOnly, SameSite=Lax e Secure em HTTPS; logout SHALL revogar a sessão e limpar o cookie.
7. IF credenciais forem incorretas, a API SHALL responder 401 genérico; a 11ª tentativa por identificador em uma janela de 15 minutos SHALL retornar 429; login bem-sucedido SHALL zerar o contador.
8. IF uma rota privada receber sessão ausente, expirada ou revogada, a API SHALL retornar 401 sem dados do casal.
9. IF uma mutação autenticada ou uma rota pública de login/ativação receber Origin ausente/fora da lista permitida, a API SHALL retornar 403; JSON inválido SHALL retornar 400, tipo de conteúdo incorreto 415 e corpo acima de 64 KiB 413.
10. WHEN consultar histórico autenticado, a API SHALL retornar eventos de ambos com filtros combináveis de pessoa, serviço e datas inclusivas; filtros inválidos SHALL retornar 400 e filtro válido sem correspondência SHALL retornar lista vazia.
11. WHEN paginar, a API SHALL ordenar por watched_on/id decrescente e retornar próxima página sem repetir/perder eventos em datas iguais; cursor inválido ou limite fora de 1–100 SHALL retornar 400.
12. WHEN responder histórico, filtros ou conexões, a API SHALL devolver somente campos públicos necessários e nunca cookies, ciphertext, hash de senha, hash de sessão, URL de banco ou chave de criptografia.
13. WHEN salvar nota, o sistema SHALL aceitar apenas números 0–10 múltiplos de 0,5 e comentário com até 2000 caracteres; entrada inválida SHALL retornar 400 sem alterar a nota anterior.
14. WHEN salvar, editar ou remover nota, a API SHALL usar exclusivamente a pessoa autenticada e a obra derivada do evento; tentativas de informar outro usuário SHALL retornar 400 e a nota da outra pessoa SHALL permanecer intacta.
15. WHEN avaliar revisitas ou episódios da mesma série, o sistema SHALL manter uma nota por pessoa/obra em upsert atômico; excluir nota SHALL afetar somente a nota da pessoa logada.
16. WHEN exibir avaliações, o sistema SHALL devolver as notas individuais e a média das notas disponíveis com contador; 0 e 10 SHALL produzir média 5 e contador 2, e uma única nota SHALL produzir essa nota como média com contador 1.
17. WHEN consultar pendentes, o sistema SHALL listar uma ocorrência por obra sem nota da pessoa logada; uma nota zero SHALL retirar a obra dos pendentes dessa pessoa, sem retirar dos pendentes da outra.
18. WHEN usar o app, a pessoa SHALL poder ativar sua senha, entrar, consultar sua identidade e sair; falha de autenticação SHALL mostrar erro em português e nenhum segredo SHALL ser persistido no armazenamento web.
19. WHEN trocar filtros ou carregar páginas, o app SHALL exibir o resultado correspondente à seleção atual, reiniciar o cursor e oferecer estados de carregamento, vazio e falha com nova tentativa.
20. WHEN abrir uma obra no histórico, a pessoa SHALL poder salvar, editar e excluir sua nota/comentário e ver a avaliação da outra pessoa e média atualizada, preservadas após recarregar a página.
21. WHEN abrir conexões, o app SHALL mostrar serviço, pessoa, perfil, último sync e estado, com orientação de reconexão em needs_login, sem expor a sessão Netflix.
22. WHEN usar o app em 390 px de largura ou por teclado, a interface SHALL permanecer sem rolagem horizontal e oferecer filtros e diálogo acessíveis, fechamento por Escape e retorno de foco ao botão que abriu o diálogo.
23. WHEN servir a PWA, o app SHALL fornecer manifest com ícones e service worker de shell; cache SHALL excluir /api e dados privados e modo offline SHALL avisar indisponibilidade de consulta/gravação.
24. WHEN subir a stack atualizada, migrations, API, worker e web SHALL iniciar sem perder os dados locais; testes de API/banco e navegador SHALL usar dados sintéticos isolados, e o PR SHALL incluir instruções de primeiro acesso.

## Implicit requirements sweep

| Dimensão | Cobertura |
| --- | --- |
| Validação | AC 4, 9–11, 13; UUID, data, JSON e limites explícitos |
| Falhas/parcial | AC 1, 5, 13, 19; erros sanitizados e transações |
| Idempotência | AC 2–3, 15; identidade da obra e unique por usuário |
| Auth/limites | AC 5–9, 14; sem signup público, Origin e throttle persistido |
| Concorrência | AC 5, 11, 15; convite consumido uma vez e upsert |
| Ciclo/expiração | AC 5–6, 8, 23; convite 1 h, sessão 14 dias e logout |
| Observabilidade | AC 12; status sem segredo, sem logs de senha/token |
| Dependências externas | TMDB opcional; coleta antiga permanece funcional, AC 1–3 |
| Transições | AC 5–8, 18; ativação, login, revogação e expiração |

## Requirement Traceability

| ID | Critérios | Estado |
| --- | --- | --- |
| APP-01 | 1–3 | verified |
| APP-02 | 4–7 | verified |
| APP-03 | 8–9 | pending |
| APP-04 | 10–12, 21 | pending |
| APP-05 | 13–17 | pending |
| APP-06 | 18 | pending |
| APP-07 | 19–22 | pending |
| APP-08 | 23–24 | pending |
