# Design da Fase 1

Fluxo: CLI de login local → estado de sessão somente em memória → AES-GCM vinculado ao ID da conexão → PostgreSQL. Worker abre um contexto Chromium isolado, verifica o perfil, captura JSON da atividade e aciona o botão de paginação do site. O normalizador devolve um lote completo ou erro. A persistência insere o lote em transação e usa a unicidade existente para replay. Enriquecimento TMDB é posterior à persistência, com pendências duráveis nos eventos.

Reutilizar collector.Event, pgx, migrations goose, erros de login/parser e configurações existentes. Uma conexão PostgreSQL dedicada mantém advisory lock durante a sincronização; a gravação da sessão pelo login usa o mesmo lock. Não manter transação aberta durante navegação ou requisições TMDB. Encerramento fecha o browser e libera lock.

## Risks & Concerns

| Risco | Mitigação |
| --- | --- |
| API interna muda e não tem contrato público | Descobrir resposta real, validar schema, não registrar corpo bruto e não marcar sucesso parcial |
| Perfil errado na mesma conta | Verificar GUID no contexto do site antes/depois da coleta e no login |
| Falha TMDB após histórico | Persistir primeiro; guardar pistas de match e repetir em execução seguinte |
| Correspondência por texto pode ser errada | Exigir match exato único; não usar resultado popular como decisão automática |
| Cookies e erros do browser contêm segredos | Criptografar em memória; propagar erros categorizados, sem mensagens do driver |
| Docker local indisponível | Atualização oficial assinada; dados persistentes preservados |
| Teste mock não prova Netflix ao vivo | Registrar separadamente captura real e testes determinísticos; não declarar validação real sem login |
