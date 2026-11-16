package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/toolheal/internal/config"
	"github.com/alexandrmotologa/toolheal/internal/executor"
	"github.com/alexandrmotologa/toolheal/internal/reconciler"
	"github.com/alexandrmotologa/toolheal/internal/telemetry"
)

// ToolInvocationRequest represents the flexible payload formats accepted by toolheal.
type ToolInvocationRequest struct {
	Tool      string          `json:"tool,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`

	// OpenAI / Anthropic function call fields
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function *struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function,omitempty"`
}

// ToolInvocationResponse is returned to the caller agent runtime.
type ToolInvocationResponse struct {
	ID        string          `json:"id"`
	Tool      string          `json:"tool"`
	Status    string          `json:"status"` // "healed_success", "passthrough_success", "fallback", "invalid"
	Healed    bool            `json:"healed"`
	Valid     bool            `json:"valid"`
	Arguments map[string]any  `json:"arguments"`
	Diffs     []reconciler.ParameterDiff `json:"diffs,omitempty"`
	Result    any             `json:"result,omitempty"`
	LatencyUs int64           `json:"latency_us"`
}

// Interceptor orchestrates schema reconciliation, execution, and telemetry.
type Interceptor struct {
	catalog    *config.Catalog
	engine     *reconciler.Engine
	dispatcher *executor.Dispatcher
	store      *telemetry.Store
	feed       *telemetry.SSEFeed
	tokensCost int
}

// NewInterceptor creates a configured middleware interceptor.
func NewInterceptor(
	catalog *config.Catalog,
	engine *reconciler.Engine,
	dispatcher *executor.Dispatcher,
	store *telemetry.Store,
	feed *telemetry.SSEFeed,
	tokensCost int,
) *Interceptor {
	if tokensCost <= 0 {
		tokensCost = 2400
	}
	return &Interceptor{
		catalog:    catalog,
		engine:     engine,
		dispatcher: dispatcher,
		store:      store,
		feed:       feed,
		tokensCost: tokensCost,
	}
}

// Engine returns the underlying reconciliation engine.
func (h *Interceptor) Engine() *reconciler.Engine {
	return h.engine
}

// Catalog returns the tool schema catalog.
func (h *Interceptor) Catalog() *config.Catalog {
	return h.catalog
}

// Store returns the telemetry SQLite store.
func (h *Interceptor) Store() *telemetry.Store {
	return h.store
}

// HandleExecute handles POST /v1/tools/execute, reconciling and executing the tool.
func (h *Interceptor) HandleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolName, rawArgs, callID, err := h.parseInvocationPayload(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	response := h.processInvocation(r.Context(), toolName, rawArgs, callID, true)

	w.Header().Set("Content-Type", "application/json")
	if !response.Valid {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(response)
}

// HandleHeal handles POST /v1/heal, reconciling arguments without executing the target tool.
func (h *Interceptor) HandleHeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolName, rawArgs, callID, err := h.parseInvocationPayload(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	response := h.processInvocation(r.Context(), toolName, rawArgs, callID, false)

	w.Header().Set("Content-Type", "application/json")
	if !response.Valid {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(response)
}

// HandleJSONRPC processes incoming Model Context Protocol (MCP) JSON-RPC 2.0 tool requests.
func (h *Interceptor) HandleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NewJSONRPCError(CodeParseError, "Parse error", err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case "tools/list":
		// Return list of all tools in catalog
		tools := h.catalog.List()
		mcpTools := make([]map[string]any, len(tools))
		for i, t := range tools {
			mcpTools[i] = map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.Parameters,
			}
		}
		_ = json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": mcpTools,
			},
		})
		return

	case "tools/call":
		toolName, args, err := ParseMCPToolCall(req.Params)
		if err != nil {
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    CodeInvalidParams,
					Message: err.Error(),
				},
			})
			return
		}

		response := h.processInvocation(r.Context(), toolName, args, fmt.Sprintf("%v", req.ID), true)

		if !response.Valid {
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    CodeInvalidParams,
					Message: fmt.Sprintf("Missing required parameters: %v", response.Diffs),
				},
			})
			return
		}

		// MCP format result
		_ = json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": formatOutputAsText(response.Result),
					},
				},
				"isError": response.Status == "fallback",
			},
		})
		return

	default:
		_ = json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    CodeMethodNotFound,
				Message: fmt.Sprintf("Method '%s' not supported", req.Method),
			},
		})
	}
}

