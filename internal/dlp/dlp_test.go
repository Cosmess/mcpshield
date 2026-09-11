package dlp

import (
	"strings"
	"testing"
)

func TestInspectJSONDetectsAndRedactsSecrets(t *testing.T) {
	secret := "super-secret-value"
	result, err := InspectJSON([]byte(`{"password":"super-secret-value","note":"safe"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != Redact || len(result.Matches) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if strings.Contains(string(result.Payload), secret) || !strings.Contains(string(result.Payload), RedactedValue) {
		t.Fatalf("payload = %s", result.Payload)
	}
}

func TestInspectJSONBlocksPrivateKeyAndDoesNotReturnPayload(t *testing.T) {
	result, err := InspectJSON([]byte(`{"key":"-----BEGIN RSA PRIVATE KEY-----"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != Block || len(result.Matches) != 1 || result.Payload != nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestInspectJSONRejectsOversizedPayload(t *testing.T) {
	if _, err := InspectJSON([]byte(`{"value":"` + strings.Repeat("x", MaxPayloadBytes) + `"}`)); err == nil {
		t.Fatal("oversized payload accepted")
	}
}
