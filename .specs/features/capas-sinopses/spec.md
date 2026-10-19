# Capas e sinopses completas

## Problem Statement

Ronald quer fotos e sinopses para lembrar de todos os itens, incluindo Naruto Shippuden: A Morte de Naruto!, que já existe no TMDB como título alternativo. A associação atual ignora aliases e pequenas diferenças de pontuação; nomes ambíguos precisam de identificação confirmada.

## Out of Scope

Inventar sinopses, associar pela popularidade ou sem comprovação, scraping de imagens no Google, substituir a coleta por terceiros, mudar notas/identidades de obras ou apagar eventos sem título.

## Assumptions & Open Questions

| Decisão | Escolha | Motivo |
| --- | --- | --- |
| Fonte | TMDB para imagens, sinopses e metadados | Regra do projeto; Naruto possui alias oficial e imagem |
| Automático | Nome/original/alias oficial único após normalizar caixa, espaços, pontuação e acentos | Corrigir equivalências sem adivinhar obras |
| Truncados | Remover reticências para busca assistida; associação automática continua exigindo nome completo equivalente | Evitar confundir obras que compartilham prefixo |
| Ambiguidade | Busca no app com capa, ano, tipo e sinopse; escolha persistida por identidade do streaming | Confirmar a versão e reutilizar nos próximos episódios |
| Alcance | Todos os registros da mesma chave de obra/serviço, entre pessoas | Manter agrupamento e nota por série, sem misturar serviços |
| Texto | Sinopse oficial no card e completa no diálogo; fallback inglês se faltar português | Contexto automático sem texto inventado |
| Ausência real | Exibir pendência e ação de identificação; investigar registro sem nome pelo ID público | Não prometer foto autêntica de obra desconhecida |
| Entrega | PR separado baseado em codex/agrupar-series, atualização local com backup e migration aditiva | Preservar PRs anteriores e dados pessoais |

**Open questions:** none. Se uma identidade não puder ser confirmada por fontes públicas, manter pendente e indicar a limitação ao usuário.

## User Stories

**Acceptance Criteria**:

1. WHEN um nome da Netflix equivaler ao nome, original ou alias oficial único do TMDB após normalização de caixa/pontuação/acentos, o coletor SHALL obter os metadados; nomes aproximados, truncados ou múltiplas correspondências SHALL permanecer sem associação automática.
2. WHEN obter detalhes, o cliente SHALL retornar capa, sinopse, ano e gêneros oficiais; se faltar sinopse portuguesa SHALL tentar inglês, sem inventar descrição; erros, payload inválido e limite de candidatos SHALL falhar com segurança sem expor token.
3. WHEN buscar identificação no app, a API SHALL aceitar apenas filme/série e consulta limitada, retornar candidatos com imagem/ano/sinopse e exigir autenticação; salvar SHALL exigir Origin autorizado e JSON validado, rejeitando identidades inválidas ou tipo incompatível com série conhecida.
4. WHEN confirmar uma obra, a transação SHALL persistir a associação por chave de origem/serviço e atualizar todos os registros dela entre pessoas, sem mudar origem, datas, usuários ou notas; itens com mesmo nome e outra chave/serviço SHALL permanecer intactos.
5. WHEN nova coleta ou enriquecimento processar a obra confirmada, o worker SHALL respeitar a associação persistida, inclusive diante de resultado automático concorrente, e não trocar sua identificação ou apagar notas.
6. WHEN exibir um item, o app SHALL mostrar a sinopse disponível diretamente no card e oferecer completar/corrigir informações; a busca e seleção SHALL funcionar no celular e por teclado, com Escape/retorno de foco e erros preservando a consulta, sem afetar o formulário de notas.
7. WHEN reprocessar metadados, a CLI SHALL oferecer comando sem login/coleta Netflix, com opção explícita de repetir não encontrados/ambíguos e limite por lote; eventos e sessões SHALL permanecer preservados.
8. WHEN implantar localmente, a migration SHALL ser aditiva, o token SHALL ficar somente no ambiente de API/worker, e contas/senhas/sessões/eventos/notas/metadados anteriores SHALL ser preservados com backup validado; testes SHALL usar somente banco sintético.
9. WHEN validar o catálogo pessoal, Naruto Shippuden: A Morte de Naruto! SHALL obter a capa e sinopse do TMDB 20982, e outras pendências confirmáveis SHALL ser preenchidas com evidência; identidades ainda não verificáveis SHALL ficar explicitamente pendentes, com ação de identificação disponível.

## Requirement Traceability

| ID | Critérios | Estado |
| --- | --- | --- |
| CMS-01 | 1–2 | pending |
| CMS-02 | 4–5, 7 | pending |
| CMS-03 | 3, 6 | pending |
| CMS-04 | 8–9 | pending |
