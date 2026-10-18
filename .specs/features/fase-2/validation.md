# Validation da Fase 2: PASS

**Veredito: PASS.** 24/24 critérios verificados e nenhum gap restante. Revisão completa em a2326b2 e revisão do delta T11 em 0bb53ab, sem mudanças de produção. A espera de conclusão do logout eliminou a corrida do teste, reproduzida com atraso controlado. Sensor de produção consolidado: 15 comportamentos discriminados.

- Data: 2026-10-18.
- Verifier: subagente independente; não escreveu a implementação nem os testes entregues.
- Range revisado: `feat/fase-1-netflix..0bb53ab`, branch `feat/fase-2-app`.
- Fonte de verdade: `.specs/features/fase-2/spec.md`, 24 critérios; design e tasks lidos.
- Isolamento: exports de `git archive a2326b2` e `git archive 0bb53ab` em diretório temporário, com junction somente para `web/node_modules`. Build, testes, probes e mutações executados no export. Nenhum `git stash`, acesso a .env/.local pessoal, cookies pessoais ou banco pessoal.
- Banco exclusivo dos testes: `127.0.0.1:55432/watched_test`, com schemas sintéticos criados/removidos por teste (`internal/testdb/db.go:37`).
- T1–T11 estão marcadas e commitadas; as marcações do autor não foram usadas como evidência de aprovação.

## Cobertura rederivada do spec

24/24 critérios têm os resultados esperados sustentados pela revisão e testes. F1, F2, null e F3/F3b estão fechados. Não há lacuna de precisão do spec ou mutante sobrevivente na revisão final.

