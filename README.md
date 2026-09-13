# MCPShield

Gateway de seguranca e governanca para o Model Context Protocol (MCP), escrito em Go.

O MCPShield fica entre um cliente de IA e servidores MCP que oferecem ferramentas como
GitHub, filesystem, bancos de dados, cloud ou APIs internas. Antes de uma chamada chegar
ao upstream, o gateway autentica o principal, aplica policy, calcula risco, inspeciona
segredos, pode exigir aprovacao humana e registra uma trilha de auditoria.

O projeto resolve um problema simples e importante: um modelo pode decidir **qual
ferramenta quer chamar**, mas nunca deve decidir sozinho **se tem permissao para chama-la**.
Essa autoridade pertence ao gateway e as suas regras deterministicas.

## Problema e objetivo

Sem uma camada de governanca, chamadas de agentes tendem a misturar identidade,
autorizacao, transporte e dados sensiveis no mesmo fluxo. Isso dificulta responder:

- quem iniciou a operacao;
- qual tenant, cliente e agente estavam envolvidos;
- qual policy permitiu ou negou a chamada;
- qual risco foi detectado;
- se havia secret ou payload sensivel;
- se uma operacao precisava de aprovacao humana;
- como investigar o evento depois.

O MCPShield separa essas responsabilidades e cria um ponto de enforcement observavel,
testavel e independente do modelo de IA.

## Arquitetura

```mermaid
flowchart LR
	client[AI Client / Agent]
	gateway[MCPShield Gateway]
	upstream[Trusted MCP Server]

	client -->|MCP over Streamable HTTP| gateway
	gateway -->|MCP after enforcement| upstream

	subgraph security[Security pipeline]
		auth[AuthN and Principal]
		dlp[DLP and Secret Detection]
		risk[Deterministic Risk]
		policy[Native Policy]
		approval[Human Approval]
		audit[Audit and Metrics]

		auth --> dlp --> risk --> policy --> approval --> audit
	end

	gateway --> security
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

O servidor MCP upstream continua sendo dono das ferramentas e dos dados de negocio. O
MCPShield nao substitui esse servidor e nao acessa diretamente seu banco de negocio para
autorizar chamadas.

## Tecnologias

| Area | Tecnologia |
| --- | --- |
| Linguagem | Go 1.27 |
| HTTP | `net/http` |
| MCP | Official Go MCP SDK, Streamable HTTP, MCP `2026-07-28` |
| Autenticacao | JWT/OIDC foundation, JWKS cacheado |
| Policy | Native Go policy engine, default deny, OPA/Rego adapter foundation |
| Risk | Deterministic score de 0 a 100 |
| DLP | Secret detection, `BLOCK`, `REDACT`, `AUDIT` |
| Persistencia | PostgreSQL opcional para approvals |
| Testes | `testing`, `httptest`, Testcontainers, race detector |
| Observabilidade | `log/slog`, health endpoints, Prometheus-style metrics |
| Execucao local | Docker Compose |

## Componentes principais

- `cmd/gateway`: montagem do processo, configuracao e graceful shutdown.
- `internal/auth`: validacao JWT e criacao do principal.
- `internal/upstream`: registry de destinos MCP confiaveis.
- `internal/mcpproxy`: proxy Streamable HTTP e enforcement antes do upstream.
- `internal/policy`: policy nativa, default deny e decisoes deterministicas.
- `internal/risk`: sinais e score de risco explicavel.
- `internal/dlp`: deteccao e redaction de secrets em payloads JSON.
- `internal/approval`: workflow de aprovacao, fingerprint, TTL e consumo unico.
- `internal/opa`: fundacao do adapter OPA/Rego com input sanitizado.
- `internal/audit`: eventos metadata-only e sink de auditoria.

## PostgreSQL e MCP

MCP e um protocolo de comunicacao, nao um banco de dados. O PostgreSQL pertence ao
MCPShield e armazena estado de governanca do gateway:

```text
MCPShield PostgreSQL             Banco do servidor MCP upstream
---------------------            -----------------------------
approvals                         business data
policies                           customer records
audit/security events              domain state
upstream registry                  tool-owned persistence
outbox events
```

O banco do upstream continua separado. Redis, quando introduzido, servira para cache,
rate limit e estado efemero; nao sera a fonte autoritativa de approvals ou auditoria.

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

## Estado atual

O projeto possui um gateway MCP funcional e uma base de seguranca evolutiva. A implementacao
foi conduzida em fatias verticais usando desenvolvimento orientado a especificacao (SDD),
com verificacao independente antes de considerar cada milestone concluido.

Consulte [docs/progress.md](docs/progress.md) para o estado atual e
[docs/portfolio-final-status.md](docs/portfolio-final-status.md) para o resumo tecnico.

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

## Como usar o MCPShield em um projeto

O MCPShield nao precisa ser incorporado ao codigo do seu servidor MCP. O uso normal e:

```text
Seu agente ou aplicacao MCP
					|
					| aponta para o MCPShield
					v
