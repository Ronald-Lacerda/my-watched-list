# Usar o app

A Fase 2 oferece histórico compartilhado, filtros por pessoa/streaming/período, notas individuais, média e pendentes de avaliação. Cada pessoa entra com seu username e sua própria senha. Os logins desta instalação são `ronald` (perfil R) e `gabriela`. O ambiente representa uma única casa: ambas podem consultar o histórico e as notas das duas pessoas.

## Visual Sessão

O app usa o tema escuro Sessão, com navegação superior no desktop e inferior no celular. O primeiro registro da seleção atual aparece em destaque, com pessoa, dia, streaming, notas individuais e média. Os demais continuam organizados por dia. Cada registro conserva sua identidade: assistidas de pessoas diferentes e episódios não são reunidos em um evento artificial.

Os filtros e as datas continuam no endereço e combinam entre si. Carregar mais mantém o destaque e acrescenta registros ao histórico. O destaque de pendentes também respeita a pessoa logada e os filtros; dar nota zero conta como avaliação.

Quando há metadados TMDB, o destaque usa o pôster da obra. Sem pôster ou com falha ao carregar, aparece arte abstrata local; título, pessoa, serviço, dia e notas continuam disponíveis. Nenhuma imagem ou avaliação de exemplo da proposta visual é incluída no produto. Configurar o token TMDB continua opcional e segue o [guia Netflix](netflix.md).

