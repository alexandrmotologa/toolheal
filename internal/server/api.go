package server

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/alexandrmotologa/toolheal/internal/config"
	"github.com/alexandrmotologa/toolheal/internal/middleware"
	"github.com/alexandrmotologa/toolheal/internal/telemetry"
)

// APIHandler coordinates REST management routes and metrics inspection.
type APIHandler struct {
	catalog     *config.Catalog
	store       *telemetry.Store
	feed        *telemetry.SSEFeed
	interceptor *middleware.Interceptor
}

// NewAPIHandler initializes the API router dependencies.
func NewAPIHandler(
	catalog *config.Catalog,
	store *telemetry.Store,
	feed *telemetry.SSEFeed,
	interceptor *middleware.Interceptor,
) *APIHandler {
	return &APIHandler{
		catalog:     catalog,
		store:       store,
		feed:        feed,
		interceptor: interceptor,
	}
}

// RegisterRoutes registers all API, middleware, and SSE endpoints on the mux.
func (a *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	// Agent Tool Execution Endpoints
	mux.HandleFunc("/v1/tools/execute", a.interceptor.HandleExecute)
	mux.HandleFunc("/v1/heal", a.interceptor.HandleHeal)
	mux.HandleFunc("/rpc", a.interceptor.HandleJSONRPC)

	// Dashboard Management and Telemetry Endpoints
	mux.HandleFunc("/api/metrics", a.handleMetrics)
	mux.HandleFunc("/api/invocations", a.handleInvocations)
	mux.HandleFunc("/api/invocations/export", a.handleExportInvocations)
	mux.HandleFunc("/api/drifts", a.handleDrifts)
	mux.HandleFunc("/api/tools", a.handleTools)
	mux.HandleFunc("/api/tools/import", a.handleImportTools)
	mux.HandleFunc("/api/patches/suggested", a.handleSuggestedPatches)
	mux.HandleFunc("/api/patches/apply", a.handleApplyPatch)
	mux.HandleFunc("/api/bench/run", a.handleRunBenchmark)
	mux.HandleFunc("/api/heal", a.interceptor.HandleHeal) // sandbox proxy
	mux.HandleFunc("/api/stream", a.feed.ServeHTTP)
}

func (a *APIHandler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if a.store == nil {
		_ = json.NewEncoder(w).Encode(telemetry.TelemetryMetrics{
			RecoveryRate: 100.0,
		})
		return
	}

	metrics, err := a.store.GetMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(metrics)
}

func (a *APIHandler) handleInvocations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if a.store == nil {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	invs, err := a.store.GetRecentInvocations(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if invs == nil {
		invs = make([]*telemetry.InvocationRecord, 0)
	}

	_ = json.NewEncoder(w).Encode(invs)
}

func (a *APIHandler) handleExportInvocations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if a.store == nil {
		http.Error(w, "Store not initialized", http.StatusInternalServerError)
		return
	}

	invs, err := a.store.GetRecentInvocations(5000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=toolheal_invocations.csv")
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"ID", "Tool", "Status", "Healed", "Valid", "TokensSaved", "LatencyUs", "CreatedAt"})
		for _, inv := range invs {
			_ = writer.Write([]string{
				inv.ID,
				inv.ToolName,
				inv.Status,
				strconv.FormatBool(inv.Healed),
				strconv.FormatBool(inv.Valid),
				strconv.Itoa(inv.TokensSaved),
				strconv.FormatInt(inv.LatencyUs, 10),
				inv.CreatedAt,
			})
		}
		writer.Flush()
		return
	}

	// Default to JSON export
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=toolheal_invocations.json")
	_ = json.NewEncoder(w).Encode(invs)
}

func (a *APIHandler) handleDrifts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if a.store == nil {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	drifts, err := a.store.GetDriftHeatmap()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if drifts == nil {
		drifts = make([]*telemetry.DriftItem, 0)
	}

	_ = json.NewEncoder(w).Encode(drifts)
}