| AC | Resultado definido no spec | Evidência de asserção e revisão | Resultado |
| --- | --- | --- | --- |
| 1 | Preservar evento/nota anterior; aceitar nota com identidade própria sem TMDB | `api/internal/library/schema_integration_test.go:40`: score==8.5 e comment=="Original"; :43 exige mesmo evento; :46 insere source_key com score 0 sem title_id. | PASS |
| 2 | Filme usa movieID; episódios/traduções conservam ID da série | `worker/netflix/work_test.go:9` fornece traduções diferentes e IDs 201/202 da série 200; :11 exige SourceWorkID 101/200/200 e episódios distintos. | PASS |
| 3 | Replay preenche identidade de evento enriquecido sem duplicar nem apagar nota | `worker/store/work_integration_test.go:58`: count==1; :61 work=="200" e title_id original; :66 score==9 e comment=="Preservar" após dois replays. | PASS |
| 4 | Senhas Unicode 12–128; Argon2id com salt; inválidas/adulteradas não autenticam | `internal/auth/password_test.go:19`: hashes diferentes, prefixo Argon2id, verificação correta/incorreta; :23 rejeita curta/129/UTF8 inválido; :28 rejeita hashes adulterados; :34 aceita 12 e 128 Unicode. | PASS |
| 5 | Convite 1h, consumo único transacional, nova senha e revogação; inválidos rejeitados | `internal/auth/service_integration_test.go:38`: TTL 3590–3600s; :50 reutilização ErrUnauthorized; :75 revoga sessão anterior; :103 convite substituído rejeitado; :124 exatamente um sucesso/uma rejeição concorrentes; :134 expirado rejeitado. `api/internal/httpapi/auth_integration_test.go:125` exige 401 no reuso. | PASS |
| 6 | Sessão 14 dias; somente hash persistido; atributos de cookie e logout | `internal/auth/service_integration_test.go:42` compara TTL; :47 compara hash persistido ao hash do token, diferente do token; :64 exige revogação. `api/internal/httpapi/auth_integration_test.go:87` exige HttpOnly/Lax/Path/MaxAge; :104 limpa cookie; :177 exige Secure em HTTPS. | PASS |
| 7 | Credencial errada 401 genérico; 11ª tentativa 429/15min; sucesso zera contador | `internal/auth/service_integration_test.go:150` exige ErrUnauthorized nas dez primeiras; :154 ErrThrottled na 11ª; :160 aceita após janela; :164 count==0. `api/internal/httpapi/auth_integration_test.go:159` exige 401 e :163 429/Retry-After 900. Mensagem única revisada em `api/internal/httpapi/app.go:157`. | PASS |
| 8 | Sem sessão/expirada/revogada: 401, sem dados privados | `api/internal/httpapi/auth_integration_test.go:107`, :112 e :131 exigem 401/rejeição sem Gabriela; `api/internal/httpapi/history_integration_test.go:114` cobre history/filters/connections; `api/internal/httpapi/ratings_integration_test.go:103` cobre PUT/DELETE sem sessão. | PASS |
| 9 | Origin ausente/estranho 403; JSON inválido 400; tipo incorreto 415; >64KiB 413 | `api/internal/httpapi/auth_integration_test.go:140` cobre Origin público; :154 códigos do login; `api/internal/httpapi/ratings_integration_test.go:107` cobre Origin das notas. `api/internal/httpapi/mutation_body_integration_test.go:31`–:37 define sintaxe/MIME/tamanho/ator/null/array/extra em logout e DELETE, conhecidos e chunked; :50 compara status exato; :53 mantém sessão válida; :58 mantém count=2/soma=11; :77 aceita {} válido. | PASS, F1 corrigida |
| 10 | Histórico de ambos, filtros combinados/inclusivos, 400 inválidos, vazio válido | `api/internal/httpapi/history_integration_test.go:64` exige 6 eventos; :69 define combinação 2, limites inclusivos 2, serviço 1, pessoa 3 e vazio 0; :72 compara total/items; :105 exige 400 para filtros inválidos. IDs filtrados também verificados em `api/internal/httpapi/journal_browser_integration_test.go:72`. | PASS |
| 11 | Ordem watched_on/id DESC, sem perdas/duplicatas; cursor/limite inválido 400 | `api/internal/httpapi/history_integration_test.go:85` rejeita repetição e ordem incorreta em datas iguais; :91/:95 verificam cursor; :101 exige 6 IDs; :105 inclui cursor ruim e limites 0/101. | PASS |
| 12 | Leituras não expõem sessão/hash/ciphertext/credenciais | `api/internal/httpapi/history_integration_test.go:121` percorre sentinela de cookies/ciphertext e nomes privados nos três endpoints; :122 exige ausência. Consultas/projeções públicas revisadas em `api/internal/library/store.go:68`, :101 e :169; não selecionam credenciais. | PASS |
| 13 | Nota 0–10/0,5; comentário até 2000 Unicode; inválida 400 preserva anterior | `api/internal/library/ratings_test.go:10` aceita 0/0.5/10 e 2000 Unicode; :15/:20 rejeitam limites/NaN/Inf/2001; `api/internal/httpapi/ratings_integration_test.go:93` exige 400 e :128 mantém score 6/comment Preservar. | PASS |
| 14 | Ator vem da sessão, obra do evento; outro user_id 400; nota alheia intacta | `api/internal/httpapi/ratings_integration_test.go:53` preserva nota/comment da parceira; :120/:123 rejeitam ator em body/query. `api/internal/httpapi/mutation_body_integration_test.go:34` inclui user_id alheio; :50 exige 400 inclusive chunked; :53 preserva sessão; :58 exige 2 notas/soma=11 depois de cada rejeição. `api/internal/httpapi/ratings.go:37` trata tamanho desconhecido. | PASS, F2 corrigida |
| 15 | Uma nota por pessoa/obra, upsert atômico; delete somente própria | `api/internal/httpapi/ratings_integration_test.go:60` exige count1 para episódios; :73 exige média10/count1/user da outra pessoa após delete; :149/:154 exigem 200 concorrentes/count1. :210/:213 comprovam rollback integral após falha sintética. | PASS |
| 16 | Notas individuais/média/contador; 0+10=5/2 e nota única=ela/1 | `api/internal/httpapi/ratings_integration_test.go:35`: count1/avg0/user/score/comment; :45 count=2/avg5; :49 avg9.25; :73 count1/avg10 da outra pessoa. | PASS |
| 17 | Pendentes por obra e pessoa; zero remove apenas pendência própria | `api/internal/httpapi/ratings_integration_test.go:31` exige 3 obras; :38 exige 2 após zero; :41 outra pessoa segue3; :63 série avaliada sai uma vez; :80 delete devolve pendente. | PASS |
| 18 | Ativação/login/me/logout; erro PT; sem segredo em storage | `api/internal/httpapi/browser_integration_test.go:132` exige erro PT; :140 identidade R após reload; :151 localStorage/sessionStorage==0 e cookie não acessível; :157 logout; :166 fragmento removido; :180 confirma senhas; :189 ativação entra no histórico. | PASS |
| 19 | Seleção atual, reset de cursor, loading/vazio/erro/retry | `api/internal/httpapi/journal_browser_integration_test.go:71`–:78 verifica IDs selecionados/vazio; :182/:184 50→61; :185 IDs únicos61; :189/:190 filtro reseta para1/ID correto; :197/:202 falha/retry. :40 só conclui estado quando feed-loading ausente. `web/src/Journal.tsx:24` aborta solicitações antigas e reinicia cursor. | PASS |
| 20 | Criar/editar/excluir própria nota/comment, outra pessoa/média, persistir reload | `api/internal/httpapi/journal_browser_integration_test.go:94` reload; :99/:103 zero persistido; :112 comentário da outra pessoa visível; :117 média5; :122 edição→4,25; :126 exclusão→0; :127 nota alheia0 preservada. Complementado por valores exatos de comentário no AC13/14. A saída antes da troca de pessoa é agora confirmada em `api/internal/httpapi/journal_browser_integration_test.go:108`. | PASS |
| 21 | UI mostra serviço/pessoa/perfil/sync/status, orientação needs_login, sem sessão | `api/internal/httpapi/history_integration_test.go:139` exige campos da API. `api/internal/httpapi/journal_browser_integration_test.go:139` exige orientação completa; :140 count==3; :144–146 define Netflix/R/R/Conectada, Netflix/Gabriela/Gabriela/Precisa de login, Prime Video/R/R/Conectada; :148 seleciona heading do serviço/perfil; :150 exige perfil exato; :151 estado; :152 pessoa exata; :155 data 18/10/2026 e hora 09:00; :159 ausência de segredo. Mutantes M12–M15 mortos. | PASS, F3/F3b corrigidas |
| 22 | 390px sem overflow; filtros/dialog acessíveis; Escape e foco de retorno | `api/internal/httpapi/journal_browser_integration_test.go:84` envia Escape; :89 exige foco no botão original; :167 scrollWidth<=innerWidth em 390; :170 mesmo limite no dialog. :67/:69 e helpers :22/:28 encontram controles por rótulo acessível; dialog encontrado por papel. | PASS |
| 23 | Manifest/ícones/SW shell; sem cache API/privado; offline avisa e não grava | `api/internal/httpapi/pwa_browser_integration_test.go:44` exige manifest standalone/start_url e ícones192/512; :61 permite apenas shell/assets; :75 aviso Sem conexão; :79 banco sem nota offline; :86/:88 banner/zero cards após reload offline; :95 recuperação6. | PASS |
| 24 | Stack inicia, dados preservados, testes sintéticos isolados, instruções de acesso | Preservação sintética AC1/3; isolamento em `internal/testdb/db.go:37` e fixtures. Config Compose local/HTTPS exit0 repetidos nesta revisão; smoke independente em stack existente: readyz200, HTML200, manifest200, history sem sessão401. Instruções em `docs/app.md:5` e convite funcional/sem token em saída em `api/cmd/admin/main_integration_test.go:34`–:49. Build/up e backup pessoais não foram repetidos; CI cobre startup em `.github/workflows/checks.yml:71`. | PASS com limite operacional descrito |

