package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthenticateValidTokenAndRejectsWrongIssuer(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"keys": []any{map[string]string{
			"kid": "key-1", "kty": "RSA", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
		}}})
	}))
	defer jwks.Close()
	validator, err := NewValidator(Config{Issuer: "https://issuer.example", Audience: "mcpshield", JWKSURL: jwks.URL, CacheTTL: time.Minute, Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "user-1", "tenant_id": "tenant-1", "iss": "https://issuer.example", "aud": "mcpshield", "exp": time.Now().Add(time.Minute).Unix()})
	token.Header["kid"] = "key-1"
	raw, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/mcp/mock", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	principal, err := validator.Authenticate(context.Background(), request)
	if err != nil || principal.Subject != "user-1" || principal.TenantID != "tenant-1" {
		t.Fatalf("Authenticate() = %#v, %v", principal, err)
	}

	wrong := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "user-1", "iss": "wrong", "aud": "mcpshield", "exp": time.Now().Add(time.Minute).Unix()})
	wrong.Header["kid"] = "key-1"
	wrongRaw, _ := wrong.SignedString(privateKey)
	request.Header.Set("Authorization", "Bearer "+wrongRaw)
	if _, err := validator.Authenticate(context.Background(), request); err == nil || strings.Contains(err.Error(), wrongRaw) {
		t.Fatalf("wrong issuer error = %v", err)
	}
}

func TestAuthenticateRequiresBearer(t *testing.T) {
	validator, err := NewValidator(Config{Issuer: "issuer", Audience: "audience", JWKSURL: "http://127.0.0.1:1", CacheTTL: time.Minute, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/mcp/mock", nil)
	if _, err := validator.Authenticate(context.Background(), request); err == nil {
		t.Fatal("Authenticate() error = nil")
	}
}
