# Validação da Fase 0

**Veredito: PASS.** Os 16 critérios de aceite correspondem aos resultados observados. Dez mutações foram detectadas pelos testes. Não há lacunas de implementação, cobertura ou precisão da spec nesta revisão.

- Data: 2026-10-18.
- Spec: `.specs/features/fase-0/spec.md`.
- Diff revisado: `c5e13e2..a2c0b41784b8affa9aee59a3efb2bb1c863db1a5` (`main..HEAD` no início da revisão).
- Verificador: subagente independente; autor diferente do verificador.
- Limite de escrita: somente este relatório na árvore real. Nenhum código, teste, migration, commit ou push foi alterado pelo verificador.
- Revisão complementar: `a2c0b41..4b00305` contém apenas atualização das referências oficiais de actions para v7 e uma linha de evidência na spec. Critérios, implementação, testes e linhas citadas permaneceram idênticos. O sensor e os gates de domínio continuam aplicáveis; a nova execução remota do workflow deve ser conferida antes de encerrar o PR.

## Entregas

Não existe `tasks.md`: a spec registra três entregas no plano de execução.

| Entrega | Resultado | Evidência |
| --- | --- | --- |
| Processos e contratos | Concluída | Testes HTTP/configuração, build, vet e execução dos binários. |
| Schema e executor | Concluída | PostgreSQL real descartável, up/down/up e invariantes. |
| Web, deploy e documentação | Concluída | Build TypeScript/Vite, configuração Compose/Caddy, smoke nativo e containers na CI Linux. |

## Critérios e resultados definidos pela spec

As referências abaixo usam linhas do commit revisado. Para configuração, documentação e o worker sem lógica de domínio, inspeção e comandos operacionais complementam os testes proporcionais ao risco.

