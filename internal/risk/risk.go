package risk

import (
	"fmt"
	"sort"
	"strings"
)

const (
	MaxSignals        = 32
	MaxSignalIDLength = 64
)

type Severity string

const (
	Low      Severity = "LOW"
	Medium   Severity = "MEDIUM"
	High     Severity = "HIGH"
	Critical Severity = "CRITICAL"
)

type Signal struct {
	ID           string
	Contribution int
	Description  string
}

type Input struct {
	Signals []string
}

type Result struct {
	Score       int
	Severity    Severity
	Signals     []Signal
	Explanation string
}

var definitions = map[string]Signal{
	"write_operation":        {ID: "write_operation", Contribution: 20, Description: "mutating operation"},
	"execution_operation":    {ID: "execution_operation", Contribution: 35, Description: "execution-like operation"},
	"production_environment": {ID: "production_environment", Contribution: 25, Description: "production environment"},
	"external_url":           {ID: "external_url", Contribution: 15, Description: "external URL"},
	"privileged_operation":   {ID: "privileged_operation", Contribution: 30, Description: "privileged operation"},
	"unseen_tool":            {ID: "unseen_tool", Contribution: 10, Description: "unseen tool"},
	"high_frequency":         {ID: "high_frequency", Contribution: 10, Description: "high request frequency"},
	"cross_tenant_target":    {ID: "cross_tenant_target", Contribution: 35, Description: "cross-tenant target"},
}

func Evaluate(input Input) (Result, error) {
	if len(input.Signals) > MaxSignals {
		return Result{}, fmt.Errorf("too many risk signals")
	}
	seen := make(map[string]struct{}, len(input.Signals))
	result := Result{}
	for _, id := range input.Signals {
		if len(id) == 0 || len(id) > MaxSignalIDLength {
			return Result{}, fmt.Errorf("invalid risk signal ID")
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		definition, known := definitions[id]
		if !known {
			return Result{}, fmt.Errorf("unknown risk signal %q", id)
		}
		result.Score += definition.Contribution
		result.Signals = append(result.Signals, definition)
	}
	if result.Score > 100 {
		result.Score = 100
	}
	result.Severity = severity(result.Score)
	sort.Slice(result.Signals, func(left, right int) bool { return result.Signals[left].ID < result.Signals[right].ID })
	result.Explanation = explanation(result)
	return result, nil
}

func severity(score int) Severity {
	switch {
	case score >= 80:
		return Critical
	case score >= 60:
		return High
	case score >= 30:
		return Medium
	default:
		return Low
	}
}

func explanation(result Result) string {
	if len(result.Signals) == 0 {
		return "risk score 0: no risk signals"
	}
	labels := make([]string, 0, len(result.Signals))
	for _, signal := range result.Signals {
		labels = append(labels, signal.ID)
	}
	return fmt.Sprintf("risk score %d (%s): %s", result.Score, result.Severity, strings.Join(labels, ", "))
}
