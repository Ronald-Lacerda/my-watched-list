# Validation da Fase 2: FAIL

**Veredito: FAIL.** O baseline passa, mas há duas violações reproduzidas do contrato HTTP e um mutante sobrevivente na orientação de reconexão. Corrigir F1–F3 e repetir a verificação antes de considerar a fase concluída.

- Data: 2026-10-18.
- Verifier: subagente independente; não escreveu a implementação nem os testes entregues.
- Range revisado: `feat/fase-1-netflix..4d94eda`, branch `feat/fase-2-app`.
- Fonte de verdade: `.specs/features/fase-2/spec.md`, 24 critérios; design e tasks lidos.
- Isolamento: export de `git archive 4d94eda` em diretório temporário, com junction somente para `web/node_modules`. Build, testes, probes e mutações executados no export. Nenhum `git stash`, acesso a .env/.local pessoal, cookies pessoais ou banco pessoal.
- Banco exclusivo dos testes: `127.0.0.1:55432/watched_test`, com schemas sintéticos criados/removidos por teste (`internal/testdb/db.go:25`).
- T1–T8 estão marcadas e commitadas; as marcações do autor não foram usadas como evidência de aprovação.

## Cobertura rederivada do spec

21/24 critérios têm os resultados esperados sustentados pela revisão e testes. AC9 e AC14 falham nos probes independentes; AC21 tem implementação presente, mas evidência de regressão insuficiente. Nenhuma lacuna de precisão do spec foi necessária para explicar esses três resultados.

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
| 9 | Origin ausente/estranho 403; JSON inválido 400; tipo incorreto 415; >64KiB 413 | `api/internal/httpapi/auth_integration_test.go:140` cobre Origin de login/activate/logout; :154 cobre 400/415/413 somente em login. `api/internal/httpapi/ratings_integration_test.go:107` cobre Origin das notas. Probe independente confirma logout 204 nos três corpos que deveriam produzir 400/415/413. `api/internal/httpapi/app.go:198` não decodifica corpo. | FAIL F1 |
| 10 | Histórico de ambos, filtros combinados/inclusivos, 400 inválidos, vazio válido | `api/internal/httpapi/history_integration_test.go:64` exige 6 eventos; :69 define combinação 2, limites inclusivos 2, serviço 1, pessoa 3 e vazio 0; :72 compara total/items; :105 exige 400 para filtros inválidos. IDs filtrados também verificados em `api/internal/httpapi/journal_browser_integration_test.go:72`. | PASS |
| 11 | Ordem watched_on/id DESC, sem perdas/duplicatas; cursor/limite inválido 400 | `api/internal/httpapi/history_integration_test.go:85` rejeita repetição e ordem incorreta em datas iguais; :91/:95 verificam cursor; :101 exige 6 IDs; :105 inclui cursor ruim e limites 0/101. | PASS |
| 12 | Leituras não expõem sessão/hash/ciphertext/credenciais | `api/internal/httpapi/history_integration_test.go:121` percorre sentinela de cookies/ciphertext e nomes privados nos três endpoints; :122 exige ausência. Consultas/projeções públicas revisadas em `api/internal/library/store.go:67`, :97 e :169; não selecionam credenciais. | PASS |
| 13 | Nota 0–10/0,5; comentário até 2000 Unicode; inválida 400 preserva anterior | `api/internal/library/ratings_test.go:10` aceita 0/0.5/10 e 2000 Unicode; :15/:20 rejeitam limites/NaN/Inf/2001; `api/internal/httpapi/ratings_integration_test.go:93` exige 400 e :128 mantém score6/comment Preservar. | PASS |
| 14 | Ator vem da sessão, obra do evento; outro user_id 400; nota alheia intacta | `api/internal/httpapi/ratings_integration_test.go:53` preserva nota/comment da parceira; :120/:123 rejeitam user_id em DELETE com tamanho conhecido/query. Probe chunked com ContentLength=-1 e user_id alheio retorna200, esperado400, pois `api/internal/httpapi/ratings.go:37` ignora corpo. Não houve demonstração de escrita na nota alheia. | FAIL F2 |
| 15 | Uma nota por pessoa/obra, upsert atômico; delete somente própria | `api/internal/httpapi/ratings_integration_test.go:60` exige count1 para episódios; :73 exige média10/count1/user da outra pessoa após delete; :149/:154 exigem 200 concorrentes/count1. :210/:213 comprovam rollback integral após falha sintética. | PASS |
| 16 | Notas individuais/média/contador; 0+10=5/2 e nota única=ela/1 | `api/internal/httpapi/ratings_integration_test.go:35`: count1/avg0/user/score/comment; :45 count2/avg5; :49 avg9.25; :73 count1/avg10 da outra pessoa. | PASS |
| 17 | Pendentes por obra e pessoa; zero remove apenas pendência própria | `api/internal/httpapi/ratings_integration_test.go:31` exige3 obras; :38 exige2 após zero; :41 outra pessoa segue3; :63 série avaliada sai uma vez; :80 delete devolve pendente. | PASS |
| 18 | Ativação/login/me/logout; erro PT; sem segredo em storage | `api/internal/httpapi/browser_integration_test.go:132` exige erro PT; :140 identidade R após reload; :151 localStorage/sessionStorage==0 e cookie não acessível; :157 logout; :166 fragmento removido; :180 confirma senhas; :189 ativação entra no histórico. | PASS |
| 19 | Seleção atual, reset de cursor, loading/vazio/erro/retry | `api/internal/httpapi/journal_browser_integration_test.go:71`–:78 verifica IDs selecionados/vazio; :161/:163 50→61; :164 IDs únicos61; :168/:169 filtro reseta para1/ID correto; :176/:181 falha/retry. :40 só conclui estado quando feed-loading ausente. `web/src/Journal.tsx:24` aborta solicitações antigas e reinicia cursor. | PASS |
| 20 | Criar/editar/excluir própria nota/comment, outra pessoa/média, persistir reload | `api/internal/httpapi/journal_browser_integration_test.go:94` reload; :99/:103 zero persistido; :111 comentário da outra pessoa visível; :116 média5; :121 edição→4,25; :125 exclusão→0; :126 nota alheia0 preservada. Complementado por valores exatos de comentário no AC13/14. | PASS |
| 21 | UI mostra serviço/pessoa/perfil/sync/status, orientação needs_login, sem sessão | `api/internal/httpapi/history_integration_test.go:139` exige valores públicos da conexão; browser `api/internal/httpapi/journal_browser_integration_test.go:137` exige apenas status visível e :138 ausência da sentinela. Remover a orientação em `web/src/Connections.tsx:9` sobrevive ao E2E e ao Full. Falta asserção da orientação e dos campos renderizados. | GAP F3 |
| 22 | 390px sem overflow; filtros/dialog acessíveis; Escape e foco de retorno | `api/internal/httpapi/journal_browser_integration_test.go:84` envia Escape; :89 exige foco no botão original; :146 scrollWidth<=innerWidth em390; :149 mesmo limite no dialog. :67/:69 e helpers :22/:28 encontram controles por rótulo acessível; dialog encontrado por papel. | PASS |
| 23 | Manifest/ícones/SW shell; sem cache API/privado; offline avisa e não grava | `api/internal/httpapi/pwa_browser_integration_test.go:44` exige manifest standalone/start_url e ícones192/512; :61 permite apenas shell/assets; :75 aviso Sem conexão; :79 banco sem nota offline; :86/:88 banner/zero cards após reload offline; :95 recuperação6. | PASS |
| 24 | Stack inicia, dados preservados, testes sintéticos isolados, instruções de acesso | Preservação sintética AC1/3; isolamento em `internal/testdb/db.go:25` e fixtures. Config Compose local/HTTPS exit0; smoke independente em stack existente: readyz200, HTML200, manifest200, history sem sessão401. Instruções em `docs/app.md:5` e convite funcional/sem token em saída em `api/cmd/admin/main_integration_test.go:34`–:49. Build/up e backup pessoais não foram repetidos; CI cobre startup em `.github/workflows/checks.yml:71`. | PASS com limite operacional descrito |

