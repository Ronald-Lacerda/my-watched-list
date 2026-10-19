# Capas e sinopses: validação independente

**Result**: PASS

Os nove critérios estão atendidos. As duas falhas da primeira revisão foram corrigidas em `dfec221` e suas regressões passaram na revalidação independente. Nenhum gap permanece aberto.

**Data**: 2026-10-18

**Spec**: `.specs/features/capas-sinopses/spec.md`

**Diff range**: `b51387f86d2cd45cdbd0a010a126b7d0513b5c69..dfec221`

**Verificador**: subagente independente, autor diferente do verificador. Primeira iteração de correção/revalidação de três permitidas.

## Tarefas e escopo

T1–T6 estão concluídas. A revisão inicial examinou todo o diff até `10ba684`, encontrou dois gaps P1 em AC1/AC3 e executou seis mutações. Esta revalidação examinou integralmente `10ba684..dfec221`, os novos testes e as asserções anteriores pertinentes. Repetiu Build e testes de cliente, armazenamento, API e CLI/worker em banco sintético. T6 não altera UI, migration ou evidências operacionais anteriores.

## Critérios e resultados

| AC | Resultado esperado na spec | `file:line` e asserções conferidas | Resultado |
| --- | --- | --- | --- |
| 1 | Nome/original/alias normalizado só associa quando único; aproximação, truncamento e múltiplas correspondências permanecem pendentes. | `internal/metadata/aliases_test.go:82`: ID/nome/capa/sinopse corretos; `:86`: ErrAmbiguous; `:90`: ErrNotFound; `:93`: zero chamadas para truncados. `internal/metadata/ambiguity_test.go:87`: canônico versus alias, mídias diferentes, duas correspondências e 21 candidatos com canônico exigem ErrAmbiguous, ID zero e zero detalhes; `:83`: falha do alias exige ErrUnavailable sem associação; `:79`: caso único preserva dados. `internal/metadata/client.go:143` expande aliases com um canônico e `:144` limita candidatos. | PASS |
| 2 | Detalhes oficiais, fallback inglês sem inventar texto, erros/payload/limites seguros sem token; Netflix público por ID e CDN oficial. | `internal/metadata/client_test.go:65`: ID 1, sinopse, capa, ano 2024 e Drama. `internal/metadata/aliases_test.go:123`: candidato com ano/capa/sinopse; `:127`: nome português, Official overview, en-US, capa inglesa e gênero; `:159`: ErrUnavailable sem token. `internal/metadata/netflix_test.go:90`: ID/texto/idioma/URL corretos e ausência de autenticação; `:109`: fallback GB; `:150`: rejeição de página não confirmada; `:165`: imagem externa produz PosterURL vazio; `:207`: erro sanitizado; `:215`: zero redirects. | PASS |
| 3 | Busca autenticada, filme/série e consulta limitada; confirmação exige Origin/JSON/identidade válidos e recusa filme para série conhecida. | `api/internal/httpapi/metadata_integration_test.go:81`: 401 sem sessão; `:86`: 400 para consulta/tipo inválidos; `:92`: candidato completo/no-store; `:96`: 403 sem Origin autorizado; `:101`: 400 para JSON/tipo inválidos; `:114`: UUID inválido; `:120`: 409 para identidade insuficiente. Novo `:33`: série conhecida apenas por Netflix, hint vazio, exige HTTP 400/incompatible_type; `:38`: Netflix/tv/title_id nulo preservados. `internal/catalogstore/store_integration_test.go:122`: ErrType pela transação. `internal/catalogstore/store.go:26` incorpora o.media_type. | PASS |
| 4 | Associação transacional por obra/serviço entre pessoas; origem/datas/pessoas/notas intactas; fonte/idioma e URL Netflix separados do caminho TMDB. | `internal/catalogstore/store_integration_test.go:45`: somente dois eventos ligados, isolando outra obra/serviço; `:105`: snapshot de origem igual; `:108`: notas iguais incluindo 0/10; `:117`: complemento sem ID TMDB inventado. `api/internal/httpapi/metadata_integration_test.go:131`: título/texto/capa/ano/gêneros/fonte/idioma e PosterURL vazio para TMDB; `:136`: duas pessoas; `:142`: fallback usa ID de obra 200; `:147`: fonte Netflix/URL separada. | PASS |
| 5 | Novas coletas e resultados concorrentes respeitam a associação persistida sem mudar identificação/notas. | `internal/catalogstore/store_integration_test.go:65`: resultado atrasado mantém Confirmada; `:69`: override no episódio futuro; `:86`: worker espera lock compartilhado; `:100`: concorrência mantém TMDB 99; `:108`: notas intactas; `:140`: fallback excluído das pendências. Teste repetido após T6. | PASS |
| 6 | Sinopse no card; completar/corrigir; celular/teclado; Escape/foco, consulta após erro e formulário de notas preservados. | `api/internal/httpapi/metadata_browser_integration_test.go:42`: foco inicial; `:49`: Minha busca preservada; `:56`: busca não associa; `:65`: Sinopse oficial no card; `:68`: fonte TMDB; `:72`: imagem correta; `:85`: sem overflow em 390px; `:95`: Escape retorna foco; `:114`: formulário mantém zero; `:117`: parceira mantém 10/10; `:123`: API/imagens ausentes do CacheStorage. UI inalterada em T6, evidência Chromium sintética de T4 preservada. | PASS |
| 7 | CLI sem login/coleta, retry explícito e lote limitado; eventos/sessões preservados. | `worker/syncer/enrich_integration_test.go:32`: backoff com zero chamadas; `:35`: retry processa 500; `:41`: 501 eventos preservados; `:44`: needs_login e last_sync nulo. `worker/cmd/worker/enrich_integration_test.go:20`: fábrica de navegador falha se chamada; `:21`: execução sem chave de sessão. Ambos repetidos nesta rodada. | PASS |
| 8 | Migration aditiva, token apenas API/worker, backup validado, dados anteriores preservados e testes sintéticos. | `api/migrations/00005_metadata.sql:2`: Up adiciona coluna/tabela/índice; `:23`: rollback com overrides recusado. `deploy/compose.yaml:46` e `:68`: token apenas API/worker. `.local/metadata-backup-verified.json:1`: dump de 166.016 bytes, 1.248 eventos, 142 títulos, zero notas. `.specs/features/capas-sinopses/tasks.md:66`: restauração/comparação operacional. `.local/checks.py:3`: banco sintético 55432, Go serial. Verificador não acessou dados pessoais, .env, sessões ou banco pessoal. | PASS, evidência operacional agregada |
| 9 | Naruto obtém TMDB 20982 com imagem/sinopse; pendências confirmáveis completadas sem inventar identificação. | `docs/metadados.md:5`: Naruto identifica 20982; `:46`: 1.248 eventos preservados/completos, 1.246 TMDB + 2 Netflix, 193 itens e 192imagensHTTP 200. `.specs/features/capas-sinopses/tasks.md:66`: Match real Naruto e identificação pública dos truncados. `web/src/Journal.tsx:58`: pendência explícita quando necessário; `:59`: completar/corrigir com chave confiável. Evidência agregada da operação autorizada, sem novo acesso pessoal nesta revisão. | PASS, evidência operacional agregada |

