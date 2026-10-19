# Validação da configuração EasyPanel

**Verdict: PASS** — configuração e procedimento aprovados para iniciar a implantação. A operação na VPS, o HTTPS público e a importação dos dados ainda precisam ser comprovados pelo responsável pelo deploy.

**Data:** 2026-10-18. **Diff:** `3b05bb8..af8c94f`. **Verificador:** subagente independente `/root/verificador_deploy`, distinto do autor.

## Critérios de aceite

| Critério | Resultado esperado e evidência | Resultado desta revisão |
| --- | --- | --- |
| AC-1 | Os cinco serviços permanecem disponíveis, usando os Dockerfiles e targets existentes; nenhuma publicação de portas. `deploy/compose.easypanel.yaml:4`, `deploy/compose.easypanel.yaml:16`, `deploy/compose.easypanel.yaml:60` e `deploy/compose.easypanel.yaml:80`. Asserções executadas: `set(services) == {'postgres', 'migrate', 'api', 'worker', 'web'}` e `all(not item.get('ports') for item in services.values())`; imagens e targets também comparados aos valores previstos. | PASS |
| AC-2 | Sem perfil, somente postgres/migrate/api/web; com `collector`, inclui worker, chave, cron e fuso externos. `deploy/compose.easypanel.yaml:59`, `deploy/compose.easypanel.yaml:68` e `deploy/compose.easypanel.yaml:70`. Asserções executadas: conjunto de `config --services` igual aos quatro serviços iniciais, `worker.profiles == ['collector']` e valores de chave/agenda iguais ao ambiente sintético. | PASS |
| AC-3 | Origem explícita obrigatória, persistência e ordem de saúde preservadas. `deploy/compose.easypanel.yaml:22`, `deploy/compose.easypanel.yaml:37`, `deploy/compose.easypanel.yaml:45`, `deploy/compose.easypanel.yaml:49`, `deploy/compose.easypanel.yaml:75` e `deploy/compose.easypanel.yaml:85`. Foram verificados volume de tipo `volume`, destino `/var/lib/postgresql/data`, condições exatas `service_healthy`/`service_completed_successfully`, healthchecks e rejeição de `APP_ORIGINS=''` pelo Compose. Encaminhamento pelo proxy para web:80 está em `docs/easypanel.md:11`, compatível com `deploy/Caddyfile:1` e `deploy/Caddyfile:9`. | PASS na configuração; HTTPS/proxy reais pendentes |
| AC-4 | Procedimento preserva dados, hashes e chave de cifra, restaura somente no banco novo e exclusivo, mantém collector desligado até conferir importação e evita exposição do banco/dump. Evidências: `docs/easypanel.md:10`, `docs/easypanel.md:15`, `docs/easypanel.md:17` e `docs/easypanel.md:19`. `pg_restore` usa transação única, parada em erro, sem owner/privilégios importados; a comparação inclui UUIDs, eventos, notas, hashes e sessões cifradas. | PASS do procedimento; migração real pendente |

## Gate executado

- `rtk proxy python -X utf8 .local/verify-easypanel-independent.py`: 22 assertivas independentes passaram; zero falhas no gate original.
- O verificador usou `docker compose --env-file <arquivo temporário sintético> ... config`, ambiente filtrado e valores exclusivamente sintéticos. Não carregou o `.env` pessoal, não iniciou containers e não acessou bancos.
- `.github/workflows/checks.yml:71` acrescenta validação sintática do Compose EasyPanel com origem sintética; a revisão independente acrescentou verificações semânticas de perfis, portas e dependências. Nenhum teste existente foi removido ou enfraquecido.
- `rtk proxy git diff --check 3b05bb8..af8c94f`: sucesso.

## Sensor de discriminação

As seis alterações foram feitas somente em cópias temporárias do YAML. Cada cópia continuou sendo interpretada pelo Compose, e uma asserção de resultado detectou o defeito.

