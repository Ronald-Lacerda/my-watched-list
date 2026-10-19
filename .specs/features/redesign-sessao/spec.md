# Redesign Sessão

## Problem Statement

Ronald aprovou a proposta Sessão para substituir a aparência clara do app. A interface passa a usar composição cinematográfica, destaque para o registro mais recente e notas individuais próximas da média do casal. A implementação usa os dados reais e mantém o contrato funcional da Fase 2.

## Out of Scope

Mudanças de banco, autenticação, coleta, ranking anual, novos streamings, edição do histórico, dados fictícios na aplicação, migração de produção e mesclagem dos PRs anteriores. Não obter pôsteres por scraping nem exigir token TMDB para usar o app.

## Assumptions & Open Questions

| Decisão | Escolha | Motivo |
| --- | --- | --- |
| Autorização | Implementação, testes, atualização local e PR separado | Pedido atual e fluxo de trabalho já autorizado |
| Base | codex/redesign-sessao parte de feat/fase-2-app em 29aea94 | PR #3 ainda aberto; diff do redesign separado |
| Referência | Proposta Sessão aprovada nesta conversa | Não implementar a proposta Diário |
| Tema | Fundo #101317, superfície #191e24, texto #f5f3ee, secundário #b2b8bf, destaque #8bc4c8 | Paleta da proposta, contraste e hierarquia |
| Tipografia | DM Sans para interface; Barlow Condensed para o título destacado; WOFF2 e licenças locais | Consistência com a prévia e shell offline sem CDN de fontes |
| Destaque | Primeiro evento retornado pela consulta atual, uma vez, sem reunir eventos de pessoas diferentes | Preservar data, identidade, filtros e paginação |
| Sem TMDB | Arte abstrata local com título legível quando não existe pôster ou ele falha | Ambiente pessoal ainda sem token |
| Pessoa | Controle Pessoa mantém select acessível e UUIDs existentes; aparência pode ser compacta | Preservar fluxos e contrato atuais |
| Avaliação | Diálogo existente mantém salvar/editar/remover e retorno de foco | Não criar outro sistema de notas |
| Operação | Atualizar apenas web no Compose local depois dos testes | Banco, worker, credenciais e sessões preservados |

**Open questions:** none.

## User Stories

**Acceptance Criteria**:

1. WHEN abrir o app no desktop ou celular, a interface SHALL apresentar tema escuro Sessão, navegação superior no desktop e inferior no celular, mantendo login, convite, logout, conexões e indicação de rede utilizáveis.
2. WHEN receber uma página não vazia de histórico ou pendentes, o app SHALL destacar o primeiro evento da seleção atual exatamente uma vez e exibir cada outro evento exatamente uma vez com título, pessoa, serviço e dia originais; histórico não deve reunir episódios ou perfis em um evento inventado.
3. WHEN combinar filtros por pessoa, streaming e datas ou carregar mais registros, o app SHALL manter URL, ordem, limites inclusivos e paginação existentes; novo filtro troca o destaque e carregar mais não duplica nem substitui o primeiro evento.
4. IF um evento não tiver pôster ou a imagem falhar, o app SHALL mostrar arte local com título e metadados legíveis, sem imagem quebrada; um caminho válido posterior SHALL poder carregar imagem e o destaque SHALL reservar espaço para ela.
5. WHEN avaliar o destaque ou outro evento, a pessoa SHALL ver as notas individuais, a média disponível e o contador corretos, incluindo nota zero; salvar, editar ou remover SHALL atualizar todos os eventos da obra sem alterar a nota da outra pessoa.
6. WHEN usar pendentes ou encontrar seleção vazia, falha, sessão expirada ou ausência de rede, o app SHALL manter as respostas e ações da Fase 2, sem um destaque obsoleto e sem apagar dados digitados após falha ao salvar.
7. WHEN usar a interface a 320, 390 e 1280 pixels ou por teclado, o app SHALL manter conteúdo sem rolagem horizontal, foco visível, alvos principais de pelo menos 44 pixels, Escape e retorno de foco do diálogo, texto longo legível e movimento reduzido respeitado.
8. WHEN construir ou abrir a PWA offline, o app SHALL servir fontes locais e shell público no novo tema, com manifest e theme-color coerentes; cache SHALL continuar excluindo API, notas, histórico, sessões e pôsteres privados.
9. WHEN atualizar o ambiente local, a implantação SHALL preservar usuários, senhas, sessões, eventos, notas e containers de outros projetos, sem executar migration nova ou coleta.

## Implicit requirements sweep

| Dimensão | Cobertura |
| --- | --- |
| Estado/identidade | AC2–3, AC5: evento e obra mantêm identidades diferentes |
| Falhas externas | AC4, AC6: pôster quebrado, API e rede |
| Autenticação | AC1, AC6: contrato e recuperação existentes |
| Privacidade/cache | AC8–9: fontes públicas locais, dados privados fora do cache |
| Concorrência | AC3: filtros continuam abortando requisições anteriores |
| Acessibilidade | AC7: foco, responsividade, textos e controles |

## Requirement Traceability

| ID | Critérios | Estado |
| --- | --- | --- |
| RDS-01 | 1, 7 | pending |
| RDS-02 | 2–6 | pending |
| RDS-03 | 8–9 | pending |
