package risk

import "testing"

func TestEvaluateIsDeterministicAndDeduplicates(t *testing.T) {
	first, err := Evaluate(Input{Signals: []string{"external_url", "write_operation", "write_operation"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Evaluate(Input{Signals: []string{"external_url", "write_operation", "write_operation"}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Score != 35 || first.Severity != Medium || first.Explanation != second.Explanation || len(first.Signals) != 2 {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
}

func TestEvaluateCapsScoreAndMapsBoundaries(t *testing.T) {
	result, err := Evaluate(Input{Signals: []string{"write_operation", "execution_operation", "production_environment", "external_url", "privileged_operation", "unseen_tool", "high_frequency", "cross_tenant_target"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Score != 100 || result.Severity != Critical {
		t.Fatalf("result = %#v", result)
	}
	for score, expected := range map[int]Severity{29: Low, 30: Medium, 59: Medium, 60: High, 79: High, 80: Critical, 100: Critical} {
		if got := severity(score); got != expected {
			t.Errorf("severity(%d) = %s, want %s", score, got, expected)
		}
	}
}

func TestEvaluateRejectsUnknownAndOversizedInput(t *testing.T) {
	if _, err := Evaluate(Input{Signals: []string{"not-a-known-signal"}}); err == nil {
		t.Fatal("unknown signal accepted")
	}
	signals := make([]string, MaxSignals+1)
	for index := range signals {
		signals[index] = "write_operation"
	}
	if _, err := Evaluate(Input{Signals: signals}); err == nil {
		t.Fatal("oversized signal input accepted")
	}
}
