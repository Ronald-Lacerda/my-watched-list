# Validação de séries agrupadas

**Veredito: PASS.** As séries passam a ocupar uma entrada por serviço/obra e mantêm a avaliação individual da obra inteira. Filmes continuam por evento.

- Data: 2026-10-18.
- Spec: `.specs/features/series-agrupadas/spec.md`.
- Range revisado: `2b98eda0733aff8cf92a1bfc442c2f3119a6a8d2..0b88e8925ecb322b3aac239f3e055c779cd8bb02`.
- Verificador: agente independente `verificador_series`, diferente do autor. Nenhuma alteração de código/teste no checkout real.
- Escopo: API de histórico, apresentação e notas, regressões afetadas e documentação. Melhorias posteriores de metadados ficam fora deste parecer.

## Tarefas

| Tarefa | Resultado | Evidência |
| --- | --- | --- |
| T1 | PASS | Agrupamento, filtros, identidade, paginação e notas verificados com PostgreSQL sintético. |
| T2 | PASS | Build e Chromium reais verificam apresentação e fluxo completo de notas. |
| T3 | PASS | Documentação revista; operação local relatada pelo autor e helper inspecionado. Limites operacionais abaixo. |

## Critérios de aceite

Os valores foram derivados da fixture e dos critérios, antes de avaliar as asserções. A fixture possui seis eventos: três filmes, dois episódios da mesma série entre pessoas e uma série legada sem chave. O resultado agrupado esperado é cinco itens; a leitura bruta continua com seis eventos.

| AC | Resultado esperado | Evidência e asserções | Resultado |
| --- | --- | --- | --- |
| 1 | Agrupar séries confiáveis entre pessoas por serviço/obra antes de paginar; preservar filmes e leitura bruta. | `api/internal/httpapi/series_integration_test.go:30`: `Total==5`, `len(Items)==5`, cursor nulo; linha34: série representada por `ids[2]`, dois registros e duas pessoas; linha47: ambas consultas brutas retornam seis. `web/src/Journal.tsx:20` aplica `group_series=true` tanto ao histórico quanto aos pendentes. Fluxo pendente agrupado em `api/internal/httpapi/series_browser_integration_test.go:56` e API em `api/internal/httpapi/series_integration_test.go:161`. | PASS |
| 2 | Participantes, quantidade, data e total devem refletir o recorte inclusivo, antes da agregação. | `api/internal/httpapi/series_integration_test.go:56`: casos pessoa, dia único inclusivo e pessoa/serviço/período combinados; linha62 compara total, UUID, dia, quantidade, pessoa e nome exatos. `api/internal/httpapi/series_browser_integration_test.go:66`: representante muda para `ids[3]`; linhas67–71 exigem um registro, Gabriela, dia16 e nota preservada. Filtros em `api/internal/library/store.go:134` precedem identidade e resumo. | PASS |
| 3 | Cinco grupos em páginas de um item, sem repetição; série com102 registros mantém representante mais recente; cursor ordena data e UUID. | `api/internal/httpapi/series_integration_test.go:112` acrescenta100 eventos à série; linha120 exige total5 e um item/página; linha124 verifica ordem decrescente e IDs únicos; linha129 exige `ids[2]` e quantidade102; linhas132–144 verificam cursores intermediários, final e exclusão de `ids[3]`. `api/internal/library/store.go:140` escolhe representante antes do cursor da linha181. | PASS |
| 4 | Uma nota por pessoa/obra; aviso explícito de escopo; zero, média, contador, edição, remoção e pendentes corretos. | `api/internal/httpapi/series_browser_integration_test.go:40`: texto exato “Sua nota vale para a série inteira.”; linhas41–46: nota0 com parceira10 produz média5 e `2/2`; linhas48–54: editar para8,5 produz9,25 e remover restaura10/`1/2`; linhas56–61: avaliar7 remove o agregado dos pendentes. `api/internal/httpapi/series_integration_test.go:150` usa episódios diferentes, compara resumo e remoção independente; `api/internal/httpapi/ratings_integration_test.go:57` exige uma única linha de nota da série após dois PUTs em episódios distintos. | PASS |
| 5 | Exibir título da série, participantes reais, registros no período e última assistida; omitir episódio do agregado; preservar filmes. | `api/internal/httpapi/series_browser_integration_test.go:21` exige heading “Série A”; linha22 exige “2 registros no período”; linhas24–33 verificam os dois nomes exatos; linha36 proíbe “episódio” e exige a data17. `web/src/Journal.tsx:48` limita resumo/participantes à série confiável. Filmes preservam nota, edição, zero, foco e duas assistidas distintas em `api/internal/httpapi/journal_browser_integration_test.go:65`, linha88 e linha99; três filmes continuam no total da API em `api/internal/httpapi/series_integration_test.go:30`. | PASS |
| 6 | Sem TMDB, agrupar pela identidade da fonte; nomes iguais e serviços diferentes não se confundem; legados continuam separados e bloqueados para nota, com fallback. | `api/internal/httpapi/series_integration_test.go:34` exige título correto com pôster nulo; linhas89–105 exigem total8, três chaves distintas e dois legados com um registro/pessoa cada. `api/internal/httpapi/ratings_integration_test.go:116` exige409 para nota da série sem chave. Guard e aviso mantidos em `web/src/Journal.tsx:58`; fallback permanece via `Poster` na linha53, sem alteração desse componente. Probe adicional independente em Chromium, descrito abaixo, confirmou botão desabilitado, aviso e fallback do legado. Regressão de imagem quebrada/ausente em `api/internal/httpapi/session_browser_integration_test.go:173` permaneceu passando. | PASS |
| 7 | Atualizar somente API/web e preservar dados/contas/sessões; login, filtros/URL, retry, offline e navegação continuam funcionando. | `api/internal/httpapi/browser_integration_test.go:113` cobre login, recarga, logout e ativação; `api/internal/httpapi/session_browser_integration_test.go:99` verifica quatro filtros na URL e recarga; `api/internal/httpapi/journal_browser_integration_test.go:191` injeta500 e exige recuperação, linha217 preserva formulário após falha; `api/internal/httpapi/pwa_browser_integration_test.go:60` proíbe cache privado e linha88 exige zero cards offline; `api/internal/httpapi/series_browser_integration_test.go:77` verifica320px sem overflow e linha80 fecha por Escape. Foco visível em `api/internal/httpapi/session_browser_integration_test.go:190`. Não há migration/alteração de coletor no diff. Operação local atribuída ao autor, abaixo. | PASS |

