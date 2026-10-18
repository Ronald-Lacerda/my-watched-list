# Validation da Fase 2: FAIL

**Veredito: FAIL.** Revalidação de 266a19e: os erros de logout, DELETE chunked e null estão corrigidos e a orientação needs_login tornou-se discriminante. Falta a evidência dos campos renderizados do card de conexão: remover o perfil ainda passa no E2E. Corrigir F3b antes de concluir.

- Data: 2026-10-18.
- Verifier: subagente independente; não escreveu a implementação nem os testes entregues.
- Range revisado: `feat/fase-1-netflix..266a19e`, branch `feat/fase-2-app`.
- Fonte de verdade: `.specs/features/fase-2/spec.md`, 24 critérios; design e tasks lidos.
- Isolamento: export de `git archive 266a19e` em diretório temporário, com junction somente para `web/node_modules`. Build, testes, probes e mutações executados no export. Nenhum `git stash`, acesso a .env/.local pessoal, cookies pessoais ou banco pessoal.
- Banco exclusivo dos testes: `127.0.0.1:55432/watched_test`, com schemas sintéticos criados/removidos por teste (`internal/testdb/db.go:25`).
- T1–T9 estão marcadas e commitadas; as marcações do autor não foram usadas como evidência de aprovação.

## Cobertura rederivada do spec

23/24 critérios têm os resultados esperados sustentados pela revisão e testes. AC9 e AC14 foram corrigidos. AC21 ainda tem uma lacuna de regressão dos campos renderizados, já incluída na F3 original. Não há lacuna de precisão do spec.