func (h *Interceptor) processInvocation(ctx context.Context, toolName string, rawArgs any, callID string, executeTarget bool) *ToolInvocationResponse {
	if callID == "" {
		callID = generateRandomID("inv_")
	}

	start := time.Now()
	toolDef, _ := h.catalog.Get(toolName)

	// Step 1: Reconcile arguments using fuzzy engine
	report := h.engine.Reconcile(toolName, rawArgs, toolDef)

	var execResult *executor.ExecutionResult
	status := "passthrough_success"

	if report.Healed {
		status = "healed_success"
	}

	// Step 2: Execute target tool if requested and parameters are valid
	if executeTarget && report.Valid {
		execResult = h.dispatcher.Dispatch(ctx, toolDef, report.HealedArguments)
		if execResult.FallbackUsed {
			status = "fallback"
		}
	} else if !report.Valid {
		status = "invalid"
	}

	totalLatencyUs := time.Since(start).Microseconds()
	tokensSaved := 0
	if report.Healed && report.Valid {
		tokensSaved = h.tokensCost
	}

	// Step 3: Record Telemetry asynchronously
	record := &telemetry.InvocationRecord{
		ID:           callID,
		ToolName:     toolName,
		Status:       status,
		OriginalArgs: report.OriginalArguments,
		HealedArgs:   report.HealedArguments,
		Diffs:        report.Diffs,
		Healed:       report.Healed,
		Valid:        report.Valid,
		TokensSaved:  tokensSaved,
		LatencyUs:    totalLatencyUs,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	if h.store != nil {
		_ = h.store.RecordInvocation(record)
	}

	// Step 4: Broadcast to live console stream
	if h.feed != nil {
		h.feed.Broadcast("invocation", record)
	}

	var outputResult any
	if execResult != nil {
		outputResult = execResult.Output
	} else {
		outputResult = report.HealedArguments
	}

	return &ToolInvocationResponse{
		ID:        callID,
		Tool:      toolName,
		Status:    status,
		Healed:    report.Healed,
		Valid:     report.Valid,
		Arguments: report.HealedArguments,
		Diffs:     report.Diffs,
		Result:    outputResult,
		LatencyUs: totalLatencyUs,
	}
}

func (h *Interceptor) parseInvocationPayload(r io.Reader) (string, any, string, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to read body: %w", err)
	}

	var req ToolInvocationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return "", nil, "", fmt.Errorf("invalid json request: %w", err)
	}

	toolName := req.Tool
	if toolName == "" {
		toolName = req.Name
	}
	callID := req.ID

	// Handle OpenAI / Anthropic function wrapper
	if req.Function != nil {
		if toolName == "" {
			toolName = req.Function.Name
		}
		var args map[string]any
		// Check if function arguments are stringified json or raw object
		if len(req.Function.Arguments) > 0 {
			trimmed := strings.TrimSpace(string(req.Function.Arguments))
			if strings.HasPrefix(trimmed, "\"") {
				var unquoted string
				_ = json.Unmarshal([]byte(trimmed), &unquoted)
				_ = json.Unmarshal([]byte(unquoted), &args)
			} else {
				_ = json.Unmarshal(req.Function.Arguments, &args)
			}
		}
		return toolName, args, callID, nil
	}

	// Handle standard arguments
	var args any
	if len(req.Arguments) > 0 {
		var parsed any
		if err := json.Unmarshal(req.Arguments, &parsed); err == nil {
			args = parsed
		} else {
			args = string(req.Arguments)
		}
	}

	if toolName == "" {
		return "", nil, "", fmt.Errorf("tool or name property required")
	}

	return toolName, args, callID, nil
}

func generateRandomID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func formatOutputAsText(out any) string {
	if out == nil {
		return "{}"
	}
	switch v := out.(type) {
	case string:
		return v
	default:
		bytes, err := json.MarshalIndent(v, "", "  ")
		if err == nil {
			return string(bytes)
		}
		return fmt.Sprintf("%v", v)
	}
}
