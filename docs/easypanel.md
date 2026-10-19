# Deploy no EasyPanel

O EasyPanel administra HTTPS e encaminha o domínio para `web`, porta `80`. Use `deploy/compose.easypanel.yaml`; nenhum serviço publica portas no host. PostgreSQL 17 mantém os dados no volume `postgres_data` da stack. O Compose local continua independente.

## Preparação

1. Crie um serviço Compose chamado `my-watched-list`. Se a licença gratuita já tiver três projetos, use um projeto existente sem alterar seus outros serviços.
2. Em Source/Git, configure o repositório, branch `main`, Build Path `/deploy` e Docker Compose File `compose.easypanel.yaml`. Assim o `.env` é criado junto ao Compose e os contextos `..` dos builds apontam para a raiz do repositório. Para repositório público, use a URL HTTPS sem credenciais. Para repositório privado, use uma chave SSH exclusiva, cadastrada como deploy key somente de leitura no GitHub.
3. Em Environment, configure `POSTGRES_USER`, `POSTGRES_DB`, `POSTGRES_PASSWORD`, `SESSION_ENCRYPTION_KEY`, `TMDB_READ_ACCESS_TOKEN`, `APP_ORIGINS`, `SYNC_CRON` e `SYNC_TIMEZONE`, seguindo `.env.example`. `APP_ORIGINS` deve ser a URL HTTPS exata, sem barra final. Habilite a criação do arquivo `.env` para interpolação do Compose. Não coloque segredos na fonte YAML ou no Git.
4. Ao migrar sessões Netflix existentes, preserve a mesma `SESSION_ENCRYPTION_KEY`. Mantenha `COMPOSE_PROFILES` vazio até concluir a restauração. O worker pertence ao perfil `collector`.
5. Em Domains, associe o domínio HTTPS à porta `80` do serviço `web` e implante. O banco permanece acessível somente na rede interna. A API aguarda a conclusão das migrations e o frontend aguarda a saúde da API.

## Levar um banco existente

Crie um backup completo com `pg_dump -Fc` e confira sua restauração em um banco temporário local. Transfira o dump por canal autenticado até o PostgreSQL desta stack, sem publicar links ou abrir portas do banco. Não execute a restauração em outros bancos da VPS.

Na primeira implantação, as migrations criam o banco vazio da aplicação. Antes de liberar o acesso, restaure nele com `pg_restore --clean --if-exists --no-owner --no-privileges --single-transaction --exit-on-error`. O dump contém a versão das migrations. Esse passo substitui exclusivamente o banco recém-criado desta aplicação. Preserve o backup original e compare UUIDs, contagens de eventos, notas, hashes de senhas e sessões cifradas entre origem e destino.

Após conferir a restauração, acrescente `COMPOSE_PROFILES=collector` em Environment e implante novamente. O worker sincroniza na inicialização e depois no cron configurado. Uma eventual expiração da sessão Netflix é sinalizada no app; as senhas dos usuários permanecem válidas. O domínio novo exige entrar novamente com `ronald` ou `gabriela`.

## Verificação e recuperação

- Confira `GET /api/readyz` com status `200`, a tela de login em HTTPS, os dados restaurados e o resultado de sincronização de cada perfil.
- A confirmação de CI deve corresponder ao commit exato de `main` usado no deploy.
- Guarde a configuração privada e o dump fora do Git. Não altere nomes dos volumes nem use `down -v` em produção.
- Para reverter código, selecione um commit anterior compatível com as migrations. Reverter uma migration requer backup e plano específico; restaurar dados substitui o banco da aplicação.

Referência: [serviço Compose do EasyPanel](https://easypanel.io/docs/services/compose).
