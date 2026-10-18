# Fase 0: fundação do My Watched List

## Problem Statement

O projeto precisa de uma base executável para desenvolver coleta própria e registro de filmes e séries para duas pessoas. Esta entrega termina na infraestrutura e no schema; o histórico da Netflix começa na Fase 1.

## Out of Scope

Coleta real, login, armazenamento de cookies, criptografia operacional, chamadas TMDB, agenda diária, notas via API, rankings e instalação offline da PWA pertencem às fases seguintes. Nenhum dado pessoal será criado automaticamente.

## Assumptions & Open Questions

| Decisão | Padrão escolhido | Motivo |
| --- | --- | --- |
| Backend | Go 1.26, chi, pgx e goose; um módulo na raiz | Compartilhar contratos sem publicação de pacotes |
| VPS não informada | Linux com Docker Compose v2 | Base portátil; dimensionamento depende da VPS real |
| Sessão/perfil | Uma conexão por serviço, perfil e dono | Coleta atribuída explicitamente a uma pessoa |
| Notas | 0–10 em passos de 0,5 | Uma escala persistida; estrelas são apresentação futura |
| Datas do provedor | Data local obrigatória e instante opcional | Não inventar horário quando só houver dia |
| Idempotência | Chave estável do evento dentro da conexão | Permitir replays e revisitas com IDs distintos |
| Agenda | Cron no worker na Fase 1 | Sem fila extra nesta fundação |
| Outros serviços e VPS | Registrar no README como dados a informar depois | Não bloqueiam a Fase 0 |

**Open questions:** none. Ambiguidades desta entrega resolvidas pelos padrões acima, conforme autorização do usuário.

## User Stories

### P1: processos básicos (BASE-01)

Como desenvolvedor, quero iniciar a API e o worker para construir as próximas fases.

**Acceptance Criteria**:
1. WHEN receber GET /api/healthz, a API SHALL responder 200 com JSON {"status":"ok"}.
2. IF o banco não responder à verificação de prontidão, a API SHALL responder 503 em GET /api/readyz com JSON {"status":"unavailable"}.
3. WHEN o banco responder à verificação de prontidão, a API SHALL responder 200 em GET /api/readyz com JSON {"status":"ready"}.
4. IF DATABASE_URL estiver ausente, os processos SHALL encerrar com erro sem imprimir segredos.
5. The worker SHALL aguardar encerramento sem executar coleta nem anunciar sucesso de sincronização.

### P1: persistência (BASE-02)

Como desenvolvedor, quero migrations reversíveis e restrições para persistir o domínio corretamente.

**Acceptance Criteria**:
1. WHEN aplicar as migrations, o PostgreSQL SHALL criar users, streaming_accounts, titles, episodes, watch_events e ratings.
2. WHEN inserir novamente a mesma chave na mesma conexão com ON CONFLICT DO NOTHING, o banco SHALL manter exatamente um watch_event; uma chave distinta SHALL permitir outro evento.
3. IF a nota estiver fora de 0–10 ou fora de passos de 0,5, o banco SHALL rejeitar a gravação; notas 0, 0,5 e 10 SHALL ser aceitas.
4. IF um episódio não pertencer ao título do evento ou a pessoa não for dona da conexão, o banco SHALL rejeitar a gravação.
5. WHEN importar evento sem metadados nem horário, o banco SHALL aceitar o título de origem e a data preservados, com title_id, episode_id e watched_at nulos.
6. WHEN reverter e reaplicar as migrations em banco descartável, o schema SHALL voltar a ser utilizável.

### P1: execução documentada (BASE-03)

Como desenvolvedor, quero compilar o frontend e executar a base via Compose.

**Acceptance Criteria**:
1. WHEN compilar a web, o build SHALL produzir uma página em português que identifique a Fase 0 sem apresentar dados de exemplo como histórico real.
2. WHEN iniciar o Compose, a infraestrutura SHALL aguardar banco saudável e migrations concluídas antes de iniciar API e worker.
3. The ambiente local SHALL publicar somente a web em 127.0.0.1:8080; PostgreSQL e API permanecem na rede interna.
4. WHERE houver domínio configurado e portas públicas disponíveis, o Compose de produção SHALL oferecer Caddy nas portas 80 e 443 com HTTPS automático.
5. The repositório SHALL incluir .env.example sem credenciais reais, instruções de execução, decisões e roteiro das fases seguintes.

## Requirement Traceability

| ID | Entrega | Estado |
| --- | --- | --- |
| BASE-01 | Processos e contratos | verified |
| BASE-02 | Persistência | verified |
| BASE-03 | Web e deploy | pending |

## Plano de execução

1. API, worker, configuração e contratos: arquivos go.mod, go.sum, api/cmd/api, api/internal/httpapi, worker/cmd/worker, worker/collector, internal/config e arquivos de controle na raiz. Verificar com go test, go vet e go build. Commit feat(base).
2. Schema e executor: api/migrations, api/cmd/migrate e testes de integração. Verificar migrations up/down/up e invariantes em PostgreSQL descartável. Commit feat(db).
3. Web, deploy e documentação: web, deploy, .env.example, README e CI. Verificar build TypeScript/Vite, Compose, HTTP e Caddy. Commit build(deploy).

As dimensões presentes são integridade, replay, falha do banco, encerramento e limites de configuração. Autenticação e dependências externas de coleta não se aplicam ao comportamento desta fase.

## Evidência por entrega

- Entrega 1 concluída: go test ./..., go vet ./... e go build ./... aprovados. BASE-01.1–3: api/internal/httpapi/router_test.go compara status HTTP e corpo JSON exatos nos três cenários. BASE-01.4: internal/config/config_test.go exige o erro exato de configuração. BASE-01.5: worker/cmd/worker/main.go bloqueia no contexto, sem chamadas de coleta; revisão de código. Os testes de configuração e HTTP se limitam a BASE-01.
- Entrega 2 concluída: TestSchema aprovado em PostgreSQL 17.5 temporário. BASE-02.1/6: migrations up/down/up e verificação das seis tabelas; BASE-02.2/5: replay, revisita e campos sem enriquecimento; BASE-02.3: sete limites de nota; BASE-02.4: vínculos de pessoa e episódio rejeitados. Build e vet aprovados. Docker Desktop indisponível por erro local no socket dockerInference; PostgreSQL temporário usado para não alterar a instalação do usuário.
