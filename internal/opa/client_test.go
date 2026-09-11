package opa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/policy"
)

func TestClientEvaluatesSanitizedInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		arguments := body["arguments"].(map[string]any)
		if arguments["password"] != "[REDACTED]" {
			t.Errorf("arguments = %#v", arguments)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"decision":"ALLOW","reason":"ok","policy_ids":["bundle-1"],"policy_version":"v1"}`))
	}))
	defer server.Close()
	client := NewClient(server.URL, time.Second)
	result, err := client.Evaluate(context.Background(), policy.Input{Principal: identity.Principal{Subject: "user"}, UpstreamID: "mock", Method: "tools/call", Tool: "safe", Arguments: map[string]any{"password": "secret"}})
	if err != nil || result.Decision != policy.Allow {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

func TestBuildInputBlocksPrivateKey(t *testing.T) {
	_, err := BuildInput(policy.Input{Arguments: map[string]any{"key": "-----BEGIN RSA PRIVATE KEY-----"}})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("error = %v", err)
	}
}