## Gates independentes

Todos os comandos shell passaram por `rtk proxy`. Go executado em série com `GOMAXPROCS=2 GOFLAGS=-p=1 GOMEMLIMIT=512MiB`; apenas `TEST_DATABASE_URL` sintético foi disponibilizado, removendo `DATABASE_URL` herdado. O teste de CLI define seu próprio DATABASE_URL para o schema sintético.

| Comando executado no export | Resultado |
| --- | --- |
| `npm --prefix web run build` | exit0, antes do Full |
| `go test -json -count=1 ./...` | exit0, 38 PASS, 0 FAIL, 0 SKIP |
| `go vet ./...` | exit0 |
| `go test -json -tags=integration -count=1 ./...` | exit0, 97 PASS, 0 FAIL, 0 SKIP |
| `go build ./...` | exit0 |
| `gofmt -l api internal worker` | exit0, saída vazia |
| `docker compose --env-file <arquivo-vazio-scratch> -f deploy/compose.yaml config --quiet` | exit0, env exclusivamente sintético |
| Config anterior com `-f deploy/compose.production.yaml` | exit0, domínio sintético |
| GETs sem cookie: /api/readyz, /, /manifest.webmanifest, /api/history | 200,200,200,401 |

As contagens Go incluem subtestes. Comparação estrutural de testes: 22 funções Test no baseline da Fase1, 44 no range atual (+22). Todos os 13 arquivos de testes adicionados têm zero linhas removidas; nenhum teste anterior foi removido ou enfraquecido. Os probes extras não integram a contagem do baseline.

