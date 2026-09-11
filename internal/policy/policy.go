package policy

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Cosmess/mcpshield/internal/identity"
)

type Decision string

const (
	Allow              Decision = "ALLOW"
	Deny               Decision = "DENY"
	RequireApproval    Decision = "REQUIRE_APPROVAL"
	AllowWithRedaction Decision = "ALLOW_WITH_REDACTION"
	AllowWithLimits    Decision = "ALLOW_WITH_LIMITS"
)

type OperationClass string

const (
	Discovery OperationClass = "DISCOVERY"
	Read      OperationClass = "READ"
	Write     OperationClass = "WRITE"
	Execution OperationClass = "EXECUTION"
	Admin     OperationClass = "ADMIN"
	Unknown   OperationClass = "UNKNOWN"
)

type Input struct {
	Principal  identity.Principal
	UpstreamID string
	Method     string
	Tool       string
	Operation  OperationClass
	Arguments  map[string]any
}

type Rule struct {
	ID             string
	Priority       int
	Decision       Decision
	Roles          []string
	TenantID       string
	UpstreamID     string
	Method         string
	ToolPattern    string
	Operation      OperationClass
	ArgumentEquals map[string]string
}

type Result struct {
	Decision   Decision
	MatchedIDs []string
	Reason     string
}

type Engine struct{ rules []Rule }

func New(rules []Rule) (*Engine, error) {
	seen := make(map[string]struct{}, len(rules))
	compiled := append([]Rule(nil), rules...)
	for index := range compiled {
		rule := &compiled[index]
		if rule.ID == "" {
			return nil, fmt.Errorf("policy ID must not be empty")
		}
		if _, exists := seen[rule.ID]; exists {
			return nil, fmt.Errorf("policy %q is duplicated", rule.ID)
		}
		seen[rule.ID] = struct{}{}
		if !validDecision(rule.Decision) {
			return nil, fmt.Errorf("policy %q has invalid decision", rule.ID)
		}
		if rule.ToolPattern != "" {
			if _, err := path.Match(rule.ToolPattern, "probe"); err != nil {
				return nil, fmt.Errorf("policy %q has invalid tool pattern", rule.ID)
			}
		}
	}
	sort.SliceStable(compiled, func(left, right int) bool { return compiled[left].Priority > compiled[right].Priority })
	return &Engine{rules: compiled}, nil
}

func (engine *Engine) Evaluate(input Input) Result {
	matched := make([]Rule, 0)
	for _, rule := range engine.rules {
		if matches(rule, input) {
			matched = append(matched, rule)
		}
	}
	if len(matched) == 0 {
		return Result{Decision: Deny, Reason: "no matching policy"}
	}
	highest := matched[0].Priority
	decision := matched[0].Decision
	ids := make([]string, 0, len(matched))
	for _, rule := range matched {
		if rule.Priority != highest {
			break
		}
		ids = append(ids, rule.ID)
		if restrictiveRank(rule.Decision) > restrictiveRank(decision) {
			decision = rule.Decision
		}
	}
	return Result{Decision: decision, MatchedIDs: ids, Reason: fmt.Sprintf("matched priority %d", highest)}
}

func matches(rule Rule, input Input) bool {
	if rule.TenantID != "" && rule.TenantID != input.Principal.TenantID {
		return false
	}
	if rule.UpstreamID != "" && rule.UpstreamID != input.UpstreamID {
		return false
	}
	if rule.Method != "" && rule.Method != input.Method {
		return false
	}
	if rule.Operation != "" && rule.Operation != input.Operation {
		return false
	}
	if rule.ToolPattern != "" {
		ok, _ := path.Match(rule.ToolPattern, input.Tool)
		if !ok {
			return false
		}
	}
	if len(rule.Roles) > 0 && !containsAny(rule.Roles, input.Principal.Roles) {
		return false
	}
	for key, expected := range rule.ArgumentEquals {
		if value, ok := input.Arguments[key].(string); !ok || value != expected {
			return false
		}
	}
	return true
}

func containsAny(expected, actual []string) bool {
	for _, want := range expected {
		for _, value := range actual {
			if want == value {
				return true
			}
		}
	}
	return false
}
func validDecision(value Decision) bool {
	switch value {
	case Allow, Deny, RequireApproval, AllowWithRedaction, AllowWithLimits:
		return true
	}
	return false
}
func restrictiveRank(value Decision) int {
	switch value {
	case Deny:
		return 5
	case RequireApproval:
		return 4
	case AllowWithLimits:
		return 3
	case AllowWithRedaction:
		return 2
	case Allow:
		return 1
	}
	return 0
}
func Classify(method, tool string) OperationClass {
	if method == "tools/list" || method == "server/discover" {
		return Discovery
	}
	lower := strings.ToLower(tool)
	switch {
	case strings.Contains(lower, "delete"), strings.Contains(lower, "admin"):
		return Admin
	case strings.Contains(lower, "execute"), strings.Contains(lower, "shell"):
		return Execution
	case strings.Contains(lower, "create"), strings.Contains(lower, "update"), strings.Contains(lower, "write"), strings.Contains(lower, "merge"):
		return Write
	case lower != "":
		return Read
	default:
		return Unknown
	}
}
