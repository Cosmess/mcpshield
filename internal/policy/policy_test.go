package policy

import (
	"strings"
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

func TestLoadJSONValidatesAndBuildsEngine(t *testing.T) {
	engine, err := LoadJSON(strings.NewReader(`{"policies":[{"id":"developer-read","priority":10,"decision":"ALLOW","roles":["developer"],"toolPattern":"github.read_*","operation":"READ"}]}`))
	if err != nil {
		t.Fatalf("LoadJSON() error = %v", err)
	}
	result := engine.Evaluate(Input{Principal: identity.Principal{Roles: []string{"developer"}}, Tool: "github.read_issue", Operation: Read})
	if result.Decision != Allow || len(result.MatchedIDs) != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestSimulationHasNoSideEffects(t *testing.T) {
	engine, err := New([]Rule{{ID: "deny", Decision: Deny}})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Tool: "shell.execute", Operation: Execution}
	simulated := engine.Simulate(input)
	live := engine.Evaluate(input)
	if simulated.Decision != live.Decision || len(simulated.MatchedIDs) != len(live.MatchedIDs) {
		t.Fatalf("simulation = %#v, live = %#v", simulated, live)
	}
	if after := engine.Evaluate(Input{Tool: "github.read_issue", Operation: Read}); after.Decision != Deny {
		t.Fatalf("simulation mutated engine: %#v", after)
	}
}
