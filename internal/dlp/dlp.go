package dlp

import (
	"encoding/json"
	"fmt"
	"regexp"
)

const (
	MaxPayloadBytes = 1 << 20
	MaxDepth        = 16
	MaxMatches      = 32
	MaxPathLength   = 256
	RedactedValue   = "[REDACTED]"
)

type Action string

const (
	Block  Action = "BLOCK"
	Redact Action = "REDACT"
	Audit  Action = "AUDIT"
)

type Match struct {
	DetectorID string
	Path       string
	Action     Action
}

type Result struct {
	Action  Action
	Matches []Match
	Payload []byte
}

type detector struct {
	id     string
	action Action
	field  *regexp.Regexp
	value  *regexp.Regexp
}

var detectors = []detector{
	{id: "aws_access_key", action: Block, value: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{id: "jwt_like", action: Redact, value: regexp.MustCompile(`\b[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)},
	{id: "github_token", action: Block, value: regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{20,}\b`)},
	{id: "bearer_token", action: Block, field: regexp.MustCompile(`(?i)(authorization|bearer|token)`), value: regexp.MustCompile(`(?i)^Bearer\s+\S+$`)},
	{id: "private_key_header", action: Block, value: regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
	{id: "api_key_field", action: Redact, field: regexp.MustCompile(`(?i)(api[_-]?key|secret|access[_-]?token)`), value: regexp.MustCompile(`.+`)},
	{id: "password_field", action: Redact, field: regexp.MustCompile(`(?i)(password|passphrase)`), value: regexp.MustCompile(`.+`)},
}

func InspectJSON(payload []byte) (Result, error) {
	if len(payload) > MaxPayloadBytes {
		return Result{}, fmt.Errorf("payload exceeds DLP limit")
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return Result{}, fmt.Errorf("decode DLP payload")
	}
	result := Result{Action: Audit}
	transformed, err := inspect(value, "$", 0, &result)
	if err != nil {
		return Result{}, err
	}
	if result.Action == Redact {
		result.Payload, err = json.Marshal(transformed)
		if err != nil {
			return Result{}, fmt.Errorf("encode redacted payload")
		}
	}
	return result, nil
}

func inspect(value any, currentPath string, depth int, result *Result) (any, error) {
	if depth > MaxDepth {
		return nil, fmt.Errorf("payload exceeds DLP depth limit")
	}
	if len(result.Matches) > MaxMatches {
		return nil, fmt.Errorf("payload exceeds DLP match limit")
	}
	switch typed := value.(type) {
	case map[string]any:
		output := make(map[string]any, len(typed))
		for key, child := range typed {
			path := currentPath + "." + key
			if len(path) > MaxPathLength {
				return nil, fmt.Errorf("DLP field path too long")
			}
			match := matchField(key, child, path)
			if match != nil {
				if err := appendMatch(result, *match); err != nil {
					return nil, err
				}
				if match.Action == Block {
					return nil, nil
				}
				if match.Action == Redact {
					output[key] = RedactedValue
					result.Action = Redact
					continue
				}
			}
			transformed, err := inspect(child, path, depth+1, result)
			if err != nil {
				return nil, err
			}
			output[key] = transformed
		}
		return output, nil
	case []any:
		output := make([]any, len(typed))
		for index, child := range typed {
			transformed, err := inspect(child, fmt.Sprintf("%s[%d]", currentPath, index), depth+1, result)
			if err != nil {
				return nil, err
			}
			output[index] = transformed
		}
		return output, nil
	case string:
		for _, item := range detectors {
			if item.field == nil && item.value.MatchString(typed) {
				if err := appendMatch(result, Match{DetectorID: item.id, Path: currentPath, Action: item.action}); err != nil {
					return nil, err
				}
				if item.action == Block {
					return nil, nil
				}
				if item.action == Redact {
					result.Action = Redact
					return RedactedValue, nil
				}
			}
		}
	}
	return value, nil
}

func matchField(field string, value any, path string) *Match {
	text, ok := value.(string)
	if !ok {
		return nil
	}
	for _, item := range detectors {
		if item.field != nil && item.field.MatchString(field) && item.value.MatchString(text) {
			return &Match{DetectorID: item.id, Path: path, Action: item.action}
		}
	}
	return nil
}

func appendMatch(result *Result, match Match) error {
	if len(result.Matches) >= MaxMatches {
		return fmt.Errorf("payload exceeds DLP match limit")
	}
	result.Matches = append(result.Matches, match)
	if match.Action == Block {
		result.Action = Block
	}
	return nil
}
