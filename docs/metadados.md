# Capas, sinopses e identificação

O histórico mostra uma sinopse em cada card quando a fonte oficial está disponível. O texto completo também aparece em **Informações da obra** e na avaliação. A legenda informa TMDB ou Netflix e avisa quando a sinopse está em inglês.

O TMDB é a fonte prioritária. A busca automática aceita nome, título original ou título alternativo oficial equivalente após normalizar caixa, acentos e pontuação. Não associa por popularidade, aproximação ou prefixo truncado. Por exemplo, o alias brasileiro “Naruto Shippuden: A Morte de Naruto!” identifica o filme TMDB 20982. Sem sinopse em português, o cliente tenta inglês e registra o idioma.

## Completar ou corrigir pelo app

1. No card, abra **Completar informações** ou **Corrigir informações**.
2. Busque pelo título, inclusive pelo nome original. Confira o tipo, o ano, a capa e a sinopse.
3. Escolha **Usar esta versão**. Só essa confirmação associa o resultado.
4. Se faltar material no TMDB, use **Usar informações oficiais da Netflix**. O servidor busca a página pública do ID da obra já salvo, sem cookies ou login. Não recebe uma URL de imagem enviada pelo navegador.

A escolha vale para todos os registros da mesma obra no mesmo streaming, entre as duas pessoas, inclusive próximos episódios. As notas, os comentários e os dados originais da coleta permanecem iguais. Uma série conhecida aceita somente série. Registros antigos sem ID confiável precisam de nova sincronização antes de corrigir informações.

O fallback Netflix foi autorizado como exceção ao TMDB. Usa somente material editorial vinculado à obra e imagens HTTPS em CDNs oficiais. A arte pode ser horizontal. Prefere a página BR; consulta GB se a versão brasileira estiver incompleta e registra o idioma. Páginas genéricas ou obras sem confirmação continuam pendentes. Não há scraping do Google nem sinopses inventadas.

## Reprocessar sem coletar novamente

Defina `TMDB_READ_ACCESS_TOKEN` no ambiente de API e worker. O Compose injeta o valor do `.env`; recrie os serviços para aplicar uma mudança de ambiente. Nunca use `VITE_*` para o token.

```sh
docker compose --env-file .env -f deploy/compose.yaml run --rm --no-deps worker /app/worker enrich
docker compose --env-file .env -f deploy/compose.yaml run --rm --no-deps worker /app/worker enrich --retry-unmatched
docker compose --env-file .env -f deploy/compose.yaml run --rm --no-deps worker /app/worker enrich --connection UUID_DA_CONEXAO --retry-unmatched
```

`enrich` não abre navegador, não descriptografa a sessão e não executa coleta. Também funciona se a conexão precisar de login. Processa até 500 eventos por perfil; repita para escoar lotes maiores. Sem a opção de repetição, respeita o backoff. Com ela, inclui não encontrados e ambíguos adiados; falhas transitórias continuam respeitando os 15 minutos. A agenda diária usa a mesma lógica após persistir a coleta.

As identificações confirmadas são persistidas em `metadata_overrides`, usando a identidade serviço/obra das notas. Um lock transacional compartilhado com o worker impede um resultado automático atrasado de trocar a escolha. O complemento Netflix usa `poster_url` separado de `poster_path` do TMDB. A migration 5 é aditiva e recusa rollback se isso apagaria identificações confirmadas.

## API privada

`GET /api/metadata/search?query=...&type=movie|tv` aceita consulta de 2 a 200 caracteres e tipo opcional. Retorna candidatos para escolha, sem salvar. `PUT /api/events/{id}/metadata` exige sessão, Origin autorizado e objeto JSON:

```json
{"provider":"tmdb","tmdb_id":20982,"media_type":"movie"}
```

Ou `{"provider":"netflix"}` para a página oficial do ID de origem. O servidor consulta os detalhes novamente, valida a identidade e grava atomicamente. A ausência do token TMDB informa indisponibilidade da busca; histórico e notas continuam disponíveis. Dados privados e imagens não entram no CacheStorage da PWA.

## Validação

Os testes de cliente usam HTTP controlado; os de API, banco e Chromium usam apenas schemas sintéticos em `TEST_DATABASE_URL`. Cobrem aliases, ambiguidade, truncamento, erros, idioma, identidade/CDN Netflix, concorrência, preservação de notas e navegação móvel por teclado. Antes da migration local, o backup é restaurado em banco temporário isolado e comparado ao original.

Na instalação local de 18/10/2026, os 1.248 eventos foram preservados e receberam imagem e sinopse: 1.246 usam TMDB e dois usam Netflix oficial (Cine Gibi e Harry Quer Casar: O Reencontro). A lista principal contém 193 itens e 192 obras distintas; uma revisita de filme continua aparecendo como evento próprio. Os 192 links distintos de imagem responderam HTTP 200 com tipo de conteúdo de imagem. Nomes truncados foram identificados pela página oficial e confirmados pelo nome completo; o matcher automático continua recusando prefixos. O registro originalmente sem nome foi identificado por seu ID público, preservando a origem.

A consulta pública Netflix usa transporte próprio HTTP/1.1, incluindo ALPN, porque a negociação HTTP/2 observada nesta instalação causava timeouts. A validação TLS, o proxy do ambiente e o cancelamento permanecem ativos; o transporte global e a rede do computador não são alterados.

Fontes: [aliases do Naruto](https://www.themoviedb.org/movie/20982-naruto/titles), [busca TMDB](https://developer.themoviedb.org/docs/search-and-query-for-details), [títulos alternativos de filmes](https://developer.themoviedb.org/reference/movie-alternative-titles), [títulos alternativos de séries](https://developer.themoviedb.org/reference/tv-series-alternative-titles).