## Gates independentes

Baseline completo executado independentemente em a2326b2. A correção posterior só adiciona uma espera observável no E2E; em 0bb53ab o build web e o teste alvo foram repetidos, com o probe abaixo. Os comandos completos anteriores não foram repetidos sem mudança de produção.

Todos os comandos shell passaram por `rtk proxy`. Go executado em série com `GOMAXPROCS=2 GOFLAGS=-p=1 GOMEMLIMIT=512MiB`; apenas `TEST_DATABASE_URL` sintético foi disponibilizado, removendo `DATABASE_URL` herdado. O teste de CLI define seu próprio DATABASE_URL para o schema sintético.

| Comando executado no export | Resultado |
| --- | --- |
| `npm --prefix web run build` | exit0, antes do Full |
| `go test -json -count=1 ./...` | exit0, 38 PASS, 0 FAIL, 0 SKIP |
| `go vet ./...` | exit0 |
| `go test -json -tags=integration -count=1 ./...` | exit0, 130 PASS, 0 FAIL, 0 SKIP |
| `go build ./...` | exit0 |
| `gofmt -l api internal worker` | exit0, saída vazia |
| `docker compose --env-file <arquivo-vazio-scratch> -f deploy/compose.yaml config --quiet` | exit0, env exclusivamente sintético |
| Config anterior com `-f deploy/compose.production.yaml` | exit0, domínio sintético |
| `validate_spec.py fase-2` e `validate_tasks.py fase-2` | exit0, 0 erros/avisos |
| GETs sem cookie: /api/readyz, /, /manifest.webmanifest, /api/history | 200,200,200,401 |