| AC | Resultado definido no spec | Evidência de asserção e revisão | Resultado |
| --- | --- | --- | --- |
| 1 | Preservar evento/nota anterior; aceitar nota com identidade própria sem TMDB | `api/internal/library/schema_integration_test.go:40`: score==8.5 e comment=="Original"; :43 exige mesmo evento; :46 insere source_key com score 0 sem title_id. | PASS |
| 2 | Filme usa movieID; episódios/traduções conservam ID da série | `worker/netflix/work_test.go:9` fornece traduções diferentes e IDs 201/202 da série 200; :11 exige SourceWorkID 101/200/200 e episódios distintos. | PASS |
| 3 | Replay preenche identidade de evento enriquecido sem duplicar nem apagar nota | `worker/store/work_integration_test.go:58`: count==1; :61 work=="200" e title_id original; :66 score==9 e comment=="Preservar" após dois replays. | PASS |
| 4 | Senhas Unicode 12–128; Argon2id com salt; inválidas/adulteradas não autenticam | `internal/auth/password_test.go:19`: hashes diferentes, prefixo Argon2id, verificação correta/incorreta; :23 rejeita curta/129/UTF8 inválido; :28 rejeita hashes adulterados; :34 aceita 12 e 128 Unicode. | PASS |
| 5 | Convite 1h, consumo único transacional, nova senha e revogação; inválidos rejeitados | `internal/auth/service_integration_test.go:38`: TTL 3590–3600s; :50 reutilização ErrUnauthorized; :75 revoga sessão anterior; :103 convite substituído rejeitado; :124 exatamente um sucesso/uma rejeição concorrentes; :134 expirado rejeitado. `api/internal/httpapi/auth_integration_test.go:125` exige 401 no reuso. | PASS |
| 6 | Sessão 14 dias; somente hash persistido; atributos de cookie e logout | `internal/auth/service_integration_test.go:42` compara TTL; :47 compara hash persistido ao hash do token, diferente do token; :64 exige revogação. `api/internal/httpapi/auth_integration_test.go:87` exige HttpOnly/Lax/Path/MaxAge; :104 limpa cookie; :177 exige Secure em HTTPS. | PASS |
| 7 | Credencial errada 401 genérico; 11ª tentativa 429/15min; sucesso zera contador | `internal/auth/service_integration_test.go:150` exige ErrUnauthorized nas dez primeiras; :154 ErrThrottled na 11ª; :160 aceita após janela; :164 count==0. `api/internal/httpapi/auth_integration_test.go:159` exige 401 e :163 429/Retry-After 900. Mensagem única revisada em `api/internal/httpapi/app.go:149`. | PASS |
| 8 | Sem sessão/expirada/revogada: 401, sem dados privados | `api/internal/httpapi/auth_integration_test.go:107`, :112 e :131 exigem 401/rejeição sem Gabriela; `api/internal/httpapi/history_integration_test.go:114` cobre history/filters/connections; `api/internal/httpapi/ratings_integration_test.go:103` cobre PUT/DELETE sem sessão. | PASS |
| 9 | Origin ausente/estranho 403; JSON inválido 400; tipo incorreto 415; >64KiB 413 | `api/internal/httpapi/auth_integration_test.go:140` cobre Origin público; :154 códigos do login; `api/internal/httpapi/ratings_integration_test.go:107` cobre Origin das notas. `api/internal/httpapi/mutation_body_integration_test.go:31`–:37 define sintaxe/MIME/tamanho/ator/null/array/extra em logout e DELETE, conhecidos e chunked; :50 compara status exato; :53 mantém sessão válida; :58 mantém count2/soma11; :77 aceita {} válido. | PASS, F1 corrigida |
| 10 | Histórico de ambos, filtros combinados/inclusivos, 400 inválidos, vazio válido | `api/internal/httpapi/history_integration_test.go:64` exige 6 eventos; :69 define combinação 2, limites inclusivos 2, serviço 1, pessoa 3 e vazio 0; :72 compara total/items; :105 exige 400 para filtros inválidos. IDs filtrados também verificados em `api/internal/httpapi/journal_browser_integration_test.go:72`. | PASS |
| 11 | Ordem watched_on/id DESC, sem perdas/duplicatas; cursor/limite inválido 400 | `api/internal/httpapi/history_integration_test.go:85` rejeita repetição e ordem incorreta em datas iguais; :91/:95 verificam cursor; :101 exige 6 IDs; :105 inclui cursor ruim e limites 0/101. | PASS |
| 12 | Leituras não expõem sessão/hash/ciphertext/credenciais | `api/internal/httpapi/history_integration_test.go:121` percorre sentinela de cookies/ciphertext e nomes privados nos três endpoints; :122 exige ausência. Consultas/projeções públicas revisadas em `api/internal/library/store.go:67`, :97 e :169; não selecionam credenciais. | PASS |
| 13 | Nota 0–10/0,5; comentário até 2000 Unicode; inválida 400 preserva anterior | `api/internal/library/ratings_test.go:10` aceita 0/0.5/10 e 2000 Unicode; :15/:20 rejeitam limites/NaN/Inf/2001; `api/internal/httpapi/ratings_integration_test.go:93` exige 400 e :128 mantém score6/comment Preservar. | PASS |
| 14 | Ator vem da sessão, obra do evento; outro user_id 400; nota alheia intacta | `api/internal/httpapi/ratings_integration_test.go:53` preserva nota/comment da parceira; :120/:123 rejeitam ator em body/query. `api/internal/httpapi/mutation_body_integration_test.go:34` inclui user_id alheio; :50 exige400 inclusive chunked; :53 preserva sessão; :58 exige2 notas/soma11 depois de cada rejeição. `api/internal/httpapi/ratings.go:37` trata tamanho desconhecido. | PASS, F2 corrigida |
| 15 | Uma nota por pessoa/obra, upsert atômico; delete somente própria | `api/internal/httpapi/ratings_integration_test.go:60` exige count1 para episódios; :73 exige média10/count1/user da outra pessoa após delete; :149/:154 exigem 200 concorrentes/count1. :210/:213 comprovam rollback integral após falha sintética. | PASS |
| 16 | Notas individuais/média/contador; 0+10=5/2 e nota única=ela/1 | `api/internal/httpapi/ratings_integration_test.go:35`: count1/avg0/user/score/comment; :45 count2/avg5; :49 avg9.25; :73 count1/avg10 da outra pessoa. | PASS |
| 17 | Pendentes por obra e pessoa; zero remove apenas pendência própria | `api/internal/httpapi/ratings_integration_test.go:31` exige3 obras; :38 exige2 após zero; :41 outra pessoa segue3; :63 série avaliada sai uma vez; :80 delete devolve pendente. | PASS |
| 18 | Ativação/login/me/logout; erro PT; sem segredo em storage | `api/internal/httpapi/browser_integration_test.go:132` exige erro PT; :140 identidade R após reload; :151 localStorage/sessionStorage==0 e cookie não acessível; :157 logout; :166 fragmento removido; :180 confirma senhas; :189 ativação entra no histórico. | PASS |
| 19 | Seleção atual, reset de cursor, loading/vazio/erro/retry | `api/internal/httpapi/journal_browser_integration_test.go:71`–:78 verifica IDs selecionados/vazio; :162/:164 50→61; :165 IDs únicos61; :169/:170 filtro reseta para1/ID correto; :177/:182 falha/retry. :40 só conclui estado quando feed-loading ausente. `web/src/Journal.tsx:24` aborta solicitações antigas e reinicia cursor. | PASS |
| 20 | Criar/editar/excluir própria nota/comment, outra pessoa/média, persistir reload | `api/internal/httpapi/journal_browser_integration_test.go:94` reload; :99/:103 zero persistido; :111 comentário da outra pessoa visível; :116 média5; :121 edição→4,25; :125 exclusão→0; :126 nota alheia0 preservada. Complementado por valores exatos de comentário no AC13/14. | PASS |
| 21 | UI mostra serviço/pessoa/perfil/sync/status, orientação needs_login, sem sessão | `api/internal/httpapi/history_integration_test.go:139` exige campos públicos da API; browser `api/internal/httpapi/journal_browser_integration_test.go:137` exige status, :138 orientação completa e :139 ausência de segredo. M8 agora morre. M12 substitui o perfil real por "Perfil indisponível" em `web/src/Connections.tsx:9`; o E2E continua passando. Nenhuma asserção do perfil/serviço/pessoa/sync renderizados foi adicionada. | GAP F3b |
| 22 | 390px sem overflow; filtros/dialog acessíveis; Escape e foco de retorno | `api/internal/httpapi/journal_browser_integration_test.go:84` envia Escape; :89 exige foco no botão original; :147 scrollWidth<=innerWidth em390; :150 mesmo limite no dialog. :67/:69 e helpers :22/:28 encontram controles por rótulo acessível; dialog encontrado por papel. | PASS |
| 23 | Manifest/ícones/SW shell; sem cache API/privado; offline avisa e não grava | `api/internal/httpapi/pwa_browser_integration_test.go:44` exige manifest standalone/start_url e ícones192/512; :61 permite apenas shell/assets; :75 aviso Sem conexão; :79 banco sem nota offline; :86/:88 banner/zero cards após reload offline; :95 recuperação6. | PASS |
| 24 | Stack inicia, dados preservados, testes sintéticos isolados, instruções de acesso | Preservação sintética AC1/3; isolamento em `internal/testdb/db.go:25` e fixtures. Config Compose local/HTTPS exit0 na primeira revisão, arquivos de deploy sem alterações na correção; smoke independente em stack existente na primeira revisão: readyz200, HTML200, manifest200, history sem sessão401. Instruções em `docs/app.md:5` e convite funcional/sem token em saída em `api/cmd/admin/main_integration_test.go:34`–:49. Build/up e backup pessoais não foram repetidos; CI cobre startup em `.github/workflows/checks.yml:71`. | PASS com limite operacional descrito |

