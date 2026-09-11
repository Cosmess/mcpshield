# MCPShield

Gateway de seguranca e firewall de politicas para o Model Context Protocol (MCP).

O MCPShield fica entre clientes de IA e servidores MCP. Ele autentica o chamador,
resolve apenas upstreams confiaveis, avalia policy e risco, inspeciona dados sensiveis,
exige aprovacao humana para operacoes criticas e registra as decisoes de seguranca.

O MCPShield nao e um servidor MCP de negocio e nao substitui os servidores MCP upstream.
Ele e o ponto de enforcement e governanca entre o agente e as ferramentas que o agente
pode utilizar. O agente de IA nunca e a autoridade final para permitir uma operacao.

## Para que serve

Sem um gateway, um agente pode chamar diretamente ferramentas de filesystem, shell,
bancos, GitHub, cloud ou APIs internas. Isso dificulta responder:

- quem iniciou a operacao;
- qual tenant e qual agente estavam envolvidos;
- qual policy permitiu ou negou a chamada;
- se havia risco elevado ou segredo no payload;
- se uma operacao precisava de aprovacao humana;
- como investigar e auditar o resultado depois.

O MCPShield centraliza essas decisoes sem colocar a autorizacao nas maos do modelo.

## Como funciona

```text
AI Client / Agent
	|
	| MCP over Streamable HTTP
	v
+-----------------------------+
| MCPShield Gateway            |
|                             |
| AuthN -> Principal           |
|        -> DLP               |
|        -> Risk              |
|        -> Policy            |
|        -> Approval          |
|        -> Audit             |
+---------------+-------------+
		|
		| MCP only after enforcement
		v
	Trusted MCP Server
```

Fluxo de uma chamada:

1. O cliente envia `tools/call` para `/mcp/{upstreamID}`.
2. O gateway valida o bearer token e cria um `Principal` confiavel.
3. O upstream e resolvido por ID registrado; URLs arbitrarias nao sao aceitas.
4. O payload passa por limites e DLP.
5. O risk engine calcula sinais e score deterministico.
6. O policy engine decide `ALLOW`, `DENY` ou `REQUIRE_APPROVAL`.
7. Operacoes permitidas chegam ao servidor MCP upstream.
8. Respostas sao inspecionadas antes de voltar ao cliente.
9. O resultado e auditado sem armazenar tokens ou secrets crus.

## Onde entra o PostgreSQL

MCP nao e um banco de dados. MCP e o protocolo de comunicacao entre o cliente e o
servidor de ferramentas. O MCPShield usa PostgreSQL como banco do proprio gateway,
separado de qualquer banco de negocio usado por um servidor MCP upstream.

```text
MCPShield PostgreSQL                 Upstream MCP PostgreSQL
---------------------                -----------------------
policies                             business data
approval records                     customer records
audit events                         domain state
upstream registry                    tool-owned persistence
tenant and identity mapping
outbox events
```

O PostgreSQL do MCPShield sera a fonte autoritativa para estado de seguranca e governanca:

- policies e versoes de policy;
- approvals, TTL, fingerprint e consumo unico;
- audit events e security events;
- registro de tenants, principals e upstreams;
- outbox para futuras publicacoes em Kafka.

Redis podera ser usado futuramente para cache, rate limit e estado efemero. Ele nao sera a
fonte autoritativa da auditoria ou das aprovacoes.

## O que ja esta implementado

- Bootstrap Go com `net/http`, health checks, metrics e graceful shutdown.
- Proxy remoto MCP usando o SDK oficial Go e Streamable HTTP.
- MCP `2026-07-28` como baseline.
- Registry de upstreams confiaveis e bloqueio de destinos arbitrarios.
- JWT/OIDC foundation com JWKS cacheado e principal tipado.
- Policy engine nativo com default deny, prioridades e simulacao.
- Risk engine deterministico com score de 0 a 100.
- DLP e secret detection com `BLOCK`, `REDACT` e `AUDIT`.
- Inspecao de request e response MCP sem registrar valores secretos.
- CI, testes de integracao, race detector, vet e validacao Docker Compose.

## Proximas fases

- **M6:** human approval, TTL, fingerprint, replay protection e consumo atomico.
- **M7:** adaptador OPA/Rego depois da semantica nativa estabilizada.
- **M8:** PostgreSQL autoritativo, transactional outbox e eventos Kafka.
- **M9:** assistente de seguranca com IA apenas para explicacao e analise.
- **M10:** Redis distribuido, mTLS, Kubernetes, HA, load tests e hardening.

IA e consultiva. Ela pode explicar uma negacao ou resumir eventos, mas nao pode autorizar,
aprovar, executar ou alterar policies automaticamente.

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
- [Contribuicao](CONTRIBUTING.md)

## Desenvolvimento local

Requisitos: Go 1.27.x. O slice atual nao exige PostgreSQL, credenciais cloud ou chave de
provedor de IA para executar os testes locais.

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

O PostgreSQL sera adicionado quando o repository de approvals/auditoria for implementado.
Isso evita esconder estado de seguranca em memoria quando o sistema passar a operar em HA.

Quando `MCP_SHIELD_DATABASE_URL` estiver configurado, o gateway usa PostgreSQL para o
estado de approvals e aplica a migration de approval no startup. Sem essa variavel, o
modo local continua usando o repository em memoria:

```bash
MCP_SHIELD_DATABASE_URL=postgres://mcpshield:mcpshield@localhost:5432/mcpshield \
go run ./cmd/gateway
```

O consumo de approval no PostgreSQL usa uma atualizacao condicional atomica para impedir
que dois requests concorrentes reutilizem a mesma aprovacao. O banco do MCPShield guarda
governanca e seguranca; ele continua separado do banco de negocio do servidor MCP.

## Documentacao de engenharia

- [Estado e progresso](docs/progress.md)
- [Fluxo de engenharia](docs/engineering-workflow.md)
- [Especificacoes SDD](.specs/README.md)
- [Contribuicao](CONTRIBUTING.md)

## Licenca

A licenca sera definida antes da primeira distribuicao publica do codigo.