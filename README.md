# MCPShield

Gateway de seguranca e firewall de politicas para o Model Context Protocol (MCP).

O MCPShield ficara entre clientes de IA e servidores MCP, aplicando autenticacao,
autorizacao, avaliacao de risco, limites, protecao de dados, aprovacao humana e
auditoria. O agente de IA nunca sera a autoridade final para permitir uma operacao.

## Estado atual

O bootstrap e a primeira fatia MCP ja foram concluidos. A implementacao continua em
fatias verticais usando desenvolvimento orientado a especificacao (SDD), com verificacao
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

## Desenvolvimento local

Requisitos: Go 1.27.x. O bootstrap atual nao exige banco de dados, credenciais cloud ou
chave de provedor de IA.

```bash
go run ./cmd/gateway
go test ./...
go test -race ./...
go vet ./...
docker compose config
```

Com o gateway em execucao, consulte `http://127.0.0.1:8080/health/live`,
`http://127.0.0.1:8080/health/ready` e `http://127.0.0.1:8080/metrics`.

Para testar um upstream MCP local, configure o endpoint confiavel e o ID publicado pelo
gateway:

```bash
MCP_SHIELD_UPSTREAM_ID=mock \
MCP_SHIELD_UPSTREAM_ENDPOINT=http://127.0.0.1:9000/mcp \
go run ./cmd/gateway
```

O proxy fica disponivel em `/mcp/mock`. O M1 usa o SDK oficial Go, Streamable HTTP e MCP
`2026-07-28`. Autenticacao, politicas, credenciais de upstream e protecao completa contra
SSRF ainda pertencem as proximas fases.

Para carregar politicas nativas no startup, defina `MCP_SHIELD_POLICY_FILE` apontando para
um documento JSON como [policies/example.json](policies/example.json). A politica padrao e
deny quando nenhuma regra corresponder.

## Licenca

A licenca sera definida antes da primeira distribuicao publica do codigo.