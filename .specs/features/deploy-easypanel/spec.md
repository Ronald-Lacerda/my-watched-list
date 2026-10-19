# Deploy EasyPanel

Disponibilizar a aplicação atual na VPS pelo EasyPanel, com HTTPS e preservação do banco pessoal. A autorização do usuário inclui merge de todos os PRs e importação de histórico e contas com as senhas existentes.

## Critérios de aceite

- AC-1: Ao carregar `deploy/compose.easypanel.yaml`, o Compose deve resolver os mesmos serviços e imagens da aplicação sem publicar nenhuma porta no host.
- AC-2: Enquanto `COMPOSE_PROFILES` estiver vazio, a implantação deve iniciar PostgreSQL, migrations, API e web, sem executar o coletor; ao ativar `collector`, deve incluir o worker com a agenda e chave externas.
- AC-3: Quando implantado no EasyPanel, o frontend deve receber tráfego HTTPS pelo proxy na porta interna 80 e a API deve exigir a origem pública explícita. Banco, migrations, dependências de saúde e volumes persistentes devem continuar configurados.
- AC-4: Ao migrar o banco pessoal, o procedimento deve preservar os dados e credenciais existentes, sem publicar dumps ou abrir PostgreSQL para a internet.

## Plano atômico

1. Adicionar configuração Compose própria, validação no CI e procedimento em `docs/easypanel.md`; verificar resolução com e sem `collector`, ausência de portas e origem obrigatória; commit `build(deploy): prepara stack para o easypanel`.
2. Revisão independente da configuração e documentação; registrar resultado e evidências. A implantação operacional e importação são verificadas na VPS e registradas ao concluir.

## Premissas

O proxy do EasyPanel termina TLS. O PostgreSQL de destino é novo e exclusivo desta aplicação. A restauração precede a ativação do coletor. A sessão Netflix pode precisar de autenticação após mudar de IP; isso não autoriza redefinir senhas dos usuários.

## Fora do escopo

Fase 3, domínio próprio comprado e alterações em serviços existentes.
