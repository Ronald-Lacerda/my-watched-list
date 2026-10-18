# Fase 1: coleta Netflix

## Problem Statement

Coletar o histórico de cada perfil Netflix com sessão própria, normalizar os eventos, salvar sem duplicações e enriquecer com TMDB. A estrutura da Fase 0 foi aprovada. O PR desta fase será separado, inicialmente baseado em feat/fase-0 enquanto o PR #1 estiver aberto.

## Out of Scope

App de notas, autenticação web, rankings, outros streamings e execução do fallback LLM. Não automatizar senha, CAPTCHA, MFA nem contornar controles da Netflix. Implantação na VPS depende dos dados de infraestrutura ainda não informados.

## Assumptions & Open Questions

| Decisão | Padrão | Motivo |
| --- | --- | --- |
| Perfis | Gabriela e R, informados pelo usuário | Conexões independentes, login validado pelo GUID do perfil |
| Sessão | Login local assistido e estado Playwright criptografado no banco | Sem arquivo de cookies em texto simples |
| API Netflix | vaModel.data no HTML inicial e atom Falcor jsonGraph.aui.viewingActivity.value na paginação | Contrato observado em sessão autorizada; endpoint/build descoberto pelo site |
| Formato | Adaptador estrito para viewedItems; fixture sintética identificada. Título explicitamente em branco vira “Título indisponível na Netflix”, com MatchTitle vazio | Decisão do usuário após ocorrência real; preservar ID/data sem busca TMDB inventada |
| Datas | Epoch em milissegundos preserva instante e dia em America/Sao_Paulo | Datas inválidas falham; sem inventar horário |
| Chave de evento | movieID + instante UTC | Independente de título traduzido e paginação; limita distinção ao que a fonte informa |
| TMDB | Somente correspondência exata única em título/título original e tipo conhecido | Ambiguidade mantém evento pendente sem associação incorreta |
| Agenda | 04:00 America/Sao_Paulo, execução inicial e diária | Recuperar pendências sem fila extra |
| Bootstrap | CLI local; usuários sem acesso web até Fase 2 | Não criar endpoints administrativos públicos |

**Open questions:** none. Validação real exige a sessão autorizada; Chrome está logado, mas não aparece entre os navegadores conectados às ferramentas. A implementação e seus testes não substituirão a evidência de captura real.

## User Stories

**Acceptance Criteria**:
1. WHEN salvar sessão, o sistema SHALL persistir somente envelope AES-256-GCM com nonce aleatório e versão 1; alteração do ciphertext ou contexto de conexão SHALL impedir a leitura.
2. IF a chave não tiver 32 bytes válidos em base64, o worker SHALL falhar antes de iniciar qualquer coleta, sem registrar seu valor.
3. WHEN normalizar JSON válido, o sistema SHALL preservar ID Netflix, título, instante UTC, dia no fuso configurado e produzir chave idêntica em replay mesmo com mudança de ordem ou título traduzido.
4. IF payload, identidade ou data estiverem inválidos, o normalizador SHALL retornar erro de parser sem devolver lote parcial; histórico explicitamente vazio SHALL ser aceito. WHEN title for uma string em branco, o sistema SHALL preservar ID/data e exibir “Título indisponível na Netflix”, sem pista de busca TMDB; title ausente/null SHALL continuar inválido.
5. WHEN coletar, o navegador SHALL usar somente a sessão da conexão, verificar o GUID de profileInfo da atividade e percorrer o histórico até o botão desaparecer ou ficar desabilitado; quando vhSize estiver presente, o total de eventos únicos SHALL coincidir com ele. Limite, repetição de página ou falha SHALL impedir sucesso parcial.
6. IF sessão estiver expirada ou perfil estiver incorreto, o sistema SHALL impedir inserções e marcar needs_login; mudança de parser SHALL marcar error sem atualizar last_sync_at.
7. WHEN sincronizar o mesmo lote duas vezes, inclusive com duas execuções concorrentes, o PostgreSQL SHALL manter um evento por conexão/chave e permitir nova visualização com instante distinto.
8. IF ocorrer falha ao persistir um lote, a transação SHALL não deixar eventos parciais nem marcar último sucesso.
9. WHEN o TMDB devolver correspondência exata única, o sistema SHALL salvar título/metadados e vincular evento; episódio só será vinculado com números explícitos e válidos.
10. IF o TMDB falhar, não encontrar correspondência ou retornar ambiguidade, o histórico SHALL permanecer salvo sem vínculo incorreto e disponível para nova tentativa posterior.
11. WHEN rodar o worker, o sistema SHALL sincronizar conexões ativas na inicialização e no cron diário, com exclusão concorrente em PostgreSQL e encerramento por contexto.
12. WHEN usar a CLI, o operador SHALL poder criar usuário local sem senha padrão, conectar um perfil por login assistido, listar status e executar sync manual sem expor cookies nos logs.
13. WHEN construir a imagem worker, ela SHALL conter Chromium e driver Playwright compatíveis; Compose SHALL iniciar worker após migrations e fornecer segredos somente no backend.

## Requirement Traceability

| ID | Componente | Estado |
| --- | --- | --- |
| NFLX-01 | Sessões, AC 1–2 | verified |
| NFLX-02 | Normalização, AC 3–4 | verified |
| NFLX-03 | Browser, AC 5–6 | verified |
| NFLX-04 | Persistência, AC 7–8 | verified |
| NFLX-05 | Metadados, AC 9–10 | verified |
| NFLX-06 | CLI e agenda, AC 6/11/12 | verified |
| NFLX-07 | Deploy, AC 13 | pending |
