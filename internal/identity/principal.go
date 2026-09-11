package identity

import "context"

type Principal struct {
	Subject              string
	TenantID             string
	ClientID             string
	AgentID              string
	Roles                []string
	Scopes               []string
	Issuer               string
	AuthenticationMethod string
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func FromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}
