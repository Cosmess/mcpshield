package opa

import (
	"testing"

	"github.com/Cosmess/mcpshield/internal/policy"
)

func TestMapOutputStrictlyMapsDecision(t *testing.T) {
	result, err := MapOutput([]byte(`{"decision":"REQUIRE_APPROVAL","reason":"production write","policy_ids":["prod-write"],"policy_version":"v12"}`))
	if err != nil || result.Decision != policy.RequireApproval || len(result.MatchedIDs) != 1 || result.Reason == "" {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

func TestMapOutputRejectsInvalidDecisionAndOversizedReason(t *testing.T) {
	if _, err := MapOutput([]byte(`{"decision":"MAYBE"}`)); err == nil {
		t.Fatal("invalid decision accepted")
	}
	if _, err := MapOutput([]byte(`{"decision":"DENY","reason":"` + string(make([]byte, 513)) + `"}`)); err == nil {
		t.Fatal("oversized reason accepted")
	}
}
