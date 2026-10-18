# Validação da Fase 1

**Result**: FAIL

Os gates passam, mas cinco mutações de comportamento sobreviveram. Quatro critérios têm lacunas de regressão automatizada: uso da sessão pelo navegador, rejeições TMDB, ciclo do worker e comandos CLI.

- Data: 18/10/2026.
- Fonte: `.specs/features/fase-1/spec.md` e `tasks.md` no commit `61a1ddd`.
- Diff: `feat/fase-0..61a1ddd`.
- Verificador: subagente independente, sem autoria da implementação ou dos testes.
- Escopo: revisão somente leitura da implementação e testes reais. Mutações exclusivamente em cópia temporária dos arquivos rastreados de `61a1ddd`.

## Tarefas

T1–T7 estão marcadas concluídas em `tasks.md`. T1, T2, T4 e os gates de T7 têm evidência suficiente. T3, T5 e T6 precisam dos reforços descritos abaixo. Os estados `verified` da rastreabilidade não substituem este resultado independente.

## Critérios ancorados na spec

As referências abaixo apontam para as linhas de `61a1ddd`. Uma referência de implementação não substitui uma asserção de teste. Resultado: **9/13 ACs integralmente sustentados; 4 ACs com gaps**, além de uma divergência de precisão na decisão TMDB.

| AC | Resultado exigido | Evidência e expressão assertiva | Resultado |
| --- | --- | --- | --- |
| 1 | Envelope v1 cifrado, nonce aleatório, autenticação da conexão e ciphertext | `internal/session/cipher_test.go:24`: `a[0] != 1 || bytes.Equal(a,b) || bytes.Contains(a,plain)` falha; :28 exige plaintext igual; :32/:39/:46/:52/:57 exigem `errors.Is(err,ErrInvalid)` para contexto, adulteração, versão, chave e truncamento. Persistência cifrada faz round-trip em `worker/store/store_integration_test.go:73`. M1 morto. | PASS |
| 2 | Chave inválida interrompe antes da rede sem expor valor | `internal/session/cipher_test.go:65` exige erro seguro exato; `worker/cmd/worker/main_test.go:15` exige mensagem exata e `out.Len()==0`, com banco inacessível configurado. | PASS |
| 3 | ID, título, UTC, dia local e identidade estável em replay/tradução/ordem | `worker/netflix/normalize_test.go:25` compara ID `101`, título, chave `netflix:101:1725233400000`, instante `2024-09-01T23:30:00Z` e dia; :29 verifica virada UTC; :37/:42 exigem mesma chave após tradução e ordem inversa. | PASS |
| 4 | Payload inválido não retorna lote parcial; vazio aceito; título branco preserva identidade sem busca | `worker/netflix/normalize_test.go:51` exige `ErrParserChanged` e lote nil; :56 aceita vazio; :69 compara literal indisponível, ID, chave, instante e `MatchTitle==""`; :73 rejeita null com lote nil. | PASS |
| 5 | Aplicar sessão da conexão, conferir GUID, completar paginação e total sem parcial | `worker/netflix/browser_integration_test.go:54/:107/:113` exigem dois eventos distintos/IDs exatos/estado ou vazio; :63/:122 exigem erro categorizado e lote nil em repetição, parser, perfil, limite e total incompleto. M3 morto. Porém todos os `Fetch` dos testes usam `state=nil` (:52/:105). M8 ignora todo state informado e sobrevive: falta comprovar aplicação e isolamento de cookies por conexão no Chromium. | GAP |
| 6 | Login/perfil inválido marca needs_login sem eventos; parser marca error sem last_sync | `worker/syncer/syncer_integration_test.go:149` exige erro sem segredo, `count==0`, status exato e last nil; `worker/syncer/syncer_test.go:18` exige status/código exatos; `worker/store/store_integration_test.go:135` exige last anterior inalterado. Browser rejeita perfil em :63/:122. | PASS |
| 7 | Replay e concorrência mantêm um evento por conexão/chave; instante novo permite revisita | `worker/store/store_integration_test.go:86` exige `ErrBusy` na execução simultânea; :101 exige count 1 após dois saves; :114 exige count 2 na revisita. `worker/syncer/syncer_integration_test.go:124/:141` exige count 1 por conexão, usando a mesma identidade sintética em conexões diferentes. M2b morto. | PASS |
| 8 | Falha de lote não deixa eventos parciais nem último sucesso | `worker/store/store_integration_test.go:126` exige erro no lote com item inválido; :129 exige count 2, sem terceiro evento parcial; :135 exige último sucesso igual ao anterior. M4b morto. | PASS |
| 9 | Match exato único salva metadados; episódio só com numeração explícita válida | `worker/tmdb/client_test.go:66` compara ID, nome, sinopse, pôster, gênero e ano; :69 compara episódio e duração; `worker/store/store_integration_test.go:162/:186` compara metadados persistidos e temporada 1/episódio 2 vinculados. Entretanto M6 aceita qualquer nome e M7 ignora validação da resposta de episódio; ambos sobrevivem. Faltam resultados inexatos, tipo desconhecido/cruzado e episódios com ID/números inválidos ou divergentes. | GAP |
| 10 | Falha/ausência/ambiguidade mantém histórico sem vínculo incorreto e permite retry | `worker/tmdb/client_test.go:73/:77/:81` exige erros de ambiguidade, ausência e indisponibilidade; `worker/syncer/syncer_integration_test.go:124/:128/:141` exige histórico salvo, zero vínculos na falha e vínculo após retry; `worker/store/store_integration_test.go:150` exige `ambiguous`, title null e retry futuro persistido. M5 morto, inclusive segunda página ambígua. | PASS |
| 11 | Startup e cron processam conexões ativas, com exclusão PostgreSQL e cancelamento | `worker/syncer/syncer_test.go:30` exige próximo dia às 04:00; lock concorrente comprovado em `worker/store/store_integration_test.go:86`. Não há asserção de chamada inicial/novo ciclo de `Serve`, seleção de ativas/error versus needs_login/disabled ou saída/liberação por contexto. M9 remove a chamada de sincronização de todos os ciclos e sobrevive. | GAP |
| 12 | CLI cria usuário sem senha padrão, login assistido, status e sync sem cookies nos logs | `worker/cmd/worker/main_test.go:15` cobre somente configuração inválida. `worker/store/store_integration_test.go:59/:63` exercita métodos do store, não o despacho/saída CLI nem `password_hash`. M10 rejeita todo user-add e sobrevive. Não há asserções dos quatro fluxos positivos e sanitização de logs da CLI. | GAP |
| 13 | Imagem contém Chromium/driver compatíveis; worker depende de migrations; segredos backend | `deploy/Dockerfile.backend:22/:29/:31` fixa Playwright, instala browser e usa pwuser; `deploy/compose.yaml:69` exige migrations concluídas; :64/:65 injeta segredos apenas no worker, sem variáveis VITE. `.github/workflows/checks.yml:64/:67/:69` exige stack, HTTP e screenshot. Observação independente: stack existente ativo/healthy, HTTP 200 e screenshot Chromium executado como usuário padrão da imagem com exit 0. Compose config com segredos sintéticos e `--env-file NUL` passou. | PASS |