| AC | Resultado exigido | Evidência independente e expressão verificada | Resultado |
| --- | --- | --- | --- |
| BASE-01.1 | GET health: HTTP 200 e `{"status":"ok"}` | `api/internal/httpapi/router_test.go:16`, `api/internal/httpapi/router_test.go:24`: `w.Code == tt.code && w.Body.String() == tt.body`, com `tt.code=200` e corpo exato. Repetido contra Caddy nativo com `code == 200 && json.loads(body) == {'status':'ok'}`. | PASS |
| BASE-01.2 | Banco indisponível: ready HTTP 503 e `{"status":"unavailable"}` | `api/internal/httpapi/router_test.go:18`, `api/internal/httpapi/router_test.go:24`: comparação exata de 503 e corpo. API compilada apontada à porta de banco inacessível 55433: `code == 503 && json.loads(body) == {'status':'unavailable'}`. | PASS |
| BASE-01.3 | Banco disponível: ready HTTP 200 e `{"status":"ready"}` | `api/internal/httpapi/router_test.go:17`, `api/internal/httpapi/router_test.go:24`: comparação exata de 200 e corpo. Repetido no proxy nativo com PostgreSQL real e JSON exato. | PASS |
| BASE-01.4 | Sem DATABASE_URL, processos encerram com erro sem imprimir segredos | `internal/config/config_test.go:7`: `err != nil && err.Error() == "DATABASE_URL é obrigatória"`. `api/cmd/api/main.go:18`, `api/cmd/api/main.go:26`, `worker/cmd/worker/main.go:14`, `api/cmd/migrate/main.go:31`: propagação para saída 1. Binários API, worker e migrate executados sem a variável: `returncode == 1`, mensagem exata em stderr e ausência de segredo sentinela. | PASS |
| BASE-01.5 | Worker aguarda encerramento, sem coleta nem sucesso de sincronização | `worker/cmd/worker/main.go:18`, `worker/cmd/worker/main.go:20`, `worker/cmd/worker/main.go:21`: contexto de sinais, aviso explícito da Fase 1 e bloqueio em `<-ctx.Done()`, sem chamada de coleta. Binário continuou vivo após observação (`poll() is None`), com aviso esperado e sem mensagem de sync bem-sucedido. Encerramento gracioso por sinal foi inspecionado; o processo de verificação no Windows foi terminado pelo harness. | PASS |
| BASE-02.1 | Criar as seis tabelas | `api/migrations/migrations_integration_test.go:58`, `api/migrations/migrations_integration_test.go:60`: para cada uma de users, streaming_accounts, titles, episodes, watch_events e ratings, `err == nil && name == table` via `to_regclass`. | PASS |
| BASE-02.2 | Replay mantém um evento por chave/conexão e chave distinta permite outro | `api/migrations/migrations_integration_test.go:67`, `api/migrations/migrations_integration_test.go:68`, `api/migrations/migrations_integration_test.go:72`, `api/migrations/migrations_integration_test.go:76`: ON CONFLICT DO NOTHING com chaves 1,1,2; total `count == 2` e `count == 1` para cada chave. | PASS |
| BASE-02.3 | Aceitar 0, 0,5, 10; rejeitar fora do intervalo ou passo 0,5 | `api/migrations/migrations_integration_test.go:88`, `api/migrations/migrations_integration_test.go:92`, `api/migrations/migrations_integration_test.go:93`, `api/migrations/migrations_integration_test.go:96`: 0/0.5/10 exigem `err == nil`; -0.5/10.5/7.2/8.01 exigem `err != nil`. `api/migrations/00001_initial.sql:90` usa numeric sem arredondamento e limites/passos completos. | PASS |
| BASE-02.4 | Rejeitar episódio de outro título e evento de pessoa que não possui a conexão | `api/migrations/migrations_integration_test.go:104`, `api/migrations/migrations_integration_test.go:105`: pessoa existente diferente exige erro. `api/migrations/migrations_integration_test.go:116`, `api/migrations/migrations_integration_test.go:117`: episódio existente de outra série exige erro. As respectivas FKs removidas no sensor fizeram esses testes falharem. | PASS |
| BASE-02.5 | Preservar origem/dia sem enriquecimento, com três campos nulos | `api/migrations/migrations_integration_test.go:83`, `api/migrations/migrations_integration_test.go:84`: exige `title == "Título de origem" && day == "2026-10-18" && !titleID.Valid && !episodeID.Valid && !instant.Valid`, além de ausência de erro. Todos os componentes do payload são comparados. | PASS |
| BASE-02.6 | Reverter e reaplicar, mantendo schema utilizável | `api/migrations/migrations_integration_test.go:46`, `api/migrations/migrations_integration_test.go:49`, `api/migrations/migrations_integration_test.go:52`: Up/Down/Up sem erro; `api/migrations/migrations_integration_test.go:55`: novo Up sem migrações pendentes (`len(applied) == 0`). As tabelas e inserções dos demais casos são verificadas após esse ciclo. | PASS |
| BASE-03.1 | Build gera página em português, identifica Fase 0, sem histórico fictício | `web/package.json:11`: `tsc --noEmit && vite build` executado com saída 0. `web/index.html:2`: `lang="pt-BR"`. `web/src/main.tsx:35`, `web/src/main.tsx:36`, `web/src/main.tsx:39`: FASE 0, coleta futura e histórico ainda futuro, sem lista de eventos fictícios. HTML servido e fallback SPA conferidos por HTTP. | PASS |
| BASE-03.2 | Banco saudável e migrations concluídas antes da API/worker | `deploy/compose.yaml:34`, `deploy/compose.yaml:43`, `deploy/compose.yaml:58`: cadeia `postgres: service_healthy` → migrate → `service_completed_successfully` para ambos os processos. JSON renderizado conferido com igualdade dessas condições. CI executou Compose up com build/--wait e obteve readyz e raiz com sucesso. | PASS |
| BASE-03.3 | Ambiente local publica apenas web em 127.0.0.1:8080 | `deploy/compose.yaml:68`: única publicação de portas. No JSON renderizado: `len(web.ports) == 1`, `host_ip == '127.0.0.1'`, `published == '8080'`; todos os outros serviços sem `ports`. A porta 55432 pertence somente ao arquivo separado de testes, `deploy/compose.test.yaml:11`. | PASS |
| BASE-03.4 | Produção oferece Caddy 80/443 e HTTPS automático quando há domínio/portas | `deploy/compose.production.yaml:5`, `deploy/compose.production.yaml:6`: domínio obrigatório e substituição de portas; JSON renderizado exige 80/TCP, 443/TCP e 443/UDP. `deploy/Caddyfile:1`: domínio é endereço do site. Caddy validate executado com domínio de teste saiu 0 e registrou ativação de TLS e redirecionamento HTTP→HTTPS. Emissão ACME real depende da futura VPS/domínio e não foi simulada. | PASS |
| BASE-03.5 | .env.example sem credenciais reais, execução, decisões e fases seguintes | `.env.example:5`, `.env.example:8`, `.env.example:12`: campos sensíveis vazios; usuário/banco são rótulos. `README.md:33`, `README.md:50`, `README.md:82`, `README.md:118`, `README.md:155`, `README.md:175`: decisões, execução, verificações, produção e roteiro. `.gitignore:1`, `.dockerignore:2` excluem arquivos reais de ambiente. | PASS |

