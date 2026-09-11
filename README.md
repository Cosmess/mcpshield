# MCPShield

Gateway de seguranca e firewall de politicas para o Model Context Protocol (MCP).

O MCPShield ficara entre clientes de IA e servidores MCP, aplicando autenticacao,
autorizacao, avaliacao de risco, limites, protecao de dados, aprovacao humana e
auditoria. O agente de IA nunca sera a autoridade final para permitir uma operacao.

## Estado atual

O projeto esta no bootstrap documental. A implementacao sera conduzida em fatias
verticais usando desenvolvimento orientado a especificacao (SDD), com verificacao
independente antes de considerar uma tarefa concluida.

Consulte [docs/progress.md](docs/progress.md) para o estado atual, o que foi
concluido e os proximos blocos de trabalho.

## Como o projeto sera desenvolvido

Cada mudanca relevante segue este ciclo:

1. especificar requisitos e criterios observaveis;
2. desenhar a solucao quando houver decisoes arquiteturais;
3. dividir o trabalho em tarefas atomicas;
4. implementar uma tarefa por vez;
5. executar gates automatizados e verificacao independente;
6. abrir Pull Request, revisar, fazer merge e atualizar o progresso.

O fluxo detalhado esta em [docs/engineering-workflow.md](docs/engineering-workflow.md).

## Documentacao

- [Estado e progresso](docs/progress.md)
- [Fluxo de engenharia](docs/engineering-workflow.md)
- [Especificacoes SDD](.specs/README.md)
- [Decisoes arquiteturais](docs/adr/)
- [Contribuicao](CONTRIBUTING.md)

## Licenca

A licenca sera definida antes da primeira distribuicao publica do codigo.