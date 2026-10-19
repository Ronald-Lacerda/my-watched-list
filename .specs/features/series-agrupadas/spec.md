# Séries agrupadas no histórico

## Problem Statement

Ronald quer uma única entrada por série na lista principal e uma nota por série, sem avaliar cada episódio. A nota por obra já existe; a mudança agrega a apresentação sem apagar as assistidas originais.

## Out of Scope

Agrupar filmes, unir catálogos de streamings diferentes, mudar notas existentes, editar/excluir episódios, criar tela detalhada de episódios, novas migrations ou alterar o coletor.

## Assumptions & Open Questions

| Decisão | Escolha | Motivo |
| --- | --- | --- |
| Identidade | Série por serviço e rating_key existente, entre perfis/pessoas | Não depender de título textual ou TMDB, manter a identidade da nota |
| Filtros | Aplicar pessoa/streaming/período antes de agrupar | Data, participantes e quantidade refletem o recorte visível |
| Representante | Evento mais recente por data e UUID | Cursor estável e mutações de nota existentes |
| Filmes | Continuam por evento | Pedido se refere às séries |
| Séries legadas sem chave | Preservar registros separados e aviso de sincronização | Não adivinhar identidades nem permitir nota por episódio |
| API | group_series=true no app; padrão false preserva leitura bruta | Manter consumidor de eventos e testes de contratos anteriores |
| Contador | Quantidade de registros no recorte, não episódios únicos | Reassistidas também são eventos; evitar progresso inventado |
| Entrega | PR separado baseado em codex/redesign-sessao, atualizar API/web locais após testes | Fluxo do projeto autorizado; não mesclar PRs |

**Open questions:** none.

## User Stories

**Acceptance Criteria**:

1. WHEN o app consultar histórico ou pendentes, a API SHALL agrupar as séries com chave confiável antes da paginação, exibindo uma entrada por serviço/obra entre pessoas, preservando todos os eventos de filmes e a leitura bruta sem group_series.
2. WHEN filtrar pessoa, serviço ou datas inclusivas, o resumo da série SHALL conter somente participantes, quantidade e data mais recente desse recorte; grupos e total SHALL respeitar o filtro.
3. WHEN carregar páginas, a série SHALL aparecer somente uma vez, ordenada pela última assistida e UUID do representante; cursor e total SHALL operar sobre os itens agrupados.
4. WHEN avaliar uma série pelo item agregado, o app SHALL indicar que a nota vale para a série inteira e manter uma avaliação por pessoa/obra, incluindo zero, média e contador exatos; editar/remover SHALL preservar a nota da outra pessoa e atualizar pendentes.
5. WHEN renderizar a série agrupada, o app SHALL mostrar título da série, participantes, quantidade de registros do recorte e indicação de última assistida, sem título de episódio; filmes SHALL manter suas informações e ações.
6. IF faltar TMDB ou chave confiável da série, o app SHALL preservar os registros e fallback de imagem existentes, sem agrupar por nomes iguais ou inventar uma identidade; séries sem chave SHALL continuar bloqueadas para nota com aviso de sincronização.
7. WHEN atualizar localmente, a implantação SHALL preservar eventos, contas, credenciais, sessões, metadados e notas, recriando somente API/web; login, filtros/URL, retry, offline e navegação móvel/teclado SHALL continuar funcionais.

## Requirement Traceability

| ID | Critérios | Estado |
| --- | --- | --- |
| SAG-01 | 1–3, 6 | implemented |
| SAG-02 | 4–5 | pending |
| SAG-03 | 7 | pending |