**Resultado**: 9/9 ACs atendidos. Nenhum gap de precisão ou comportamento pendente.

## Correções verificadas

| Gap anterior | Prova anterior em 10ba684 | Correção e prova final |
| --- | --- | --- |
| P1/AC1, canônico escondia alias concorrente | Busca com ID 1 canônico Nome Comum e ID 2 com alias oficial igual retornou ID 1/erro nulo, em vez de ambiguidade. Teste independente compilou e falhou. | `internal/metadata/client.go:143` verifica aliases dos candidatos ainda não correspondentes; `:144` aplica cap de 20 com canônico. TestCanonicalAndAliasUniqueness passa nos seis modos. Regressão retorna no sensor M7. |
| P1/AC3, série Netflix perdia tipo conhecido | Complemento Netflix tv com hint vazio aceitou ConfirmTMDB(movie), erro nulo em vez de ErrType. Teste independente compilou e falhou. | `internal/catalogstore/store.go:26` lê tipo do override. Testes banco/API exigem rejeição e associação anterior intacta; ambos voltam a falhar no sensor M9. |

## Sensor de discriminação

Mutações compiladas em scratch isolado, sem alterar asserções. Profundidade ampliada por envolver autenticação e integridade de dados.

| Mutante | Local | Falha injetada | Evidência de morte |
| --- | --- | --- | --- |
| Inicial M1 | `internal/metadata/client.go:95` | Igualdade normalizada aceita prefixo. | aliases_test.go:90, aproximação associada. |
| Inicial M2 | Fallback de overview no cliente | Texto inglês oficial substituído por texto inventado. | aliases_test.go:127, texto incorreto. |
| Inicial M3 | `internal/metadata/netflix.go:344` | Host externo evil.invalid aceito como CDN. | Validação de CDN no armazenamento. Seleção inicial somente TestNetflix passou; incluir armazenamento matou o mesmo mutante. |
| Inicial M4 | `internal/catalogstore/store.go:114` | Lote perde filtro por chave de obra. | store_integration_test.go:46, cinco eventos em vez de dois. |
| Inicial M5 | `worker/store/enrichment.go:56` | Worker ignora override após lock. | store_integration_test.go:66, escolha substituída por resultado atrasado. |
| Inicial M6 | `api/internal/httpapi/metadata.go:55` | Origin deixa de ser validado. | Asserção HTTP de Origin, resposta indevida 200. |
| T6 M7 | `internal/metadata/client.go:143` | Reverte len(matches)<=1 para ==0. | ambiguity_test.go:88, canônico versus alias/cross_media/cap retornam ID 1 indevido; :84, falha do alias associa. Exit 1. |
| T6 M8 | `internal/metadata/client.go:144` | Cap20 deixa de valer com canônico. | ambiguity_test.go:88, canonical_limit muda ErrAmbiguous para ErrUnavailable após processar alias inválido da fixture. Evidência complementar do limite, não usada isoladamente para provar unicidade. Exit 1. |
| T6 M9 | `internal/catalogstore/store.go:26` | Remove tipo do override do CASE/coalesce. | store_integration_test.go:123, ErrType ausente; metadata_integration_test.go:34, HTTP 200 em vez de 400. Exit 1 nos dois pacotes. |
| T6 M10 | `internal/metadata/client.go:144` | Reduz cap de 20 para um candidato, rejeitando correspondência única entre dois candidatos. | ambiguity_test.go:80, unique_primary deveria retornar dados corretos e retorna ambiguidade. Exit 1. |