### Precisão da decisão TMDB

`.specs/features/fase-1/spec.md:20` limita associação a tipo conhecido, mas `docs/netflix.md:84` documenta busca em filme e série para tipo desconhecido e `worker/tmdb/client.go:79` a implementa. O comportamento esperado para tipo desconhecido deve ser explícito na fonte de verdade, incluindo unicidade entre ambos os tipos. Este relatório não altera a spec nem escolhe uma interpretação silenciosamente.

## Gates

| Gate | Resultado independente |
| --- | --- |
| `rtk proxy go test -json ./...` | Exit 0 |
| `rtk proxy go vet ./...` | Exit 0 |
| `rtk proxy go test -tags=integration -count=1 -json ./...` | Exit 0; 57 testes/subtestes aprovados; zero falhas e zero testes ignorados |
| `rtk proxy go build ./...` | Exit 0 |
| `rtk proxy npm --prefix web run build` | Exit 0, TypeScript e Vite |
| Compose config | Exit 0 com `--env-file NUL` e valores sintéticos, sem ler `.env` |
| Stack/HTTP | Containers worker/API/web ativos; API healthy; `/api/readyz` e `/` HTTP 200 |
| Chromium na imagem | `docker exec my-watched-list-worker-1 /app/playwright screenshot --browser chromium http://web/ /tmp/independent-smoke.png`, exit 0 |

Os três pacotes reportados pelo runner como `skip` são pacotes **sem arquivos de teste**, não testes pulados: api/cmd/api, api/cmd/migrate e worker/collector. A integração usou somente PostgreSQL de teste em `127.0.0.1:55432`, com schemas próprios criados/removidos pelos testes. Nenhum acesso ao banco pessoal de `55433`.

Contagem estática de funções `Test`: 3 antes da feature, 16 em `61a1ddd`, delta +13. Nenhum arquivo de teste anterior foi removido ou enfraquecido no diff. Build/up de imagens não foi repetido pelo verificador sobre a stack pessoal existente: a execução anterior está registrada em `tasks.md`, e os containers resultantes foram observados e usados no smoke independente. Não foram parados containers nem removidos volumes.

## Sensor de discriminação

Profundidade ampliada por sessões e integridade de dados. **10 mutações válidas: 5 mortas, 5 sobreviventes.** Cada mutação partiu do original de `61a1ddd`; testes reais permaneceram inalterados. Os comandos foram `rtk proxy go test -tags=integration -count=1 -json <pacote>`, com `-timeout=30s` nas repetições de persistência.