func (a *APIHandler) handleTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		tools := a.catalog.List()
		_ = json.NewEncoder(w).Encode(tools)
		return
	}

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		if err := a.catalog.LoadFromJSON(body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "registered",
			"tools":  a.catalog.List(),
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (a *APIHandler) handleImportTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var payload struct {
		URL  string `json:"url"`
		Spec string `json:"spec"`
	}
	_ = json.Unmarshal(body, &payload)

	var count int
	if payload.URL != "" {
		count, err = a.catalog.FetchAndLoadOpenAPI(payload.URL)
		if err != nil {
			http.Error(w, fmt.Sprintf("Import from URL failed: %v", err), http.StatusBadRequest)
			return
		}
	} else if payload.Spec != "" {
		count, err = a.catalog.LoadFromOpenAPI([]byte(payload.Spec))
		if err != nil {
			http.Error(w, fmt.Sprintf("Import from spec failed: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Attempt direct parse of body as OpenAPI
		count, err = a.catalog.LoadFromOpenAPI(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid import payload: %v", err), http.StatusBadRequest)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":         "success",
		"imported_count": count,
		"total_tools":    len(a.catalog.List()),
	})
}

func (a *APIHandler) handleSuggestedPatches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if a.interceptor == nil || a.interceptor.Engine() == nil {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	patches := a.interceptor.Engine().FastCache().ListSuggestedPatches(1)
	_ = json.NewEncoder(w).Encode(patches)
}

func (a *APIHandler) handleApplyPatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ToolName     string `json:"tool_name"`
		CanonicalKey string `json:"canonical_key"`
		Alias        string `json:"alias"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid patch payload", http.StatusBadRequest)
		return
	}

	tool, exists := a.catalog.Get(req.ToolName)
	if !exists {
		http.Error(w, fmt.Sprintf("Tool '%s' not found", req.ToolName), http.StatusNotFound)
		return
	}

	prop, propExists := tool.Parameters.Properties[req.CanonicalKey]
	if !propExists {
		http.Error(w, fmt.Sprintf("Property '%s' not found in tool '%s'", req.CanonicalKey, req.ToolName), http.StatusBadRequest)
		return
	}

	// Add alias if not already present
	hasAlias := false
	for _, al := range prop.Aliases {
		if strings.EqualFold(al, req.Alias) {
			hasAlias = true
			break
		}
	}
	if !hasAlias {
		prop.Aliases = append(prop.Aliases, req.Alias)
		tool.Parameters.Properties[req.CanonicalKey] = prop
		a.catalog.Register(tool)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "applied",
		"tool":    req.ToolName,
		"key":     req.CanonicalKey,
		"aliases": prop.Aliases,
	})
}

func (a *APIHandler) handleRunBenchmark(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, err := os.Open("examples/failing_invocations.jsonl")
	if err != nil {
		http.Error(w, "Benchmark file not found", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	engine := a.interceptor.Engine()
	totalCases := 0
	healedCases := 0
	validCases := 0
	var totalLatencyUs int64 = 0
	fixCounts := make(map[string]int)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var item struct {
			Tool      string `json:"tool"`
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}

		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}

		toolName := item.Tool
		if toolName == "" {
			toolName = item.Name
		}

		toolDef, _ := a.catalog.Get(toolName)
		report := engine.Reconcile(toolName, item.Arguments, toolDef)

		totalCases++
		totalLatencyUs += report.LatencyUs
		if report.Healed {
			healedCases++
		}
		if report.Valid {
			validCases++
		}
		for _, d := range report.Diffs {
			fixCounts[d.FixType]++
		}
	}

	recoveryRate := 0.0
	avgLatencyUs := int64(0)
	if totalCases > 0 {
		recoveryRate = (float64(validCases) / float64(totalCases)) * 100.0
		avgLatencyUs = totalLatencyUs / int64(totalCases)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"total_cases":      totalCases,
		"healed_cases":     healedCases,
		"valid_cases":      validCases,
		"recovery_rate":    recoveryRate,
		"avg_latency_us":   avgLatencyUs,
		"tokens_saved":     validCases * 2400,
		"dollars_saved":    float64(validCases*2400) * 0.000003, // Based on $3 per 1M tokens
		"fix_breakdown":    fixCounts,
	})
}
