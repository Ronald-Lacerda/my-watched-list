# Direção e integração

Sessão usa o título mais recente como elemento dominante. A paleta vem da prévia aprovada; o verde claro funciona como destaque de ações e média, com superfícies de carvão e tipografia de cartaz no nome da obra. A composição é alinhada à esquerda, com navegação superior e filtros compactos antes do destaque. No celular, a arte passa para cima e as notas ficam abaixo do título.

```text
marca       Histórico / Para avaliar / Conexões       conta
Nosso histórico
Pessoa / Streaming / De / Até                         limpar
primeiro evento: título, pessoa, serviço | arte
notas individuais / média / avaliação
outros eventos por data / carregar mais
```

O backend já oferece tudo o que a tela precisa. Não acrescentar endpoints ou campos, não sintetizar dados nem agrupar os dois perfis. `.watch-card` e `data-event-id` continuam identificando exatamente um evento, inclusive no destaque. `.couple-score` mantém a média e os rótulos acessíveis continuam compatíveis com as ações existentes.

Poster recebe modo destacado e reserva espaço. Usa apenas o caminho TMDB entregue pela API. A ausência/falha de imagem cai para arte CSS e o título fica sempre em texto HTML. Sem imagens externas escolhidas por nome ou dados da prévia no produto.

Fontes ficam em public/fonts, com licença OFL. O build inclui WOFF2 no shell PWA. Nenhuma dependência de Google Fonts em tempo de execução, nenhuma alteração da política de cache de dados privados.

A UI usa os tokens #101317, #191e24, #f5f3ee, #b2b8bf e #8bc4c8. A hierarquia se apoia no conteúdo e em espaços; retirar os elementos editoriais claros da versão anterior. A restrição de tema escuro foi escolhida pelo usuário e supera a preferência anterior da Fase 2.
