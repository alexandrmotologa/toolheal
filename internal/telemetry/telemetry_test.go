package telemetry

import (
	"testing"
	"time"

	"github.com/alexandrmotologa/toolheal/internal/config"
	"github.com/alexandrmotologa/toolheal/internal/reconciler"
)

func TestStorePersistenceAndMetrics(t *testing.T) {
	cfg := &config.TelemetryConfig{
		DatabasePath:          ":memory:",
		TokensSavedPerHealing: 2500,
	}

	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}
	defer store.Close()

	rec := &InvocationRecord{
		ID:       "inv_test_001",
		ToolName: "create_user",
		Status:   "healed_success",
		OriginalArgs: map[string]any{
			"user_id": "usr_1",
		},
		HealedArgs: map[string]any{
			"userId": "usr_1",
		},
		Diffs: []reconciler.ParameterDiff{
			{
				OriginalKey: "user_id",
				HealedKey:   "userId",
				FixType:     "KEY_CASE",
			},
		},
		Healed:      true,
		Valid:       true,
		TokensSaved: 2500,
		LatencyUs:   450,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if err := store.RecordInvocation(rec); err != nil {
		t.Fatalf("failed to record invocation: %v", err)
	}

	recents, err := store.GetRecentInvocations(10)
	if err != nil {
		t.Fatalf("failed to get recent invocations: %v", err)
	}
	if len(recents) != 1 {
		t.Fatalf("expected 1 recent invocation, got %d", len(recents))
	}
	if recents[0].ToolName != "create_user" {
		t.Errorf("expected tool_name create_user, got %s", recents[0].ToolName)
	}

	metrics, err := store.GetMetrics()
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}
	if metrics.TotalInvocations != 1 || metrics.HealedInvocations != 1 {
		t.Errorf("expected 1 total & 1 healed, got %+v", metrics)
	}
	if metrics.TokensSaved != 2500 {
		t.Errorf("expected 2500 tokens saved, got %d", metrics.TokensSaved)
	}

	drifts, err := store.GetDriftHeatmap()
	if err != nil {
		t.Fatalf("failed to get drift heatmap: %v", err)
	}
	if len(drifts) != 1 || drifts[0].OriginalKey != "user_id" {
		t.Errorf("expected drift item for user_id, got %+v", drifts)
	}
}