As contagens Go incluem subtestes. Comparação estrutural de testes: 22 funções Test no baseline da Fase1, 45 no range atual (+23). Todos os 14 arquivos de testes adicionados têm zero linhas removidas; nenhum teste anterior foi removido ou enfraquecido. Os probes extras não integram a contagem do baseline.

O verificador não reconstruiu/reiniciou os containers pessoais nem consultou seu conteúdo. A preservação foi comprovada com fixtures sintéticas; backup/contagens pessoais da narrativa T8 não são evidência deste relatório. A abertura automática file:// do convite pessoal foi bloqueada; o usuário foi instruído a abrir pelo Explorador/Chrome e escolher sua senha manualmente. Essa ativação permanece pendente; o fluxo de ativação aprovado aqui usa somente convite sintético.

## Sensor de discriminação

Sensor consolidado: 15 comportamentos distintos, todos mortos. M1–M11 foram mortos em 266a19e; o diff 266a19e..a2326b2 só altera o E2E de conexões e documentação, sem alteração de produção. Essa evidência foi mantida. M12–M15 foram executados nesta revisão no export de a2326b2.

Comando: `go test -json -tags=integration -count=1 -run <teste> <pacote>`. Cada mutante de frontend teve build web anterior. Falhas foram de testes, sem erro de compilação. Mutantes isolados em export temporário, nunca na árvore real.

| ID | Local e alteração comportamental | Teste / resultado |
| --- | --- | --- |
| M1 | `internal/auth/service.go:154`: ignora VerifyPassword incorreto | TestCredentialLifecycle, exit1 em 266a19e, morto |
| M2 | `internal/auth/service.go:176`: remove expiração | TestCredentialLifecycle, exit1 em 266a19e, morto |
| M3 | `api/internal/httpapi/app.go:87`: desliga Origin | TestHTTPOriginAndJSON + TestRatingValidationAndAuthorization, exit1 em 266a19e, mortos |
| M4 | `api/internal/library/ratings.go:64`: delete aceita qualquer pessoa | TestIndividualRatingsAndPending, exit1 em 266a19e, morto |
| M5 | `api/internal/library/store.go:134`: exige score>0 para retirar pendência | TestIndividualRatingsAndPending, exit1 em 266a19e, morto |
| M6 | `api/internal/library/store.go:171`: cursor < vira <= | TestHistoryFiltersAndPagination, exit1 em 266a19e, morto |
| M7 | `web/src/service-worker.js:10`: permite /api e armazena resposta na Cache API | TestBrowserPWAOfflinePrivacy, exit1 em 266a19e, morto |
| M8 | `web/src/Connections.tsx:9`: remove orientação needs_login | TestBrowserJournalAndIndividualRatings, exit1 em 266a19e, morto pela asserção :138 |
| M9 | `api/internal/httpapi/app.go:212`: pula validação opcional no logout | TestOptionalMutationBodies/logout, exit1 em 266a19e; códigos de sintaxe/MIME/tamanho/ator/null/array/extra falham |
| M10 | `api/internal/httpapi/ratings.go:37`: !=0 retorna a >0 e ignora chunked | TestOptionalMutationBodies/rating, exit1 em 266a19e; 7 casos chunked falham |
| M11 | `api/internal/httpapi/app.go:104`: permite null chegar ao decoder de objeto | TestOptionalMutationBodies, exit1 em 266a19e; null falha nos dois endpoints/comprimentos |
| M12 | `web/src/Connections.tsx:9`: "Perfil {item.profile}" vira "Perfil indisponível" | TestBrowserJournalAndIndividualRatings em a2326b2, exit1, morto em 14,77 s: card/perfil esperado não encontrado (:148/:149) |
| M13 | `web/src/Connections.tsx:9`: nome da pessoa vira "Outra pessoa" | Mesmo teste em a2326b2, exit1, morto em 7,31 s: pessoa incorreta (:152) |
| M14 | `web/src/Connections.tsx:9`: heading do serviço vira "Outro serviço" | Mesmo teste em a2326b2, exit1, morto em 15,50 s: card do serviço esperado não encontrado (:147/:148) |
| M15 | `web/src/Connections.tsx:9`: fuso America/Sao_Paulo vira UTC | Mesmo teste em a2326b2, exit1, morto em 6,83 s: sync 12:00 em vez de 09:00 (:155) |

