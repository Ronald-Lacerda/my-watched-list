# Validação do redesign Sessão

**Result**: PASS

Os nove critérios têm evidência compatível. O sensor encontrou uma lacuna no contador de avaliações; T4 corrigiu o teste e a revalidação matou o mesmo mutante. Nenhum gap permanece aberto.

- Data: 2026-10-18.
- Verifier: subagente independente, sem autoria da implementação ou dos testes.
- Spec: `.specs/features/redesign-sessao/spec.md`.
- Diff range: `29aea94121c874e498f9268d3af9b36da5c9dcb4..d56622cd8547d111fcf6d90abcffc8274b3f2571`.
- T1, T2, T3 e T4 estão marcadas como concluídas. CI remoto ainda é gate de entrega do PR.

## Resultados por critério

| AC | Resultado esperado e evidência | Resultado |
| --- | --- | --- |
| 1 | Tema escuro, navegação superior desktop e inferior mobile: `web/src/style.css:50`, `:89`, `:91`; shell e ações em `web/src/App.tsx:17`. `api/internal/httpapi/browser_integration_test.go:113` cobre convite, ativação, login e logout. `journal_browser_integration_test.go:144` exige status/perfil/pessoa/sync corretos nas três conexões. O teste offline exige banner e recuperação em `pwa_browser_integration_test.go:95`. | PASS por regressões e inspeção de código/artefatos visuais. |
| 2 | `session_browser_integration_test.go:15` exige um destaque com UUID exatamente igual ao primeiro por data/ID. `:47` verifica título, pessoa, Netflix e dia esperado; `:52` exige cada um dos seis IDs uma vez. `web/src/Journal.tsx:43` seleciona primeiro evento e `:44` retira somente esse evento dos grupos. | PASS. |
| 3 | `session_browser_integration_test.go:85` exige evento correto após streaming; `:93` combina pessoa, Netflix e intervalo inclusivo; `:99` exige os quatro valores na URL e `:106` repete após reload. `:111` exige primeiro UUID estável e 61 IDs distintos ao acrescentar a segunda página. Regressão de limites e cursor em `history_integration_test.go:61`. | PASS. |
| 4 | `session_browser_integration_test.go:172` exige título longo integral e `:173` exige ausência de img quebrada com arte local. `:202` exige o título do próximo evento e `:203` naturalWidth=1; `:204` exige width/height, eager/high. `web/src/Poster.tsx:4` associa falha ao caminho. A busca desmonta o destaque antes de novos itens em `web/src/Journal.tsx:24`; não existe atualização de metadados em tempo real nesta feature. | PASS no fluxo disponível no produto. |
| 5 | `session_browser_integration_test.go:44` exige contador exato 1/2 antes de avaliar; `:60` percorre os dois eventos da obra exigindo média 5 (`:62`), contador 2/2 (`:65`), exatamente duas linhas R=0/Gabriela=10 (`:68`/`:76`) e ação Sua nota:0 (`:79`). `journal_browser_integration_test.go:100` verifica propagação e persistência do zero; `:123` cobre edição 8,5, média 4,25 e remoção preservando zero. O teste não consulta apenas presença genérica de texto. | PASS após T4 e morte de M5 na rechecagem. |
| 6 | `session_browser_integration_test.go:88` exige zero destaques em seleção vazia. `journal_browser_integration_test.go:138` cobre pendentes e remoção da obra avaliada; `:194` cobre erro/retry; `:217` exige comentário inalterado após falha. `pwa_browser_integration_test.go:85` exige nenhuma gravação offline e `:97` nenhum histórico reaparecido. Expiração 401 em `auth_integration_test.go:131` e resposta em `web/src/api.ts:31`. Não há alteração desse contrato no diff. | PASS. |
| 7 | `session_browser_integration_test.go:174` roda 1280/390/320, exige scrollWidth<=innerWidth e botão de nota >=44x44; `:190` exige Escape fechando e foco visível no mesmo botão. Fonte reduzida para título longo e wrapping em `web/src/app.css:69`/`:186`; alvos de navegação em `web/src/style.css:51`/`:92`; reduced motion em `:97`. Captura sintética de 320 px inspecionada, sem corte horizontal. | PASS. |
| 8 | `session_pwa_integration_test.go:37` compara manifest, meta e color-scheme escuro; `:40` exige os dois caminhos exatos no cache; `:56` rejeita fonte remota; `:74` exige ambas carregadas offline. `pwa_browser_integration_test.go:66` mantém allowlist pública e rejeita dados privados. A única mudança no teste antigo acrescenta os dois WOFF2 explícitos. | PASS. |
| 9 | `docs/app.md:18` limita deploy a build web e up --no-deps web. Inspecionado helper operacional `.local/design-research/deploy-session.py`: compara credenciais em memória, preserva IDs de eventos/notas/sessões e todos os outros containers. Root executou e informou 1248 eventos, zero notas, prontidão e assets/fontes correspondentes. Diff não inclui migration, API, worker ou Compose. | PASS com evidência operacional atribuída ao root; Verifier não acessou banco pessoal ou segredos. |

