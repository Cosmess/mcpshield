package policy

import (
	"testing"

	"github.com/Cosmess/mcpshield/internal/identity"
)

func TestEngineDefaultsToDenyAndMatchesDimensions(t *testing.T) {
	engine, err := New([]Rule{{ID: "developer-read", Priority: 10, Decision: Allow, Roles: []string{"developer"}, TenantID: "tenant-a", ToolPattern: "github.read_*", Operation: Read}, {ID: "default-deny", Priority: 0, Decision: Deny}})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Principal: identity.Principal{TenantID: "tenant-a", Roles: []string{"developer"}}, Tool: "github.read_issue", Operation: Read}
	result := engine.Evaluate(input)
	if result.Decision != Allow || len(result.MatchedIDs) != 1 || result.MatchedIDs[0] != "developer-read" {
		t.Fatalf("result = %#v", result)
	}
	input.Tool = "github.delete_repo"
	if result = engine.Evaluate(input); result.Decision != Deny {
		t.Fatalf("unmatched result = %#v", result)
	}
}

func TestEngineUsesPriorityAndDenyWinsTies(t *testing.T) {
	engine, err := New([]Rule{{ID: "allow", Priority: 5, Decision: Allow}, {ID: "deny", Priority: 5, Decision: Deny}})
	if err != nil {
		t.Fatal(err)
	}
	result := engine.Evaluate(Input{})
	if result.Decision != Deny || len(result.MatchedIDs) != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestClassifyOperationsConservatively(t *testing.T) {
	if Classify("tools/list", "") != Discovery || Classify("tools/call", "shell.execute") != Execution || Classify("tools/call", "github.create_issue") != Write || Classify("tools/call", "github.read_issue") != Read || Classify("tools/call", "") != Unknown {
		t.Fatal("unexpected operation classification")
	}
}

func TestRejectsInvalidRules(t *testing.T) {
	if _, err := New([]Rule{{ID: "bad", Decision: Decision("maybe")}}); err == nil {
		t.Fatal("invalid decision accepted")
	}
	if _, err := New([]Rule{{ID: "bad", Decision: Allow, ToolPattern: "["}}); err == nil {
		t.Fatal("invalid pattern accepted")
	}
}
