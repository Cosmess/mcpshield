package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/jwks"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Issuer   string
	Audience string
	JWKSURL  string
	CacheTTL time.Duration
	Timeout  time.Duration
}

type Validator struct {
	config Config
	keys   *jwks.Cache
}

func NewValidator(config Config) (*Validator, error) {
	if config.Issuer == "" || config.Audience == "" || config.JWKSURL == "" {
		return nil, fmt.Errorf("issuer, audience, and JWKS URL are required")
	}
	if config.CacheTTL <= 0 || config.Timeout <= 0 {
		return nil, fmt.Errorf("JWKS cache TTL and timeout must be positive")
	}
	return &Validator{config: config, keys: jwks.New(config.JWKSURL, config.CacheTTL, config.Timeout)}, nil
}

func (validator *Validator) Authenticate(ctx context.Context, request *http.Request) (identity.Principal, error) {
	header := request.Header.Get("Authorization")
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return identity.Principal{}, fmt.Errorf("authentication required")
	}
	token, err := jwt.Parse(parts[1], func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("unsupported signing algorithm")
		}
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("missing key ID")
		}
		return validator.keys.Key(ctx, kid)
	}, jwt.WithIssuer(validator.config.Issuer), jwt.WithAudience(validator.config.Audience), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"RS256"}))
	if err != nil || !token.Valid {
		return identity.Principal{}, fmt.Errorf("authentication failed")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return identity.Principal{}, fmt.Errorf("authentication failed")
	}
	return identity.Principal{Subject: stringClaim(claims, "sub"), TenantID: stringClaim(claims, "tenant_id"), ClientID: stringClaim(claims, "client_id"), AgentID: stringClaim(claims, "agent_id"), Roles: stringList(claims["roles"]), Scopes: stringList(claims["scope"]), Issuer: stringClaim(claims, "iss"), AuthenticationMethod: "jwt"}, nil
}

func stringClaim(claims jwt.MapClaims, name string) string {
	value, _ := claims[name].(string)
	return value
}
func stringList(value any) []string {
	switch values := value.(type) {
	case []any:
		result := make([]string, 0, len(values))
		for _, item := range values {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	case string:
		return strings.Fields(values)
	}
	return nil
}