| ID | Local original | Mutação de comportamento | Resultado |
| --- | --- | --- | --- |
| M1 | `internal/session/cipher.go:40` | Zerar nonce depois de sua geração, repetindo o envelope para o mesmo estado | Morto por TestEnvelope, exit 1 |
| M2b | `worker/store/store.go:86` | Acrescentar relógio à chave na persistência, duplicando replay | Morto por TestPersistence, exit 1 |
| M3 | `worker/netflix/browser.go:250` | Desativar rejeição de GUID diferente na atividade paginada | Morto por TestBrowserFalcor/perfil-na-paginacao, exit 1 |
| M4b | `worker/store/store.go:90` | Commit por evento, permitindo persistência parcial antes da falha do lote | Morto por TestPersistence, exit 1 |
| M5 | `worker/tmdb/client.go:120` | Aceitar múltiplos candidatos | Morto por TestMatch/ambiguous e /multi_page, exit 1 |
| M6 | `worker/tmdb/client.go:73` | Comparação de títulos sempre retorna true | Sobreviveu, exit 0 |
| M7 | `worker/tmdb/client.go:180` | Desativar validação do ID/temporada/episódio da resposta | Sobreviveu, exit 0 |
| M8 | `worker/netflix/browser.go:57` | Ignorar todo storage state recebido | Sobreviveu, exit 0 |
| M9 | `worker/syncer/syncer.go:155` | Remover chamada de sincronização nos ciclos Serve | Sobreviveu, exit 0 |
| M10 | `worker/cmd/worker/main.go:84` | Rejeitar user-add mesmo com argumentos válidos | Sobreviveu, exit 0 |

Tentativas preliminares excluídas da contagem: M2 removeu ON CONFLICT e atingiu uma falha precoce com lease ainda aberto; o cleanup do teste bloqueou e somente o processo Go dessa tentativa foi encerrado. M4 trocou `tx.Exec` por `Conn.Exec` na mesma conexão, o que continua dentro da transação corrente e não constitui mutação semântica. As substituições M2b/M4b acima verificam os comportamentos desejados e foram mortas normalmente.

Isolamento: `git status --porcelain` real era vazio antes e depois de cada uma das duas rodadas. As duas cópias temporárias foram removidas e a igualdade foi conferida pelo script. Nenhum stash foi usado. O relatório e as lições foram escritos somente após encerrar o sensor. Logs estruturados temporários: `%TEMP%/fase1-independent-results.json` e `%TEMP%/fase1-independent-results-repeat.json`.

## Qualidade e limites

O diff acompanha os sete componentes planejados e mantém mudanças de infraestrutura, documentação e dependências relacionadas à coleta. As asserções de normalização e persistência comparam valores e estados completos, sem depender só da ocorrência de chamadas. Não encontrei remoção de testes nem funcionalidades alheias ao escopo. As lacunas de caminhos negativos TMDB, aplicação da sessão e despacho/orquestração impedem aprovar a expectativa de cobertura por camada.

As diretrizes aplicadas foram `C:/Users/Ronal/.codex/RTK.md`, a skill tlc-spec-driven e suas referências validate/sub-agents/coding-principles/lessons. Nenhum arquivo `.env`, `.local`, sessão ou histórico pessoal Netflix foi lido pelo verificador. A captura Netflix real anterior está documentada em `docs/netflix.md:98`; não foi repetida nesta revisão. Não há token TMDB real disponível: HTTP controlado e PostgreSQL foram validados; chamada real TMDB permanece pendente. UAT humano não foi substituído por este smoke técnico.

## Correções necessárias, em prioridade

1. **Major, AC 5:** criar teste browser controlado que exija o cookie/estado recebido e alterne dois perfis/sessões, provando que ignorar ou reutilizar o estado errado falha. Mata M8.
2. **Major, AC 9:** testar resultado inexato, igualdade por título original, tipo desconhecido com unicidade entre tipos, e resposta de episódio com ID/números inválidos/divergentes. Exigir ausência de vínculo nos casos inválidos. Mata M6/M7. Alinhar a decisão de tipo desconhecido na spec.
3. **Major, AC 11:** testar Serve iniciando e repetindo o ciclo, filtrando conexões e encerrando pelo contexto com liberação de lock. Remover o ciclo deve fazer o teste falhar. Mata M9.
4. **Major, AC 12:** testar CLI user-add/status/login/sync com banco controlado e fronteira browser controlada; conferir usuário desabilitado para senha, argumentos/saídas, perfil e ausência de cookies/chaves nos logs. Mata M10.

Revalidar após correções; não promover a feature a concluída enquanto houver sobreviventes. Este relatório não modifica tarefas, spec, código ou testes. As lições extraídas desses sinais são registradas somente por `lessons.py`.

Gate estrutural final: `validate_state.py fase-1` retorna exit 1 pelo verdict FAIL, conforme esperado. Foram registradas seis lições candidatas via script: cinco sobreviventes M6–M10 e a divergência de precisão TMDB. Nenhuma lição foi promovida a confirmada por uma única feature.
