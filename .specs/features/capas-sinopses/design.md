# Identificação de metadados

Cliente TMDB compartilhado em internal/metadata, com MatchInput e Title próprios; worker/tmdb mantém adaptador compatível para collector.Event. Os testes existentes do cliente acompanham a mudança de diretório, preservando asserções. Search e Details são reutilizados pela API, sem token no frontend.

Migration 00005 adiciona metadata_overrides(source_key PK, provider, title_id FK opcional, display_title, overview, overview_language, poster_url, updated_by FK, updated_at) e idioma da sinopse em titles. source_key usa a mesma identidade serviço/obra das notas; ausência de chave confiável bloqueia correção em lote. A confirmação atualiza o vínculo de todos os eventos correspondentes, preservando campos de origem e avaliações. Episódios vinculados a outro título têm apenas o vínculo de metadados limpo, preservando números e origem. O complemento Netflix não precisa inventar ID TMDB; a leitura agrega seus campos por chave, inclusive em novos episódios. poster_url fica separado de poster_path e só aceita HTTPS de CDNs oficiais.

internal/catalogstore compartilha upsert de título e lock transacional por source_key. Confirmação e Link do worker usam o mesmo lock; o worker verifica override depois de adquirir o lock para que um resultado automático iniciado antes da confirmação não a substitua. Overrides persistidos fornecem metadados para novos eventos sem depender de nova busca.

API privada: GET /api/metadata/search?query=...&type=movie|tv (type opcional); PUT /api/events/{id}/metadata com provider=tmdb, tmdb_id e media_type, ou provider=netflix para buscar a página pública do ID já salvo. A API consulta detalhes oficiais novamente antes de gravar; não recebe URL de imagem ou sinopse do cliente. Sem token, a busca TMDB informa indisponibilidade e histórico/notas continuam disponíveis. Netflix é fallback explícito para informação ausente; nunca usar OG genérico ou transferir cookies. Português preferido, GB usado se a obra estiver indisponível em BR; idioma registrado.

Journal mostra sinopse e botão discreto de informações. MetadataDialog segue modal/foco do RatingDialog, busca abortável e candidatos com capa/ano/sinopse. Confirmar recarrega o histórico para atualizar todas as ocorrências e o agrupamento. Avaliações mantêm seu fluxo e chave.

CLI worker enrich [--connection UUID] [--retry-unmatched] processa somente metadados de eventos existentes. Sync diário continua coletando e enriquecendo, respeitando overrides. Lote de até 500 por perfil e backoff existentes são preservados.
