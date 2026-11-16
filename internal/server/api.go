package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

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
	mux.HandleFunc("/api/drifts", a.handleDrifts)
	mux.HandleFunc("/api/tools", a.handleTools)
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