O verificador não reconstruiu/reiniciou os containers pessoais nem consultou seu conteúdo. A preservação foi comprovada com fixtures sintéticas; backup/contagens pessoais da narrativa T8 não são evidência deste relatório. O convite pessoal ainda depende de abertura manual/ativação pela pessoa; o fluxo de ativação aprovado aqui usa somente convite sintético.

## Sensor de discriminação

Profundidade expandida por envolver autenticação e integridade: oito mutantes comportamentais isolados, sete mortos e um sobrevivente. Cada arquivo foi restaurado antes do próximo mutante. Testes foram compilados/executados; nenhum resultado foi obtido por erro de sintaxe.

Comando para M1–M8: `go test -json -tags=integration -count=1 -run <teste> <pacote>`. M7/M8 também recompilaram web antes do teste. M8 foi confirmado com o Full completo, sem os probes independentes.

| ID | Local e alteração comportamental | Teste / resultado |
| --- | --- | --- |
| M1 | `internal/auth/service.go:154`: ignora falha de VerifyPassword | TestCredentialLifecycle, exit1, morto: senha anterior autenticou |
| M2 | `internal/auth/service.go:176`: remove condição de expiração de sessão | TestCredentialLifecycle, exit1, morto: sessão expirada aceita |
| M3 | `api/internal/httpapi/app.go:86`: desliga validação Origin | TestHTTPOriginAndJSON e TestRatingValidationAndAuthorization, exit1, morto |
| M4 | `api/internal/library/ratings.go:64`: delete aceita qualquer user_id em vez do ator | TestIndividualRatingsAndPending, exit1, morto: exclusão afetou outra pessoa |
| M5 | `api/internal/library/store.go:134`: só score>0 conta como nota nos pendentes | TestIndividualRatingsAndPending, exit1, morto: zero continua pendente |
| M6 | `api/internal/library/store.go:171`: cursor < vira <= | TestHistoryFiltersAndPagination, exit1, morto: repetição/ordem |
| M7 | `web/src/service-worker.js:10`: permite /api e armazena respostas na Cache API | TestBrowserPWAOfflinePrivacy, exit1, morto: arquivo privado inesperado no cache |
| M8 | `web/src/Connections.tsx:9`: remove texto que orienta entrar novamente em needs_login | TestBrowserJournalAndIndividualRatings, exit0; Full adicional 97 PASS/0FAIL/0SKIP. Sobreviveu. |