MCPShield /mcp/{upstreamID}
					|
					| encaminha somente depois do enforcement
					v
Servidor MCP do seu projeto
```

### 1. Registre o upstream

Em desenvolvimento local, configure o servidor MCP confiavel por ambiente:

```bash
MCP_SHIELD_UPSTREAM_ID=orders \
MCP_SHIELD_UPSTREAM_ENDPOINT=http://127.0.0.1:9000/mcp \
go run ./cmd/gateway
```

O endpoint publico do gateway passa a ser:

```text
http://127.0.0.1:8080/mcp/orders
```

O cliente nunca deve enviar uma URL de destino arbitraria. Ele envia apenas o ID do
upstream publicado pelo gateway.

### 2. Aponte o cliente MCP para o gateway

Clientes que aceitam servidores MCP remotos normalmente usam uma configuracao equivalente
a esta. O nome exato da chave pode variar entre clientes:

```json
{
	"mcpServers": {
		"orders-secure": {
			"url": "http://127.0.0.1:8080/mcp/orders",
			"headers": {
				"Authorization": "Bearer <development-token>"
			}
		}
	}
}
```

Em producao, use um token JWT emitido pelo seu IdP e configure `MCP_SHIELD_OIDC_ISSUER`,
`MCP_SHIELD_OIDC_AUDIENCE` e `MCP_SHIELD_OIDC_JWKS_URL`. Nao coloque tokens reais em
arquivos versionados.

### 3. Use pelo SDK Go

Uma aplicacao Go pode apontar o SDK diretamente para o endpoint protegido:

```go
type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (transport bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+transport.token)
	return transport.base.RoundTrip(clone)
}

client := mcp.NewClient(
		&mcp.Implementation{Name: "orders-agent", Version: "1.0.0"},
		nil,
)

session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: "http://127.0.0.1:8080/mcp/orders",
		HTTPClient: &http.Client{
			Transport: bearerTransport{token: tokenFromSecretManager, base: http.DefaultTransport},
				Timeout: 10 * time.Second,
		},
}, nil)
if err != nil {
		return err
}
defer session.Close()
```

O exemplo acima mostra a fronteira de integracao; a implementacao real do `RoundTripper`
deve obter o token de um secret manager ou provider OAuth, nunca de um valor hardcoded.

### 4. Carregue uma policy

Para o modo nativo, use `MCP_SHIELD_POLICY_FILE`:

```bash
MCP_SHIELD_POLICY_FILE=policies/example.json \
MCP_SHIELD_UPSTREAM_ID=orders \
MCP_SHIELD_UPSTREAM_ENDPOINT=http://127.0.0.1:9000/mcp \
go run ./cmd/gateway
```

Sem uma regra correspondente, o policy engine usa `DENY`. Uma chamada bloqueada nao chega
ao servidor MCP upstream.

Para carregar politicas nativas no startup, defina `MCP_SHIELD_POLICY_FILE` apontando para
um documento JSON como [policies/example.json](policies/example.json). A politica padrao e
deny quando nenhuma regra corresponder.

O PostgreSQL e usado pelo repository de approvals quando configurado. O modo local sem
`MCP_SHIELD_DATABASE_URL` continua usando memoria para facilitar desenvolvimento.

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