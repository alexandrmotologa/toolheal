package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/alexandrmotologa/toolheal/internal/config"
)

func TestCircuitBreakerTripping(t *testing.T) {
	cfg := &config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
		ResetTimeoutMs:         100,
		BackoffBaseMs:          50,
		BackoffMaxMs:           500,
	}

	cb := NewCircuitBreaker(cfg)

	// Failure 1
	_, err := cb.Execute(func() (any, error) {
		return nil, errors.New("upstream 500 error")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after 1 failure, got %s", cb.State())
	}

	// Failure 2 -> should trip to OPEN
	_, err = cb.Execute(func() (any, error) {
		return nil, errors.New("upstream 500 error")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN after 2 failures, got %s", cb.State())
	}

	// While OPEN, should reject immediately with ErrCircuitOpen
	_, err = cb.Execute(func() (any, error) {
		return "should not run", nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestDispatcherMockExecution(t *testing.T) {
	disp := NewDispatcher(nil)
	tool := &config.ToolDefinition{
		Name: "create_user",
		Target: config.ExecutionTarget{
			Type: "mock",
		},
	}

	args := map[string]any{
		"userId":    "usr_42",
		"userEmail": "test@domain.com",
		"role":      "ADMIN",
		"isActive":  true,
	}

	res := disp.Dispatch(context.Background(), tool, args)
	if res.Status != "executed_success" {
		t.Errorf("expected status executed_success, got %s", res.Status)
	}

	outMap, ok := res.Output.(map[string]any)
	if !ok || outMap["status"] != "created" {
		t.Errorf("expected mock output with status=created, got %+v", res.Output)
	}
}