Não foram encontrados critérios vagos que impedissem determinar o resultado esperado. Não há conjunções pendentes de implementação.

## Gates independentes

Comandos executados no checkout real, em sequência:

```text
npm --prefix web run build
python -X utf8 .local/checks.py full quick vet build
python -X utf8 C:/Users/Ronal/.codex/skills/tlc-spec-driven/scripts/validate_spec.py series-agrupadas
python -X utf8 C:/Users/Ronal/.codex/skills/tlc-spec-driven/scripts/validate_tasks.py series-agrupadas
git diff --check 2b98eda..0b88e89
```

Todos passaram com exit0. Integração usou exclusivamente PostgreSQL sintético em55432; Go foi serializado com `GOMAXPROCS=2`, `GOFLAGS=-p=1`, `GOMEMLIMIT=512MiB`. Nenhum acesso do verificador a `.env`, cookies, senhas ou banco pessoal.

- Testes de topo no range:54 antes,59 depois, incremento de cinco testes (quatro API e um browser). Full passou com59 funções de teste; os subtestes também foram executados, sem contagem individual neste relatório.
- Quick:16 funções de teste preservadas; passou. Não houve exclusão de teste ou mecanismo de skip nos arquivos de teste.
- Build TypeScript/Vite, `go vet` e `go build` passaram.
- As mudanças nas regressões de interface6→5 e61→60 correspondem à série agrupada. Asserções de zero, média, ownership, filtro, recuperação, cache privado e foco permanecem. O contrato API bruto e suas asserções6/61 não foram reduzidos.

## Sensor de discriminação

O sensor usou `git archive` do HEAD revisado em diretório temporário isolado. As dependências web foram acessadas por junction, sem alteração de `node_modules`. Cada mutante foi compilado antes de testar, restaurado antes do seguinte e executado contra banco sintético. Falhas de compilação não contam como morte comportamental.

