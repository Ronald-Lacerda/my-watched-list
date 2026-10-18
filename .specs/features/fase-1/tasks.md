# Tarefas da Fase 1

## Test Coverage Matrix

| Camada | Verificação |
| --- | --- |
| Sessão | Testes unitários de cifragem, nonce, contexto, adulteração e configuração |
| Normalização | Fixtures sintéticas, datas, ordem, tradução, vazio e erros |
| Navegador | Integração com site local controlado em Chromium, perfil e paginação |
| Persistência | PostgreSQL real; replay, concorrência, rollback e status |
| TMDB | Servidor HTTP controlado, match, ambiguidade e falhas |
| Agenda/CLI | Testes de orquestração e comandos sem rede real |
| Deploy | Compose/build/smoke e CI |

## Gate Check Commands

Quick: `go test ./...` e `go vet ./...`.
Full: `go test -tags=integration -count=1 ./...` com TEST_DATABASE_URL descartável; testes browser usam Chromium instalado.
Build: Quick + Full + `go build ./...`, `npm --prefix web run build`, Compose build/up e checks HTTP.

## Execution Plan

Uma entrega por componente, executada localmente em sequência; sete tarefas, sem delegação de implementação.

```text
T1 -> T2 -> T3 -> T4 -> T5 -> T6 -> T7
```

## Task Breakdown

### T1: sessões criptografadas
Where: internal/session
Depends on: none
Tests: unit
Gate: Quick
- [x] Cifrar, autenticar contexto da conexão e validar chave; NFLX-01. go test/vet aprovados; internal/session/cipher_test.go verifica igualdade do plaintext recuperado, nonce distinto e rejeição de contexto, chave, versão e ciphertext alterados. Integração da configuração no processo em T6.

### T2: normalizador Netflix
Where: worker/netflix
Depends on: T1
Tests: unit
Gate: Quick
- [x] Lote estrito, chave estável e datas corretas; NFLX-02. TestNormalize preserva campos e confere virada UTC/dia local, tradução e ordem. TestInvalidPayloadIsAtomic rejeita lote parcial, IDs/datas inválidos e schema ausente; vazio explícito aceito. go test/vet aprovados. Corrigido literal horário UTC da fixture sintética durante criação do teste, mantendo comparação exata e adicionando caso de virada do dia.

### T3: adaptador Playwright
Where: worker/netflix
Depends on: T2
Tests: integration browser
Gate: Full
- [x] Login assistido, perfil, paginação e falhas categorizadas; NFLX-03. TestBrowserCapture passou em Chromium nos cinco cenários: duas páginas, perfil errado, schema alterado, página repetida e redirecionamento ao login. go test/vet aprovados. Login Netflix real ainda depende da janela local; não é substituído por este teste.

### T4: repositório de coleta
Where: worker/store
Depends on: T3
Tests: integration postgres
Gate: Full
- [x] Migration de pistas de enriquecimento, lock, transação, status e replay concorrente; NFLX-04. TestPersistence em PostgreSQL Docker aprovado: round-trip cifrado, replay count=1, lock concorrente, revisita count=2, rollback de lote parcial e last_sync_at inalterado na falha. Suite completa com integration aprovada.

### T5: enriquecimento TMDB
Where: worker/tmdb
Depends on: T4
Tests: unit HTTP e integração postgres
Gate: Full
- [ ] Match conservador, metadados, episódios e pendências; NFLX-05.

### T6: comandos e agenda
Where: worker/cmd/worker
Depends on: T5
Tests: unit e integração
Gate: Full
- [ ] Bootstrap, login, status, sync e agenda com contexto; NFLX-06.

### T7: ambiente e operação
Where: deploy
Depends on: T6
Tests: build e smoke
Gate: Build
- [ ] Imagem Chromium, variáveis, CI, documentação e Docker recuperado; NFLX-07.

Dependências do diagrama coincidem com cada Depends on. Todos os componentes possuem testes na mesma tarefa. Usuário autorizou implementação, testes e PR; decisões simples seguem os padrões documentados, sem nova aprovação de ferramentas.