**Resultado**: inicial 6/6 mortos; T6 4/4 mortos; total 10/10, zero sobreviventes na cobertura pertinente. Todas as mortes ocorreram por asserção, nenhuma por falha de compilação.

Isolamento: worktrees temporários destacados em 10ba684 e dfec221, ambos removidos. Nesta rodada o porcelain antes/depois foi exatamente `?? .specs/features/capas-sinopses/validation.md`. Nenhum stash. A única escrita do verificador na árvore real foi este relatório.

## Gates e integridade

- Build repetido pelo verificador no scratch dfec221: `rtk proxy python C:/Users/Ronal/Desenvolvimento/my-watched-list/.local/checks.py build`, exit 0.
- Baseline: checks.py exec go test -tags=integration -count=1 com seleção pertinente em internal/metadata, internal/catalogstore, api/internal/httpapi, worker/syncer e worker/cmd/worker. Cinco pacotes passaram, zero falhas/skip na seleção.
- Seleção inclui Match/conservação/aliases/detalhes/payload/Netflix antigos, nova unicidade, armazenamento/concorrência, HTTP normal/nova série Netflix, enriquecimento e CLI sem sessão.
- Full/Quick/vet/build, build web e Compose completos passaram após T6, segundo execução informada pelo orquestrador e `.specs/features/capas-sinopses/tasks.md:68`. Verificador repetiu Build/camadas afetadas/sensor focal; não repetiu a suíte integral.
- Contagem estática confirmada de funções func Test: 59 antes da feature, 76 na primeira revisão, 78 no HEAD final, delta total +19; subcasos não incluídos. Nenhum teste antigo removido/enfraquecido. Cliente TMDB movido com asserções; seletores browser apenas especificam .rating-button após nova ação no card.
- Sem nova UAT humana nesta revisão delegada. Evidência de interface: Chromium sintético T4, sem alteração de UI em T6.
- validate_state.py capas-sinopses: PASS, exit 0 após este relatório.

## Qualidade e rastreabilidade

Diff limitado à feature e às correções comprovadas. Cliente compartilhado por API/worker, identidade/lock compartilhados no armazenamento. Sem mudança de escopo ou redução de asserções. Os testes novos T6 correspondem diretamente a AC1/AC3; cobertura anterior segue mapeada aos demais critérios.

Diretrizes: C:/Users/Ronal/.codex/RTK.md, skill tlc-spec-driven e references/validate.md, sub-agents.md e coding-principles.md. Não havia grafo Graphify; sua criação ficou fora do escopo de somente leitura da árvore real.

| Requisito | Resultado |
| --- | --- |
| CMS-01 | Verificado AC1–2 |
| CMS-02 | Verificado AC4–5/7 |
| CMS-03 | Verificado AC3/6 |
| CMS-04 | Verificado com evidência operacional agregada AC8–9 |

As duas lições ac_gap iniciais foram registradas pelo orquestrador via script oficial em T6. Esta revalidação não produziu novo sinal para lições. Spec/tasks/código/testes não foram alterados pelo verificador.

**Overall**: PASS. Nenhum gap priorizado pendente.
