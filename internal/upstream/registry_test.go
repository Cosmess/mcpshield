package upstream

import (
	"testing"
	"time"
)

func validDefinition() Definition {
	return Definition{ID: "mock", Endpoint: "http://127.0.0.1:9000/mcp", Enabled: true, Timeout: time.Second, ProtocolVersion: "2026-07-28"}
}

func TestRegistryResolvesOnlyEnabledDefinitions(t *testing.T) {
	definition := validDefinition()
	registry, err := NewRegistry([]Definition{definition, {ID: "disabled", Endpoint: definition.Endpoint, Enabled: false, Timeout: time.Second, ProtocolVersion: definition.ProtocolVersion}})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if resolved, ok := registry.Resolve("mock"); !ok || resolved.Endpoint != definition.Endpoint {
		t.Fatalf("Resolve(mock) = %#v, %t", resolved, ok)
	}
	if _, ok := registry.Resolve("disabled"); ok {
		t.Fatal("Resolve(disabled) = true, want false")
	}
	if _, ok := registry.Resolve("https://attacker.example"); ok {
		t.Fatal("Resolve(arbitrary URL) = true, want false")
	}
}

func TestRegistryRejectsDuplicatesAndInvalidDefinitions(t *testing.T) {
	definition := validDefinition()
	if _, err := NewRegistry([]Definition{definition, definition}); err == nil {
		t.Fatal("NewRegistry(duplicate) error = nil")
	}
	definition.Endpoint = "file:///tmp/socket"
	if _, err := NewRegistry([]Definition{definition}); err == nil {
		t.Fatal("NewRegistry(non-HTTP endpoint) error = nil")
	}
}
