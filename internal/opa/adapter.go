package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Cosmess/mcpshield/internal/dlp"
	"github.com/Cosmess/mcpshield/internal/policy"
)

type Client struct {
	Endpoint   string
	HTTPClient *http.Client
}

func NewClient(endpoint string, timeout time.Duration) *Client {
	return &Client{Endpoint: endpoint, HTTPClient: &http.Client{Timeout: timeout}}
}

func (client *Client) Evaluate(ctx context.Context, input policy.Input) (policy.Result, error) {
	payload, err := BuildInput(input)
	if err != nil {
		return policy.Result{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return policy.Result{}, fmt.Errorf("create OPA request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.HTTPClient.Do(request)
	if err != nil {
		return policy.Result{}, fmt.Errorf("call OPA: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return policy.Result{}, fmt.Errorf("OPA returned status %d", response.StatusCode)
	}
	resultBytes, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil {
		return policy.Result{}, fmt.Errorf("read OPA response: %w", err)
	}
	if len(resultBytes) > 64*1024 {
		return policy.Result{}, fmt.Errorf("OPA response exceeds limit")
	}
	return MapOutput(resultBytes)
}

func BuildInput(input policy.Input) ([]byte, error) {
	arguments, err := json.Marshal(input.Arguments)
	if err != nil {
		return nil, fmt.Errorf("marshal policy arguments: %w", err)
	}
	inspected, err := dlp.InspectJSON(arguments)
	if err != nil {
		return nil, fmt.Errorf("inspect policy arguments: %w", err)
	}
	if inspected.Action == dlp.Block {
		return nil, fmt.Errorf("policy input blocked by DLP")
	}
	if inspected.Action == dlp.Redact {
		arguments = inspected.Payload
	}
	var safeArguments map[string]any
	if err := json.Unmarshal(arguments, &safeArguments); err != nil {
		return nil, fmt.Errorf("decode policy arguments: %w", err)
	}
	payload := map[string]any{"principal": input.Principal, "mcp": map[string]any{"upstream_id": input.UpstreamID, "method": input.Method, "tool": input.Tool, "operation": input.Operation}, "arguments": safeArguments}
	return json.Marshal(payload)
}

type Output struct {
	Decision      policy.Decision `json:"decision"`
	Reason        string          `json:"reason"`
	PolicyIDs     []string        `json:"policy_ids"`
	PolicyVersion string          `json:"policy_version"`
}

func MapOutput(payload []byte) (policy.Result, error) {
	if len(payload) > 64*1024 {
		return policy.Result{}, fmt.Errorf("OPA response exceeds limit")
	}
	var output Output
	if err := json.Unmarshal(payload, &output); err != nil {
		return policy.Result{}, fmt.Errorf("decode OPA response")
	}
	if !validDecision(output.Decision) {
		return policy.Result{}, fmt.Errorf("OPA response has invalid decision")
	}
	if len(output.Reason) > 512 {
		return policy.Result{}, fmt.Errorf("OPA response reason exceeds limit")
	}
	for _, id := range output.PolicyIDs {
		if id == "" || len(id) > 128 {
			return policy.Result{}, fmt.Errorf("OPA response has invalid policy ID")
		}
	}
	return policy.Result{Decision: output.Decision, MatchedIDs: append([]string(nil), output.PolicyIDs...), Reason: sanitizeReason(output.Reason, output.PolicyVersion)}, nil
}

func validDecision(decision policy.Decision) bool {
	switch decision {
	case policy.Allow, policy.Deny, policy.RequireApproval, policy.AllowWithRedaction, policy.AllowWithLimits:
		return true
	}
	return false
}

func sanitizeReason(reason, version string) string {
	reason = strings.TrimSpace(reason)
	if version == "" {
		return reason
	}
	if reason == "" {
		return "policy version " + version
	}
	return reason + " (policy version " + version + ")"
}