| Falha injetada | Evidência no arquivo original | Resultado |
| --- | --- | --- |
| Publicar web:80 como porta 8080 do host | `deploy/compose.easypanel.yaml:79` | Morto pela asserção de ausência de portas |
| Remover perfil do worker e executá-lo antes da restauração | `deploy/compose.easypanel.yaml:59` | Morto pela asserção dos quatro serviços iniciais |
| Aceitar origem ausente com fallback localhost | `deploy/compose.easypanel.yaml:45` | Morto pela exigência de erro ao omitir origem |
| Remover volume persistente do banco | `deploy/compose.easypanel.yaml:22` | Morto pela asserção de volume e destino |
| Trocar conclusão de migrations por simples inicialização | `deploy/compose.easypanel.yaml:49` | Morto pela asserção da condição exata |
| Substituir chave externa por valor fixo incorreto | `deploy/compose.easypanel.yaml:68` | Morto pela comparação com a chave sintética fornecida |

**Sensor:** 6/6 mortos, nenhum sobrevivente. As cópias temporárias foram descartadas e `git status --porcelain` antes/depois do sensor permaneceu idêntico. Nenhum arquivo real de implementação foi modificado pelo verificador.

## Qualidade e limites

A mudança usa os mesmos padrões do Compose existente, limita-se ao deploy e não altera a aplicação nem os serviços já instalados no EasyPanel. Não surgiram lacunas que exijam correção ou nova lição de projeto.

Esta aprovação não comprova o AC-4 operacional nem a disponibilidade pública do AC-3. Antes de concluir o pedido do usuário, registrar o commit exato de `main` com CI aprovado, destino HTTPS, `/api/readyz` 200, resultado da restauração com comparação de dados/credenciais e resultado da sincronização de ambos os perfis. Não foram executados deploy, restauração, redefinição de senha, alteração de branch, commit ou push nesta revisão.

## Operação comprovada na VPS

Após a revisão independente acima, o responsável pelo deploy implantou `main` no commit `3afd88c2bb103a7b1cf22194b21b6dcd03eab539`, com [CI aprovado](https://github.com/Ronald-Lacerda/my-watched-list/actions/runs/39189983616). Os PRs #1–#7 foram mesclados na branch principal por merge commits.

- AC-3 operacional: domínio HTTPS do EasyPanel encaminhado para `web:80`; certificado validado pelo cliente HTTPS; `/`, `/api/readyz` e `/manifest.webmanifest` responderam 200. `/api/history` sem autenticação respondeu 401. A tela de login foi aberta no navegador. Build Path `/deploy` mantém `.env` junto ao Compose e contexto `..` na raiz do repositório, conforme `docs/easypanel.md:8`.
- AC-4 operacional: dump custom PostgreSQL 17 de 184.154 bytes restaurado primeiro em banco temporário local e depois transferido por console autenticado ao PostgreSQL novo e exclusivo da stack. Checksum SHA-256 e tamanho conferidos antes da restauração. Comparação de contagem e digest do conteúdo completo das 11 tabelas aprovada antes de ativar o worker, incluindo usuários, hashes de senhas, sessões de autenticação, sessões Netflix cifradas, histórico, catálogo, notas e versão das migrations. Arquivos temporários da transferência removidos do container; o backup original permanece local e ignorado pelo Git.
- Coletor: perfil `collector` habilitado somente após a comparação. O worker iniciou com cron `0 4 * * *`, fuso `America/Sao_Paulo`, e completou os dois perfis sem erro. Nova coleta manteve os 1.248 eventos. A conferência após o sync confirmou conteúdo integral idêntico de users, ratings e watch_events; ambas as conexões ficaram `active`.
- Evidências privadas, fora do Git: `.local/production-import-verified.json`, `.local/deploy-restore-verified.png`, `.local/producao-integridade.png`, `.local/producao-worker.png` e `.local/producao-login.png`. O endereço público e os segredos ficam na configuração privada do deploy. Não foram redefinidas senhas nem criadas avaliações de demonstração.

**Resultado operacional: PASS.** Backups periódicos e testes de login com a senha digitada pelo usuário continuam sendo operações distintas; este deploy preservou os hashes existentes e verificou a autenticação obrigatória sem usar as senhas pessoais.
