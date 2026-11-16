package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/alexandrmotologa/toolheal/internal/config"
)

// Dispatcher coordinates tool execution through mock, HTTP, CLI, or passthrough targets.
type Dispatcher struct {
	mu          sync.RWMutex
	breakers    map[string]*CircuitBreaker
	breakerCfg  *config.CircuitBreakerConfig
	mockServer  *MockServer
	httpClient  *http.Client
}

// ExecutionResult wraps the tool execution payload and telemetry details.
type ExecutionResult struct {
	ToolName      string `json:"tool_name"`
	Status        string `json:"status"` // "executed_success", "passthrough", "fallback"
	Output        any    `json:"output"`
	ExecutionTimeUs int64  `json:"execution_time_us"`
	FallbackUsed  bool   `json:"fallback_used"`
}

// NewDispatcher creates a new tool dispatcher instance.
func NewDispatcher(breakerCfg *config.CircuitBreakerConfig) *Dispatcher {
	return &Dispatcher{
		breakers:   make(map[string]*CircuitBreaker),
		breakerCfg: breakerCfg,
		mockServer: NewMockServer(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// getCircuitBreaker returns or creates a circuit breaker for a given tool.
func (d *Dispatcher) getCircuitBreaker(toolName string) *CircuitBreaker {
	d.mu.Lock()
	defer d.mu.Unlock()

	cb, exists := d.breakers[toolName]
	if !exists {
		cb = NewCircuitBreaker(d.breakerCfg)
		d.breakers[toolName] = cb
	}
	return cb
}

// Dispatch executes the healed tool parameters against the configured target.
func (d *Dispatcher) Dispatch(ctx context.Context, tool *config.ToolDefinition, args map[string]any) *ExecutionResult {
	start := time.Now()
	toolName := "unknown"
	if tool != nil {
		toolName = tool.Name
	}

	targetType := "mock"
	if tool != nil && tool.Target.Type != "" {
		targetType = tool.Target.Type
	}

	cb := d.getCircuitBreaker(toolName)

	var output any
	var err error

	// If target is passthrough, immediately return cleaned args
	if targetType == "passthrough" {
		return &ExecutionResult{
			ToolName:        toolName,
			Status:          "passthrough",
			Output:          args,
			ExecutionTimeUs: time.Since(start).Microseconds(),
			FallbackUsed:    false,
		}
	}

	// Execute through circuit breaker
	_, execErr := cb.Execute(func() (any, error) {
		switch targetType {
		case "mock":
			output, err = d.mockServer.Execute(toolName, args)
			return output, err

		case "http":
			output, err = d.dispatchHTTP(ctx, tool.Target, args)
			return output, err

		case "cli":
			output, err = d.dispatchCLI(ctx, tool.Target, args)
			return output, err

		default:
			output, err = d.mockServer.Execute(toolName, args)
			return output, err
		}
	})

	durationUs := time.Since(start).Microseconds()

	if execErr != nil {
		receipt := GenerateFallbackReceipt(toolName, args, execErr, "tool execution failure guarded by circuit breaker")
		return &ExecutionResult{
			ToolName:        toolName,
			Status:          "fallback",
			Output:          receipt,
			ExecutionTimeUs: durationUs,
			FallbackUsed:    true,
		}
	}

	return &ExecutionResult{
		ToolName:        toolName,
		Status:          "executed_success",
		Output:          output,
		ExecutionTimeUs: durationUs,
		FallbackUsed:    false,
	}
}

func (d *Dispatcher) dispatchHTTP(ctx context.Context, target config.ExecutionTarget, args map[string]any) (any, error) {
	if target.Endpoint == "" {
		return nil, fmt.Errorf("http endpoint missing in tool target")
	}

	jsonBytes, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request arguments: %w", err)
	}

	method := target.Method
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, target.Endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range target.Headers {
		req.Header.Set(k, v)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream returned http status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var parsed any
	if err := json.Unmarshal(bodyBytes, &parsed); err == nil {
		return parsed, nil
	}

	return string(bodyBytes), nil
}

func (d *Dispatcher) dispatchCLI(ctx context.Context, target config.ExecutionTarget, args map[string]any) (any, error) {
	if len(target.Command) == 0 {
		return nil, fmt.Errorf("cli command empty in tool target")
	}

	jsonBytes, _ := json.Marshal(args)
	cmd := exec.CommandContext(ctx, target.Command[0], target.Command[1:]...)
	cmd.Stdin = bytes.NewReader(jsonBytes)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cli command error: %w (output: %s)", err, string(output))
	}

	var parsed any
	if err := json.Unmarshal(output, &parsed); err == nil {
		return parsed, nil
	}

	return string(output), nil
}