| Mutante | Local | Falha injetada | Teste que matou | Resultado |
| --- | --- | --- | --- | --- |
| 1 | `api/internal/library/store.go:136` | Agrupar também filmes. | `TestSeriesGroupingAndFilters`: total4 em vez de5. | Killed |
| 2 | `api/internal/library/store.go:136` | Incluir pessoa na identidade, separando o casal. | `TestSeriesGroupingAndFilters`: total6 em vez de5. | Killed |
| 3 | `api/internal/library/store.go:136` | Agrupar pelo título textual. | `TestSeriesGroupingIdentity`: total6 em vez de8. | Killed |
| 4 | `api/internal/library/store.go:140` | Escolher evento mais antigo como representante. | `TestSeriesGroupingPagination`: representante/data incorretos na série. | Killed |
| 5 | `api/internal/library/store.go:134` | Ignorar filtro de pessoa no modo agrupado. | `TestSeriesGroupingAndFilters`: total e resumo incorretos para pessoa filtrada. | Killed |
| 6 | `api/internal/library/store.go:144` | Contar pessoas em vez de registros. | `TestSeriesGroupingPagination`: quantidade2 em vez de102. | Killed |
| 7 | `web/src/Journal.tsx:20` | Solicitar eventos brutos no app. | `TestBrowserGroupedSeriesAndRatings`: recorte exibe dois itens em vez de um. | Killed |
| 8 | `web/src/RatingDialog.tsx:31` | Informar que a nota vale para um episódio. | `TestBrowserGroupedSeriesAndRatings`: aviso exato de série inteira ausente. | Killed |

**Resultado:8 injetados,8 compilados,8 mortos,0 sobreviventes.** Depois de restaurar o código no scratch, build web e os cinco testes novos passaram novamente. A remoção da junction precedeu a remoção do scratch. Porcelain real antes e depois do sensor foi exatamente vazio. Nenhum stash/checkout/deploy foi usado.

Logs locais do sensor: `C:/Users/Ronal/AppData/Local/Temp/mwl-series-verifier-r7sh4lvw`. Contêm apenas dados sintéticos e resultados de teste.

## Probe adicional e limites

- Um segundo arquivo temporário do HEAD revisado acrescentou, somente no scratch, verificações Chromium ao legado `ids[5]`: `button.disabled===true`, texto “Aguardando atualização”, título contendo “nova sincronização” e `.poster-letter` visível. O fluxo completo de série passou com essas verificações; scratch removido e porcelain real preservado. O probe complementa a evidência de código/contrato e não adiciona teste ao repositório.
- A atualização local não foi repetida pelo verificador. O autor relatou sucesso de build/up de somente API/web com `--no-deps --wait`, prontidão e assets corretos. O helper `.local/design-research/deploy-series.py` foi inspecionado: compara contas e digest de credenciais, preservação dos IDs de sessões/eventos/títulos/notas e IDs dos demais containers. Resultado relatado:1248 eventos,142 títulos e zero notas preservados. Isso não equivale a uma comparação independente de todas as colunas do banco. As consultas do histórico são read-only (`api/internal/library/store.go:165`) e não há nova migration neste range.
- Não houve UAT humano neste parecer. Chromium exercitou o produto compilado com dados sintéticos. Testar o aplicativo pessoal após recarregar permanece uma etapa de uso pelo usuário, sem impedir a revisão do PR.

## Qualidade e rastreabilidade

- Mudanças limitadas à consulta/apresentação necessárias e documentação. Nenhuma nova dependência, tabela, coletor ou abstração genérica.
- Identidade não depende de texto ou TMDB. Filtros precedem agregação, e agregação precede cursor. Contagem inclui reassistidas, conforme spec.
- Cada teste novo cobreAC1–6; regressões existentes sustentamAC7. Código e testes seguem os padrões da biblioteca, fixtures e orientações de teste em `docs/app.md:113`.
- `SAG-01`, `SAG-02` e `SAG-03`: implementados → verificados neste range.
- Nenhuma falha fundamentada, sobrevivente ou desvio de spec. Nenhuma lição nova a registrar.

**Pronto para revisão.** CI remoto e prontidão do PR devem ser confirmados pelo autor no HEAD publicado.