## Probes independentes e tarefas de correção

### F1 — P2: validar corpo opcional no logout (AC9)

Raiz: `api/internal/httpapi/app.go:198` verifica Origin e revoga a sessão sem verificar corpo.

Probe em arquivo de teste temporário do pacote httpapi, sobre implementação original: POST /api/auth/logout com Origin permitido e cookie sintético.

| Corpo / Content-Type | Esperado | Obtido |
| --- | --- | --- |
| `{` / application/json | 400 | 204 |
| `{}` / text/plain | 415 | 204 |
| 65537 caracteres / application/json | 413 | 204 |

Adicionar validação do corpo quando presente, preservando logout legítimo sem corpo. Cobrir os três erros e provar que solicitação rejeitada não revoga sessão. Verificar também corpo com tamanho desconhecido.

### F2 — P2: ler DELETE independentemente do tamanho conhecido (AC14/AC9)

Raiz: `api/internal/httpapi/ratings.go:37` só decodifica se ContentLength>0. Requisição chunked válida tem ContentLength=-1.

Probe: httptest.NewRequest DELETE /api/events/{evento-sintético}/rating com JSON `{"user_id":"<id-da-outra-pessoa>"}`, ContentLength=-1, TransferEncoding=["chunked"], Origin permitido e sessão sintética. Esperado400, obtido200. Trata-se de violação do contrato de entrada; a revisão não identificou permissão para escrever na nota alheia.

Detectar corpo por leitura limitada/EOF, sem confiar em ContentLength>0. Testar user_id, JSON malformado, tipo incorreto e excesso com chunked. Conferir nota original e nota alheia preservadas após rejeição.

### F3 — P2: tornar orientação e campos da conexão discriminantes (AC21)

A orientação existe na implementação, mas o teste atual aceita sua remoção. Asserir no card de Gabriela/needs_login a orientação concreta, serviço, pessoa, perfil e último sync. Repetir M8: o teste deve falhar quando a orientação for removida.

## Qualidade, limites e rastreabilidade

Escopo da alteração é compatível com a Fase2; consultas usam parâmetros, senhas/tokens ficam fora dos payloads públicos, a UI segue os componentes locais. Diretrizes verificadas: RTK.md, invariantes do schema, matriz de testes das tarefas e coding-principles do skill. Nenhum SPEC_DEVIATION foi encontrado.

Os testes entregues cobrem rollback, concorrência, inválidos/ausentes, limites, zero, tradução, paginação, offline e foco. A cobertura por camada não passa integralmente devido a F1–F3. UAT pessoal/visual com o usuário não foi declarado como concluído pelo verificador.

| Requisito | Estado efetivo desta revisão |
| --- | --- |
| APP-01, APP-02, APP-06, APP-08 | Verificados nos limites documentados |
| APP-03 | Needs fix: AC9 |
| APP-04 | Needs fix: AC21 |
| APP-05 | Needs fix: AC14 |
| APP-07 | Needs fix: evidência da UI de conexões |

O spec não foi alterado pelo verificador. As marcações anteriores "verified" devem ser reconciliadas pelo autor depois das correções.

Isolamento final: porcelain real vazio antes do sensor e vazio depois das mutações/probes, antes de escrever este relatório. Somente validation.md e os arquivos de lessons produzidos pelo script foram escritos na árvore real. Scratch, junction e arquivos auxiliares descartados após os resultados. O fechamento determinístico validate_state.py deve rejeitar este FAIL; não é autorização para marcar a fase pronta.