**Cobertura: 16/16 ACs. Lacunas de precisão: 0.** O teste adicional de episódio em filme (`api/migrations/migrations_integration_test.go:121`) cobre o vínculo título/episódio e a decisão registrada em `README.md:46`.

## Gates e evidência operacional

Execuções locais independentes em cópia extraída de `git archive a2c0b41`, fora da árvore real:

| Comando/verificação | Resultado |
| --- | --- |
| `rtk proxy go test -count=1 -v ./...` | Saída 0. HTTP e configuração passaram. |
| `rtk proxy go test -tags=integration -count=1 -v ./...` | Saída 0. Três testes de nível superior e 14 subtestes; 15 casos folha aprovados, zero falhas, zero skips. |
| `rtk proxy go vet ./...` | Saída 0. |
| `rtk proxy go build ./...` | Saída 0. |
| `rtk proxy gofmt -l api internal worker` | Nenhum arquivo listado após restauração byte a byte da cópia. |
| `rtk proxy npm ci` e `rtk proxy npm run build`, em web | Saída 0 em ambos; TypeScript e Vite aprovados. |
| Compose `config --quiet`, local, produção e testes | Saída 0 nos três; JSON renderizado também conferido quanto a dependências e portas. |
| Caddy `validate --config deploy/Caddyfile --adapter caddyfile`, com domínio de teste | Saída 0; TLS e redirecionamento automático habilitados. |
| Binários independentes sem DATABASE_URL | API, worker e migrate saíram 1, com mensagem exata e sem segredo sentinela. |
| API independente com banco inacessível | health 200/ok; ready 503/unavailable. |
| Stack nativa já ativa em 127.0.0.1:18081 | health/ready: 200 e JSON exato; web pt-BR e fallback SPA: 200 e HTML iguais. |
| Worker independente | Processo permaneceu aguardando; log avisa que coleta/agenda virão na Fase 1. |

Banco usado: PostgreSQL temporário em 127.0.0.1:55432, banco `watched_test`. Cada execução criou schema aleatório e o removeu ao finalizar, conforme `api/migrations/migrations_integration_test.go:29`, `api/migrations/migrations_integration_test.go:34` e `api/migrations/migrations_integration_test.go:38`. Não foi usado schema public para os testes de domínio e o servidor não foi parado.

O baseline `c5e13e2` é uma árvore Git vazia: contagem anterior zero testes. A entrega acrescentou três testes de nível superior e 14 subtestes. Nenhum teste anterior foi removido ou enfraquecido.

### Containers e race detector na CI

O Docker Desktop local não iniciou devido ao erro de socket dockerInference. Nenhum runtime Docker local é declarado validado.

