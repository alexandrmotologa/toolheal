package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/toolheal/internal/config"
	"github.com/alexandrmotologa/toolheal/internal/executor"
	"github.com/alexandrmotologa/toolheal/internal/reconciler"
	"github.com/alexandrmotologa/toolheal/internal/telemetry"
)

func setupTestInterceptor(t *testing.T) (*Interceptor, *config.Catalog, *telemetry.Store) {
	catalog := config.NewCatalog()
	err := catalog.LoadFromFile("../../examples/tool_definitions.json")
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	engine := reconciler.NewEngine(nil)
	dispatcher := executor.NewDispatcher(nil)
	store, err := telemetry.NewStore(&config.TelemetryConfig{
		DatabasePath:          ":memory:",
		TokensSavedPerHealing: 2400,
	})
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}
	feed := telemetry.NewSSEFeed()

	interceptor := NewInterceptor(catalog, engine, dispatcher, store, feed, 2400)
	return interceptor, catalog, store
}

func TestExecuteEndpointWithHealing(t *testing.T) {
	interceptor, _, _ := setupTestInterceptor(t)

	// Hallucinated payload for process_payment:
	// - customer instead of customerId
	// - amount encoded as string "5000" instead of amountCents (integer)
	// - currency as lowercase "usd"
	// - capture as string "true"
	payload := map[string]any{
		"name": "process_payment",
		"arguments": map[string]any{
			"customer": "cus_9941",
			"amount":   "5000",
			"currency": "usd",
			"capture":  "true",
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/tools/execute", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	interceptor.HandleExecute(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ToolInvocationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !resp.Healed {
		t.Errorf("expected response to be healed")
	}
	if !resp.Valid {
		t.Errorf("expected response to be valid")
	}
	if resp.Arguments["customerId"] != "cus_9941" {
		t.Errorf("expected customerId=cus_9941, got %v", resp.Arguments["customerId"])
	}
	if resp.Arguments["amountCents"] != float64(5000) && resp.Arguments["amountCents"] != 5000 {
		t.Errorf("expected amountCents=5000, got %v", resp.Arguments["amountCents"])
	}
	if resp.Arguments["currency"] != "USD" {
		t.Errorf("expected currency=USD, got %v", resp.Arguments["currency"])
	}
}

func TestMCPJSONRPCtoolsCall(t *testing.T) {
	interceptor, _, _ := setupTestInterceptor(t)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: json.RawMessage(`{
			"name": "create_user",
			"arguments": {
				"user_id": "usr_alpha",
				"email": "agent@system.local",
				"role": "admin"
			}
		}`),
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	interceptor.HandleJSONRPC(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &rpcResp); err != nil {
		t.Fatalf("failed to parse rpc response: %v", err)
	}

	if rpcResp.Error != nil {
		t.Fatalf("expected no rpc error, got %+v", rpcResp.Error)
	}
}