Histórico: até 4d94eda, M8 sobreviveu. Em 266a19e, M8 foi morto e M12 sobreviveu. Em a2326b2, M12 foi morto e pessoa/serviço/fuso também foram discriminados. Nenhum sobrevivente permanece.

## Fechamento das correções

| Lacuna anterior | Evidência de fechamento |
| --- | --- |
| F1, logout ignorava JSON/MIME/tamanho | `api/internal/httpapi/mutation_body_integration_test.go:50` compara códigos exatos; :53 preserva sessão; M9 morto em 266a19e |
| F2, DELETE ignorava corpo chunked | :34/:50 exigem 400 para ator alheio conhecido/chunked; :58 exige count=2/soma=11; M10 morto em 266a19e |
| Null não objeto podia alterar nota/sessão | :35/:50 exigem 400, :53/:58 preservam estado, :77 aceita {} válido; M11 morto em 266a19e |
| F3, orientação não discriminante | `api/internal/httpapi/journal_browser_integration_test.go:138`; M8 morto em 266a19e |
| F3b, campos de conexão não discriminantes | :139–157 exigem todos os três cards e seus valores; M12–M15 mortos em a2326b2 |

Não há tarefa de correção pendente. As lições L-007–L-009 mantêm o histórico das falhas resolvidas. O delta T11 gerou uma lição sobre sincronização de logout em testes de navegador.

## Delta T11: corrida de logout no E2E

Commit revisado: 0bb53abb69e060d5c82ca4d3f72b68127d8d481f. O diff desde a validação só adiciona uma espera em `api/internal/httpapi/journal_browser_integration_test.go:108`, além das tarefas/documentação. Nenhuma asserção foi removida e nenhum timeout/retry foi alterado. O diff de produção desde 266a19e continua vazio.

O teste anterior clicava Sair e chamava loginBrowser, que navega imediatamente. Essa navegação podia cancelar o POST/logout. `web/src/App.tsx:41` só altera o usuário/tela depois de await da API; observar o heading de entrada estabelece a conclusão antes da próxima navegação. A mesma sincronização já era usada em `api/internal/httpapi/browser_integration_test.go:157`.

Falha original relatada no CI pull_request: run 39175731069/job 117817878530, enquanto o job push de 845b472 passou. A causa foi confirmada independentemente no export de 0bb53ab: um handler de Playwright para `**/api/auth/logout` retém a requisição por 500 ms antes de route.Continue(). Essa injeção existe somente no scratch.

Comando real, em série, após build web: `go test -json -tags=integration -count=1 -run '^TestBrowserJournalAndIndividualRatings$' ./api/internal/httpapi`.

| Execução independente | Resultado |
| --- | --- |
| Teste atual original, sem atraso | exit0, 1 PASS |
| Teste atual com logout retido 500 ms | exit0, 1 PASS |
| Mesmo atraso, removendo somente a nova espera no scratch | exit1, espera pelo heading de entrada excedeu 8000 ms; reproduziu a falha antiga |
| Espera restaurada, mesmo atraso | exit0, 1 PASS |

A comparação é discriminante: o atraso é idêntico e somente a espera muda. A falha esperada do controle não é uma falha da versão atual. A verificação remota do commit publicado continua a cargo do orquestrador; este relatório não declara CI remoto já concluído.

## Qualidade, limites e rastreabilidade

Escopo da alteração é compatível com a Fase2; consultas usam parâmetros, senhas/tokens ficam fora dos payloads públicos, a UI segue os componentes locais. Diretrizes verificadas: RTK.md, invariantes do schema, matriz de testes das tarefas e coding-principles do skill. Nenhum SPEC_DEVIATION foi encontrado.

Os testes entregues cobrem rollback, concorrência, inválidos/ausentes, limites, zero, tradução, paginação, offline e foco. A cobertura por camada está aprovada após os reforços de T9/T10. UAT pessoal/visual com o usuário não foi declarado como concluído pelo verificador.

| Requisito | Estado efetivo desta revisão |
| --- | --- |
| APP-01–APP-08 | Verificados, 24/24 ACs, nos limites documentados |

O spec não foi alterado pelo verificador. O estado "verified" dos requisitos agora é sustentado pela evidência independente final.

Isolamento final: a árvore real estava limpa antes e após o probe de T11, antes da escrita deste relatório e da lição. Somente validation.md e os arquivos de lessons produzidos pelo script foram escritos nesta passagem. Nenhum código/teste real foi alterado. Scratch, junction, ZIP e script auxiliar descartados. validate_state.py fase-2 aprovou o relatório PASS.