Não há lacuna de precisão na spec. Os estados e valores usados nas asserções correspondem aos critérios. A lacuna de AC5 encontrada na primeira rodada foi corrigida sem mudar o produto ou enfraquecer testes.

## Gates independentes

- `npm --prefix web run build`: PASS, exit 0; TypeScript e build Vite.
- `.local/checks.py full quick vet build`: PASS, exit 0. Executa Go serial, banco sintético em 55432 e limites de memória definidos para Windows.
- Full: 54 testes principais no HEAD, 50 na base. Quick: 16 testes principais. Subcasos parametrizados não entram nessa contagem; nenhum teste foi removido, desativado ou pulado. Busca por t.Skip/Skipf/SkipNow não encontrou usos.
- Quatro testes novos mapeiam AC2–8. Nenhuma asserção antiga foi enfraquecida. A allowlist da PWA só recebe dois arquivos públicos locais específicos.
- Código alterado permanece no frontend e testes da apresentação. A autenticação e as regras de banco permanecem fora do diff. Não foram adicionadas dependências, endpoints, abstrações de plugin ou dados fictícios ao produto.
- Rechecagem independente após T4: build e TestBrowserSessionHighlightAndFilters passaram no scratch de d56622c; o mesmo M5 foi morto. T4 só mudou testes/specs/lesson. Os gates Full/Quick/vet/build acima foram executados no HEAD anterior 257a751 e permanecem evidência da mesma implementação. Root também repetiu Full httpapi após T4, com sucesso; esta repetição é evidência atribuída ao root.

## Sensor de discriminação

Scratch criado com git archive do HEAD validado; node_modules compartilhado por junction apenas para leitura. Cada mutante recebeu build próprio e os testes relevantes; builds passaram, portanto os resultados abaixo representam comportamento. Fontes reais do projeto nunca foram alteradas. Scratch e junction descartados após a execução. Porcelain real antes/depois do sensor: vazio, idêntico.

| Mutante | Local | Teste e resultado |
| --- | --- | --- |
| M1: selecionar items[1] como destaque | `web/src/Journal.tsx:43` | HighlightAndFilters, morto por UUID diferente do primeiro esperado. |
| M2: manter o primeiro também nos grupos com slice(0) | `web/src/Journal.tsx:44` | HighlightAndFilters, morto por contagem incorreta de registros. |
| M3: falha da imagem não ativa fallback | `web/src/Poster.tsx:6` | PosterFallbackAndKeyboard, morto pela espera da arte sem img quebrada. |
| M4: retirar DM Sans da lista do shell | `web/vite.config.ts:12` | LocalFonts, morto pela ausência do caminho exato no cache. |
| M5: contador visual constante 0/2 | `web/src/Journal.tsx:49` | Sobreviveu na primeira rodada. Após T4, HighlightAndFilters falhou em `session_browser_integration_test.go:45`, contador 0/2 diferente de 1/2. **Morto na rechecagem**. |
| D1: erro permanente na mesma instância de Poster | `web/src/Poster.tsx:6` | Diagnóstico passou. Excluído da contagem contratual: a aplicação desmonta destaque durante a busca; metadados não mudam em instância retida nesta feature. Não impor infraestrutura de hot refresh para um fluxo inexistente. Código atual já vincula erro ao caminho. |
| M6: destaque usa loading=lazy | `web/src/Poster.tsx:6` | PosterFallbackAndKeyboard, morto pela asserção eager/high. |

Resultado contratual final: 6 mutantes, 6 mortos, zero sobreviventes pendentes. Um experimento diagnóstico adicional sem falha alcançável no produto. Cinco resultados preservados da primeira rodada; M5 repetido no commit de correção. Sensor PASS. Na rechecagem, scratch e junction foram descartados e porcelain real permaneceu idêntico ao baseline, contendo somente o relatório ainda não commitado.

## Correção concluída

T4, commit d56622c, fortaleceu o teste de notas no destaque para exigir R=0, Gabriela=10, média=5 e texto `Média · 2/2` nos dois eventos da obra. Acrescentou contador 1/2 antes de salvar. Teste, build e M5 reexecutados no scratch. Lesson L-011 registrada pelo script como candidata. Nenhuma mudança funcional foi necessária.

RDS-01, RDS-02 e RDS-03 têm evidência compatível e podem ser marcadas como verificadas pelo orquestrador. Nenhuma credencial, sessão, cookie ou dado pessoal foi lido pelo Verifier. UAT humana do resultado final fica com Ronald; as capturas sintéticas e os testes não substituem a preferência visual dele.