## Gates independentes

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

Segunda revisão independente em 266a19e: 12 mutantes comportamentais isolados, 11 mortos e 1 sobrevivente. Build web precedeu cada mutante de frontend; arquivos restaurados antes de cada próxima mutação. As falhas vieram de asserções de testes, sem erro de compilação.

Comando: `go test -json -tags=integration -count=1 -run <teste> <pacote>`.

| ID | Local e alteração | Teste / resultado |
| --- | --- | --- |
| M1 | `internal/auth/service.go:154`: ignora VerifyPassword incorreto | TestCredentialLifecycle, exit1, morto |
| M2 | `internal/auth/service.go:176`: remove expiração | TestCredentialLifecycle, exit1, morto |
| M3 | `api/internal/httpapi/app.go:87`: desliga Origin | TestHTTPOriginAndJSON + TestRatingValidationAndAuthorization, exit1, mortos |
| M4 | `api/internal/library/ratings.go:64`: delete aceita qualquer pessoa | TestIndividualRatingsAndPending, exit1, morto |
| M5 | `api/internal/library/store.go:134`: exige score>0 para retirar pendência | TestIndividualRatingsAndPending, exit1, morto |
| M6 | `api/internal/library/store.go:171`: cursor < vira <= | TestHistoryFiltersAndPagination, exit1, morto |
| M7 | `web/src/service-worker.js:10`: permite /api e armazena resposta na Cache API | TestBrowserPWAOfflinePrivacy, exit1, morto |
| M8 | `web/src/Connections.tsx:9`: remove orientação needs_login | TestBrowserJournalAndIndividualRatings, exit1, morto pela nova asserção :138 |
| M9 | `api/internal/httpapi/app.go:212`: pula validação opcional no logout | TestOptionalMutationBodies/logout, exit1; sintaxe/MIME/tamanho/ator/null/array/extra rejeitados pelos testes |
| M10 | `api/internal/httpapi/ratings.go:37`: !=0 retorna a >0 e ignora chunked | TestOptionalMutationBodies/rating, exit1; 7 casos chunked falham |
| M11 | `api/internal/httpapi/app.go:104`: permite null chegar ao decoder de objeto | TestOptionalMutationBodies, exit1; null falha nos dois endpoints e nos dois comprimentos |
| M12 | `web/src/Connections.tsx:9`: "Perfil {item.profile}" vira "Perfil indisponível" | TestBrowserJournalAndIndividualRatings, exit0, 1 PASS. Sobreviveu; evidencia conjunção de campos da F3 original |