DM Sans e Barlow Condensed são servidas pelo próprio app e incluídas no shell offline. Os arquivos WOFF2 e licenças SIL Open Font License ficam em `web/public/fonts`; fontes de origem: [Google Fonts](https://github.com/google/fonts). O navegador não consulta Google Fonts em tempo de execução. O cache continua limitado a arquivos públicos e não guarda o histórico, as notas ou a sessão.

Para aplicar apenas a interface em uma instalação já saudável, sem recriar API, worker ou banco:

```sh
docker compose --env-file .env -f deploy/compose.yaml build web
docker compose --env-file .env -f deploy/compose.yaml up -d --no-deps web
```

Recarregue a página depois da atualização. O novo shell tem uma versão própria de cache e preserva o login existente. O redesign não exige migration, nova coleta, redefinição de senha ou novo convite.

## Usernames e primeiro acesso

O login usa apenas username e senha. Nomes são normalizados para minúsculas e aceitam 3–40 caracteres: começam com letra e continuam com letras ASCII, números ou `_`. Espaços externos são ignorados. O limite de tentativas usa a mesma identidade canônica, inclusive com caixa diferente. E-mail não autentica.

A migration 4 preserva as contas e lhes dá um username provisório único `user_...`. Para atribuir os logins amigáveis, use a CLI administrativa com os identificadores legados existentes:

```sh
docker compose --env-file .env -f deploy/compose.yaml exec api /app/admin username --email r@mywatchedlist.invalid --username ronald
docker compose --env-file .env -f deploy/compose.yaml exec api /app/admin username --email gabriela@mywatchedlist.invalid --username gabriela
```

Isso altera somente o login; senhas, sessões, IDs, notas e histórico permanecem. Para novas pessoas criadas por `worker user-add`, atribua também um username antes do primeiro acesso. O campo e-mail continua no cadastro administrativo legado do coletor, sem aparecer no formulário ou nos DTOs de autenticação. A CLI de convites ainda aceita `--email` para compatibilidade.

## Primeiro acesso

Suba a stack conforme o README. Os usuários existentes do coletor continuam sem senha até consumir um convite. Não há cadastro público nem senha padrão.

Com Docker, gere um arquivo pessoal para uma identidade já criada:

```sh
docker compose --env-file .env -f deploy/compose.yaml exec api /app/admin invite --username ronald --output /tmp/primeiro-r.html
docker compose --env-file .env -f deploy/compose.yaml cp api:/tmp/primeiro-r.html .local/primeiro-r.html
docker compose --env-file .env -f deploy/compose.yaml exec api rm /tmp/primeiro-r.html
```

Crie `.local` antes de copiar. Repita com `--username gabriela`, usando outro nome de arquivo. A CLI não cria usuários novos.

Abra o HTML no navegador. Ele leva ao app para escolher e confirmar uma senha de 12 a 128 caracteres. O convite expira em uma hora, vale uma vez e é removido do endereço ao abrir. Não envie o arquivo pelo chat nem o versione; apague-o após usar. Novo convite substitui o anterior. Para recuperar o acesso, gere outro arquivo; ao definir a nova senha, todas as sessões anteriores daquela pessoa são revogadas.

Fora do Docker, com `DATABASE_URL` e `APP_ORIGINS` no ambiente:

```sh
go run ./api/cmd/admin invite --username ronald --output .local/primeiro-r.html
```

O padrão de origem é `http://127.0.0.1:8080`; a primeira origem configurada é o destino do convite. Em produção, o Compose override usa `https://APP_DOMAIN`, e o cookie recebe `Secure`. Em desenvolvimento, API e Vite devem ser acessados pela mesma origem pública autorizada: para Vite na porta 5173, configure `APP_ORIGINS=http://127.0.0.1:5173` na API e reinicie-a. Não use URLs públicas HTTP para senhas.

## Histórico e notas

- **Histórico:** eventos mais recentes primeiro, 50 por página. Datas inicial e final incluem o dia selecionado. Os filtros se combinam e ficam no endereço; “Limpar filtros” volta ao histórico completo. A data original do streaming é preservada, sem inventar horário.
- **Dar minha nota:** aceite de 0 a 10 em passos de 0,5, com comentário opcional de até 2000 caracteres Unicode. Zero é uma nota válida. É possível editar e remover somente a própria avaliação.
- **Média:** usa as notas disponíveis, com contador. Uma nota 0 e outra 10 dão média 5. Uma única nota dá média igual a ela. A série inteira recebe uma nota por pessoa, compartilhada entre episódios e reassistidas daquele serviço.
- **Para avaliar:** uma ocorrência por obra sem nota da pessoa logada. A nota da outra pessoa não remove a sua pendência.
- **Conexões:** serviço, perfil, pessoa, último sync e estado. Quando pedir login, reconecte pelo [guia Netflix](netflix.md). O app não importa nem mostra cookies.

Avaliar não depende do TMDB. Até configurar o token, aparecem os nomes coletados e uma capa gráfica local; pôster, sinopse e gêneros surgem após enriquecimento. As notas usam a identidade da obra no streaming. Séries de outro serviço terão identidade própria nesta fase; associação entre catálogos fica para uma evolução futura.

Eventos de séries coletados antes da migration 3 precisam de novo sync para preencher o ID da série. Eles continuam visíveis com “Aguardando atualização”; execute `worker sync` conforme o guia. O replay preenche esse ID sem duplicar eventos nem apagar notas. Filmes antigos usam seu ID original. Registros sem título continuam preservados.

## Instalar e usar sem rede

No navegador compatível, use a opção de instalar/adicionar à tela inicial. HTTPS é necessário fora de localhost. Há manifest, ícones de 192/512 px e service worker versionado. O cache guarda apenas HTML, JS, CSS, ícones e as duas fontes públicas locais. Nenhuma rota `/api`, nota, histórico ou credencial entra no cache ou no armazenamento web.

Sem rede, o app avisa que não atualiza o histórico nem salva notas. Um formulário já aberto preserva o texto após falha e permite tentar novamente; não há fila de gravações offline. Ao reabrir offline, somente a estrutura pública é carregada. Reconecte e use “Tentar novamente”. Sessões duram até 14 dias e “Sair” revoga a sessão no servidor.

## Operação, testes e rollback

Antes de atualizar, faça backup de PostgreSQL e guarde a chave `SESSION_ENCRYPTION_KEY` separadamente. As migrations 3 e 4 são aditivas e preservam usuários, conexões, eventos, notas, senhas e sessões anteriores. `admin` está na mesma imagem da API. A API precisa de `APP_ORIGINS`, sem caminho, com origens separadas por vírgula e de um mesmo esquema HTTP/HTTPS. O override de produção usa exclusivamente o domínio HTTPS.

O rollback de migration para a Fase 1 é bloqueado quando existem notas novas sem `title_id`: isso impede perda silenciosa. Faça backup e planeje exportação/associação dessas notas antes de reverter o schema. Reverter somente as imagens mantém os dados novos no banco, mas a Fase 1 não oferece a interface de notas. Nunca use `down -v` no ambiente com histórico.

```sh
npm --prefix web ci
npm --prefix web run build
go vet ./...
go test ./...
go test -tags=integration -count=1 ./...
go build ./...
```

Integração requer `TEST_DATABASE_URL` de um PostgreSQL descartável (veja README) e Chromium instalado pelo playwright-go. Os testes criam schemas isolados com pessoas/eventos sintéticos; nunca apontam para `DATABASE_URL`. O build web é obrigatório antes dos testes Chromium. No Windows com pouca memória virtual, use `GOMAXPROCS=2`, `GOFLAGS=-p=1` e `GOMEMLIMIT=512MiB` e execute as verificações em sequência.

Os testes exercitam login/convite/expiração/logout, isolamento de notas, zero, concorrência, rollback, filtros/cursor, duas pessoas no browser, erros com retry, celular/teclado e PWA offline sem cache privado. O CI também sobe o Compose, verifica prontidão, manifest, histórico sem sessão (401) e Chromium no worker.

Reverter a migration 4 remove os usernames, mantendo os identificadores legados e credenciais; uma nova aplicação exige atribuir os logins amigáveis novamente.

Endpoints privados: `GET /api/auth/me`, `/api/filters`, `/api/history`, `/api/connections`; `PUT/DELETE /api/events/{id}/rating`. Atores vêm da sessão, nunca do corpo. Login e ativação usam `POST /api/auth/login` e `/api/auth/activate`; logout usa `POST /api/auth/logout`. Mutações exigem `Origin` permitido e JSON quando há corpo. Cookies são opacos, HttpOnly e SameSite=Lax; somente hashes de tokens ficam no banco. Senhas usam Argon2id com salt aleatório. O login limita tentativas persistidas por identificador; erros de credencial não distinguem pessoa inexistente de senha incorreta.