A [execução 39166847423 do GitHub Actions](https://github.com/Ronald-Lacerda/my-watched-list/actions/runs/39166847423) foi confirmada independentemente via `gh run view`: `headSha=a2c0b41784b8affa9aee59a3efb2bb1c863db1a5`, estado completed e conclusão success. Os três jobs passaram:

- Go: gofmt, vet, `go test -tags=integration -race -count=1 ./...` em PostgreSQL 17 e build.
- Web: npm ci e build.
- Containers: configuração local/produção, `docker compose -f deploy/compose.yaml up --build -d --wait`, GET `/api/readyz`, GET `/` e descarte dos volumes de teste.

Essa execução comprova build dos Dockerfiles e runtime da stack no runner Linux. Publicação real em VPS, DNS e certificado de um domínio real continuam condicionados ao ambiente futuro definido pela spec.

## Sensor de discriminação

Profundidade ampliada por integridade de dados. Dez mutações independentes foram aplicadas apenas na cópia temporária. Os testes do repositório permaneceram intactos. Cada mutação foi revertida antes da seguinte; as sete primeiras executaram `go test -tags=integration -count=1 -v ./api/migrations`, e as três últimas executaram `go test -count=1 -v ./api/internal/httpapi`, sempre por RTK.

| Mutação | Código original | Falha introduzida | Resultado observado |
| --- | --- | --- | --- |
| M1 | `api/migrations/00001_initial.sql:79` | Acrescentar id à unicidade, retirando a identidade por conexão/chave | Morta; replay falhou em `migrations_integration_test.go:69`, SQLSTATE 42P10 porque o ON CONFLICT exigido perdeu sua restrição. |
| M2 | `api/migrations/00001_initial.sql:76` | Remover FK que exige dono da conexão | Morta; `migrations_integration_test.go:106`: evento de pessoa diferente aceito. |
| M3 | `api/migrations/00001_initial.sql:77` | Remover FK episódio/título | Morta; `migrations_integration_test.go:118`: episódio de outro título aceito. |
| M4 | `api/migrations/00001_initial.sql:90` | Reduzir limite inferior para -1 | Morta; `migrations_integration_test.go:97`: -0.5 indevidamente aceito. |
| M5 | `api/migrations/00001_initial.sql:90` | Elevar limite superior para 11 | Morta; `migrations_integration_test.go:97`: 10.5 indevidamente aceito. |
| M6 | `api/migrations/00001_initial.sql:90` | Remover módulo do passo 0,5 | Morta; `migrations_integration_test.go:97`: 7.2 e 8.01 indevidamente aceitos. |
| M7 | `api/migrations/00001_initial.sql:90` | Trocar numeric por numeric(3,1), arredondando antes do CHECK | Morta; `migrations_integration_test.go:97`: 8.01 arredondado e aceito. |
| M8 | `api/internal/httpapi/router.go:17` | health responde 201 em lugar de 200 | Morta; `router_test.go:25` compara status 201 recebido com 200 esperado. |
| M9 | `api/internal/httpapi/router.go:23` | ready com banco indisponível responde 200 em lugar de 503 | Morta; `router_test.go:25` compara status 200 recebido com 503 esperado. |
| M10 | `api/internal/httpapi/router.go:26` | ready com banco disponível responde status JSON ok em lugar de ready | Morta; `router_test.go:25` compara os corpos exatos. |

**Resultado: 10 injetadas, 10 mortas, 0 sobreviventes.** Todas saíram 1 por falha do teste correspondente. Após restaurar os arquivos originais byte a byte, o gate completo com integração passou novamente.

Isolamento: a árvore real começou com `git status --porcelain` vazio. A cópia de código do sensor foi descartada; depois do descarte e antes de escrever este relatório, o porcelain real permaneceu byte a byte igual ao baseline vazio. Nenhum git stash foi usado. Scripts e logs operacionais temporários não fazem parte do repositório.

## Qualidade e limites

- Mudanças restritas à fundação descrita pela spec; coleta real, login e dados pessoais continuam fora do escopo.
- HTTP compara status e corpo; persistência compara contagens, identidade da origem, data e nulidade, sem se limitar a verificar chamadas.
- Testes de rejeição foram discriminados removendo as próprias restrições, de forma que erros alheios não mascararam os vínculos avaliados.
- Configuração e documentação foram conferidas com saídas renderizadas e processos reais; sem testes triviais que apenas repetem os arquivos.
- Não há fluxo complexo de interface que exija UAT interativo nesta entrega. O conteúdo inicial foi inspecionado e compilado.
- Diretrizes aplicadas: `C:/Users/Ronal/.codex/RTK.md` e os princípios da skill tlc-spec-driven. Sem grafo Graphify prévio; nenhum artefato adicional foi criado na árvore real, respeitando o escopo do verificador.

## Lacunas ranqueadas e rastreabilidade

Nenhuma lacuna acionável identificada. BASE-01, BASE-02 e BASE-03 estão verificados no commit revisado. O verificador não alterou a spec, conforme seu limite de escrita.

Não há sinal de falha para destilar em lessons: nenhum mutante sobrevivente, AC sem cobertura, desvio de spec ou lacuna de precisão.

**Overall: PASS.** Fase 0 pronta para revisão do PR. Este relatório não autoriza implantação na VPS nem avança o escopo para a Fase 1.