Histórico: no range até 4d94eda, M1–M7 morreram e M8 sobreviveu inclusive ao Full. Em 266a19e, M8 foi morto; o único sobrevivente agora é M12. Foram testados somente um campo representativo e a orientação, sem multiplicar mutações equivalentes.

## Fechamento das correções e tarefa restante

F1 fechada: logout passa a ler corpo opcional antes de revogar. Asserções :50/:53 do novo teste provam códigos exatos e ausência de revogação indevida.

F2 fechada: DELETE com ContentLength=-1 passa pela validação. Asserções :50/:58 provam 400 para ator alheio/null e preservação de ambas as notas. M10 e M11 morrem. Null é explicitamente rejeitado como não objeto; {} legítimo continua aceito (:77).

F3 parcialmente fechada: a orientação completa agora é asserida e sua remoção mata M8. A F3 original também exigia valores visuais no card. Esse trecho permanece pendente.

### F3b — P2: asserir os campos da conexão renderizados (AC21)

O E2E abre Conexões, verifica título/status/orientação/ausência de segredo, mas não confere perfil, serviço, pessoa ou sync no card. Trocar o perfil pelo texto "Perfil indisponível" ainda produz PASS. A implementação original continua exibindo o valor correto; a falha é de poder discriminante da cobertura.

Adicionar asserções no card sintético de Gabriela/needs_login para serviço Netflix, perfil Gabriela, pessoa Gabriela e última sincronização derivada da fixture 2026-10-18T12:00:00Z, além do estado e da orientação já cobertos. Repetir M12 e exigir falha do teste quando o perfil for removido. A data exibida deve respeitar America/Sao_Paulo.

## Qualidade, limites e rastreabilidade

Escopo da alteração é compatível com a Fase2; consultas usam parâmetros, senhas/tokens ficam fora dos payloads públicos, a UI segue os componentes locais. Diretrizes verificadas: RTK.md, invariantes do schema, matriz de testes das tarefas e coding-principles do skill. Nenhum SPEC_DEVIATION foi encontrado.

Os testes entregues cobrem rollback, concorrência, inválidos/ausentes, limites, zero, tradução, paginação, offline e foco. A cobertura por camada não passa integralmente devido à F3b. UAT pessoal/visual com o usuário não foi declarado como concluído pelo verificador.

| Requisito | Estado efetivo desta revisão |
| --- | --- |
| APP-01, APP-02, APP-03, APP-05, APP-06, APP-08 | Verificados nos limites documentados |
| APP-04, APP-07 | Needs fix: conjunção visual de AC21/F3b |

O spec não foi alterado pelo verificador. As marcações anteriores "verified" devem ser reconciliadas pelo autor depois das correções.

Isolamento final da segunda revisão: porcelain real vazio antes do sensor e vazio depois das mutações/probes, antes de escrever este relatório. Somente validation.md e os arquivos de lessons produzidos pelo script foram escritos na árvore real. Scratch, junction e arquivos auxiliares descartados após os resultados. O fechamento determinístico validate_state.py deve rejeitar este FAIL; não é autorização para marcar a fase pronta.
