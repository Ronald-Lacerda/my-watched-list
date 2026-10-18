# Operar a coleta Netflix

A CLI administrativa roda localmente, sem endpoints públicos. Cada conexão corresponde a um usuário e um perfil. Os perfis informados para este projeto são R e Gabriela. Cada sessão é cifrada antes de ser persistida; a chave fica somente no ambiente.

## Configurar

No `.env`, configure POSTGRES_PASSWORD, SESSION_ENCRYPTION_KEY (32 bytes em base64), TMDB_READ_ACCESS_TOKEN e, se necessário, SYNC_CRON/SYNC_TIMEZONE. O token TMDB é o **API Read Access Token**, enviado como Bearer. Nunca use variável VITE_* para esses segredos. Sem o token, a coleta persiste eventos e deixa metadados pendentes.

Se ainda não tiver token, entre na sua conta TMDB pelo navegador de desktop, acesse Configurações → API e solicite acesso seguindo o [guia oficial](https://developer.themoviedb.org/docs/getting-started). Copie o **API Read Access Token** da seção API para `TMDB_READ_ACCESS_TOKEN` no `.env` local, sem enviá-lo pelo chat. Essa é a credencial Bearer descrita na [documentação de autenticação](https://developer.themoviedb.org/docs/authentication-application). Depois recrie o worker com Compose para aplicar o ambiente e execute sync para processar pendências.

Para o login com navegador visível na máquina local, inicie o banco com a publicação opcional em loopback:

```sh
docker compose --env-file .env -f deploy/compose.yaml -f deploy/compose.admin.yaml up -d --wait postgres
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6201.1 install chromium
```

O arquivo admin publica somente PostgreSQL em 127.0.0.1:55433. Não o inclua no deploy de produção. Em VPS, faça login local usando um túnel SSH até o banco, mantendo a mesma chave de sessão no ambiente local e remoto. A transferência manual de cookies pelo chat não faz parte do fluxo.

Os comandos Go leem variáveis de ambiente. Para carregar o `.env` no PowerShell sem imprimir seus valores:

```powershell
Get-Content .env | Where-Object { $_ -and -not $_.StartsWith('#') } | ForEach-Object {
  $k, $v = $_.Split('=', 2)
  [Environment]::SetEnvironmentVariable($k, $v, 'Process')
}
$env:DATABASE_URL = "postgres://$($env:POSTGRES_USER):$($env:POSTGRES_PASSWORD)@127.0.0.1:55433/$($env:POSTGRES_DB)?sslmode=disable"
go run ./api/cmd/migrate up
```

Em Bash, exporte as variáveis no shell ou carregue um arquivo local confiável com `set -a; . ./.env; set +a`; valores com espaços como SYNC_CRON precisam estar entre aspas para esse uso. Exporte DATABASE_URL apontando ao banco local/túnel. O Compose lê o `.env` diretamente e aceita o cron sem aspas.

## Criar identidades e conectar perfis

```sh
go run ./worker/cmd/worker user-add --name R --email r@mywatchedlist.invalid
go run ./worker/cmd/worker user-add --name Gabriela --email gabriela@mywatchedlist.invalid
```

Guarde os UUIDs exibidos e use cada um no comando correspondente. Esses identificadores de e-mail são locais e não recebem mensagens. Os usuários são criados sem senha padrão, com acesso web desabilitado até a Fase 2. Não recrie identidades já existentes.

```sh
go run ./worker/cmd/worker login --user-id UUID_DO_USUARIO --profile R --channel chrome
go run ./worker/cmd/worker login --user-id UUID_DA_GABRIELA --profile Gabriela --channel chrome
```

O canal `chrome` usa uma janela isolada do Chrome instalado. Omita `--channel` para usar o Chromium instalado pelo Playwright. No Windows desta máquina, o Chrome instalado foi validado para o login visível; o worker usa Chromium headless. A sessão já aberta no navegador comum não é copiada nem modificada.

Na janela aberta, faça login e escolha o perfil solicitado. Mantenha a janela aberta e pressione Enter no terminal; o coletor a fecha ao salvar. A página de atividade é carregada para confirmar o GUID/nome do perfil atual. O identificador em browse pode diferir do identificador da atividade, por isso a confirmação usa profileInfo da atividade. Só após essa verificação a sessão é cifrada em memória e gravada. Um perfil diferente é rejeitado. Para reconectar, repita o comando com o mesmo usuário, referência da conta e perfil; o histórico é preservado.

Para conectar outro perfil da mesma conta sem repetir a senha, acrescente `--reuse-connection UUID_DA_CONEXAO_JA_SALVA`. O navegador reaproveita a autenticação cifrada, mas exige a seleção e confirmação do novo perfil antes de salvar uma sessão independente.

`--account principal` identifica a conta dentro do serviço. Se usar outra conta Netflix com o mesmo usuário, dê uma referência diferente para evitar misturá-las.

## Sincronizar e consultar status

```sh
go run ./worker/cmd/worker status
go run ./worker/cmd/worker sync
go run ./worker/cmd/worker sync --connection UUID_DA_CONEXAO
go run ./worker/cmd/worker serve
```

`status` exibe UUID, perfil, estado, último sucesso e código de erro. Não exibe ciphertext ou cookies. O daemon executa na inicialização e depois às 04:00 de America/Sao_Paulo, configurável por SYNC_CRON (cinco campos) e SYNC_TIMEZONE. Cada perfil usa advisory lock numa conexão PostgreSQL dedicada, inclusive no login, evitando disputa entre jobs e renovação de sessão.

`needs_login` exige reconexão pelo comando login; conexões nesse estado não entram no cron. `error` permite nova tentativa diária. Falhas não atualizam o último sync bem-sucedido. Cancelamento encerra navegador e libera lock.

No Compose, após conectar os perfis:

```sh
docker compose --env-file .env -f deploy/compose.yaml up --build -d --wait
docker compose --env-file .env -f deploy/compose.yaml exec worker /app/worker status
docker compose --env-file .env -f deploy/compose.yaml exec worker /app/worker sync
```

O app ainda não mostra notificações de sessão: isso será feito na Fase 2, consumindo os estados já persistidos.

## Contrato e limites da coleta

O navegador visita `https://www.netflix.com/viewingactivity`, lê o JSON inicial em `netflix.reactContext.models.vaModel.data` e captura o atom `jsonGraph.aui.viewingActivity.value` nas respostas Falcor de paginação. O site escolhe endpoint, build e parâmetros; o coletor aciona Mostrar mais/Show More e aceita respostas da própria origem referentes à atividade. A URL interna com build não é fixada. O GUID em `profileInfo` é conferido no início, na paginação e ao finalizar. O botão pode desaparecer ou ficar desabilitado ao terminar; o total de eventos únicos precisa coincidir com `vhSize`, quando informado. Mudanças nesse contrato falham explicitamente, sem salvar lote parcial.

O normalizador reconhece `viewedItems`, IDs movieID, date em epoch de milissegundos, título e pistas opcionais de série/episódio. Campo ausente ou lote com item inválido rejeita o lote inteiro. Título explicitamente em branco, observado na conta real e autorizado pelo usuário, vira “Título indisponível na Netflix”; ID/data continuam preservados e MatchTitle vazio impede busca TMDB. Título ausente/null continua inválido. Array explicitamente vazio é válido. A fixture no repositório é sintética, não é evidência de payload real atual da Netflix.

A chave `netflix:movieID:epoch_ms` não depende do título traduzido nem da posição na página. Reassistir em outro instante gera outro evento. Se a fonte colapsar duas visualizações no mesmo ID/instante, elas são indistinguíveis; não inventamos eventos. Dia local e instante UTC ficam separados. Limites: 1000 páginas e 16 MiB por resposta; atingir limite ou repetir página é erro, sem sucesso parcial.

O TMDB só é consultado após persistir o histórico. Só há associação automática com título/título original exato e único no tipo conhecido; tipo desconhecido busca filme e série. Sem nome de série confiável, não há tentativa de adivinhar o título a partir do episódio. Pôster é armazenado como caminho de imagem TMDB. Episódios exigem temporada/episódio explícitos e confirmação pelo endpoint de episódio.

Falha transitória TMDB adia a tentativa por 15 minutos; ausência/ambiguidade adia por um dia. As pendências são duráveis em watch_events e processadas na próxima execução, inclusive sync manual. Cada ciclo considera até 500 pendências por perfil; se necessário, execute novamente para escoar lotes maiores. O último sucesso da coleta é independente do enriquecimento. Não há fallback LLM ativo.

## Verificações

```sh
go test ./...
go vet ./...
docker compose -f deploy/compose.test.yaml up -d --wait
```

Exporte TEST_DATABASE_URL para o banco descartável de porta 55432 e execute `go test -tags=integration -count=1 ./...`. Os testes criam schemas próprios e não usam dados pessoais. Cobrem normalização, identidade, paginação em Chromium, perfil errado, expiração, mudanças de parser, cifragem, replay, lock concorrente, rollback e retry de metadados. Use `docker compose -f deploy/compose.test.yaml down -v` somente para o banco descartável de testes.

A coleta real foi validada em 18/10/2026 nos perfis R (557 eventos) e Gabriela (691 eventos). Repetir o sync manteve as mesmas contagens e chaves únicas. As sessões ficam somente no banco local cifrado; os testes e este repositório não contêm payloads pessoais. O usuário ainda não tem token TMDB: o enriquecimento foi validado com servidor controlado e PostgreSQL, e a chamada real ao TMDB permanece pendente da configuração do token.

Referências: [histórico Netflix por perfil](https://help.netflix.com/en/node/101917), [contexto e cookies Playwright](https://playwright.dev/docs/api/class-apirequestcontext), [busca TMDB](https://developer.themoviedb.org/docs/search-and-query-for-details).
