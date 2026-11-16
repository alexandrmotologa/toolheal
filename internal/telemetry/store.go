package telemetry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/alexandrmotologa/toolheal/internal/config"
	"github.com/alexandrmotologa/toolheal/internal/reconciler"
)

// InvocationRecord stores a single tool invocation audit entry.
type InvocationRecord struct {
	ID           string                     `json:"id"`
	ToolName     string                     `json:"tool_name"`
	Status       string                     `json:"status"` // healed_success, passthrough, fallback, invalid
	OriginalArgs map[string]any             `json:"original_args"`
	HealedArgs   map[string]any             `json:"healed_args"`
	Diffs        []reconciler.ParameterDiff `json:"diffs"`
	Healed       bool                       `json:"healed"`
	Valid        bool                       `json:"valid"`
	TokensSaved  int                        `json:"tokens_saved"`
	LatencyUs    int64                      `json:"latency_us"`
	CreatedAt    string                     `json:"created_at"`
}

// DriftItem aggregates hallucination trends per tool and parameter.
type DriftItem struct {
	ToolName    string `json:"tool_name"`
	OriginalKey string `json:"original_key"`
	HealedKey   string `json:"healed_key"`
	FixType     string `json:"fix_type"`
	Occurrences int    `json:"occurrences"`
}

// TelemetryMetrics summarizes system-wide healing performance and token savings.
type TelemetryMetrics struct {
	TotalInvocations  int     `json:"total_invocations"`
	HealedInvocations int     `json:"healed_invocations"`
	RecoveryRate      float64 `json:"recovery_rate"` // Percentage e.g. 88.5
	TokensSaved       int     `json:"tokens_saved"`
	AverageLatencyUs  int64   `json:"average_latency_us"`
}

// Store handles embedded SQLite persistence for toolheal metrics.
type Store struct {
	mu  sync.RWMutex
	db  *sql.DB
	cfg *config.TelemetryConfig
}

// NewStore initializes a SQLite database file or in-memory store.
func NewStore(cfg *config.TelemetryConfig) (*Store, error) {
	if cfg == nil {
		cfg = &config.DefaultConfig().Telemetry
	}

	dsn := cfg.DatabasePath
	if dsn == "" {
		dsn = "toolheal.db"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize SQLite performance for concurrent reads and writes
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, p := range pragmas {
		_, _ = db.Exec(p)
	}

	store := &Store{
		db:  db,
		cfg: cfg,
	}

	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}

	return store, nil
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS invocations (
		id TEXT PRIMARY KEY,
		tool_name TEXT NOT NULL,
		status TEXT NOT NULL,
		original_args TEXT,
		healed_args TEXT,
		diffs_json TEXT,
		healed INTEGER NOT NULL,
		valid INTEGER NOT NULL,
		tokens_saved INTEGER NOT NULL,
		latency_us INTEGER NOT NULL,
		created_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_invocations_created_at ON invocations(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_invocations_tool ON invocations(tool_name);

	CREATE TABLE IF NOT EXISTS schema_drifts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tool_name TEXT NOT NULL,
		original_key TEXT NOT NULL,
		healed_key TEXT NOT NULL,
		fix_type TEXT NOT NULL,
		occurrences INTEGER NOT NULL DEFAULT 1,
		UNIQUE(tool_name, original_key, healed_key, fix_type)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// RecordInvocation persists an execution report and updates drift frequency.
func (s *Store) RecordInvocation(rec *InvocationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	origJSON, _ := json.Marshal(rec.OriginalArgs)
	healedJSON, _ := json.Marshal(rec.HealedArgs)
	diffsJSON, _ := json.Marshal(rec.Diffs)

	healedInt := 0
	if rec.Healed {
		healedInt = 1
	}
	validInt := 0
	if rec.Valid {
		validInt = 1
	}

	query := `
	INSERT INTO invocations (
		id, tool_name, status, original_args, healed_args, diffs_json,
		healed, valid, tokens_saved, latency_us, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	_, err := s.db.Exec(query,
		rec.ID, rec.ToolName, rec.Status,
		string(origJSON), string(healedJSON), string(diffsJSON),
		healedInt, validInt, rec.TokensSaved, rec.LatencyUs, rec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert invocation: %w", err)
	}

	// Update schema drifts table
	driftQuery := `
	INSERT INTO schema_drifts (tool_name, original_key, healed_key, fix_type, occurrences)
	VALUES (?, ?, ?, ?, 1)
	ON CONFLICT(tool_name, original_key, healed_key, fix_type)
	DO UPDATE SET occurrences = occurrences + 1;
	`
	for _, d := range rec.Diffs {
		_, _ = s.db.Exec(driftQuery, rec.ToolName, d.OriginalKey, d.HealedKey, d.FixType)
	}

	return nil
}

// GetRecentInvocations retrieves up to `limit` latest invocations.
func (s *Store) GetRecentInvocations(limit int) ([]*InvocationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT id, tool_name, status, original_args, healed_args, diffs_json,
	       healed, valid, tokens_saved, latency_us, created_at
	FROM invocations
	ORDER BY created_at DESC
	LIMIT ?;
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query invocations: %w", err)
	}
	defer rows.Close()

	var records []*InvocationRecord
	for rows.Next() {
		var r InvocationRecord
		var origStr, healedStr, diffsStr string
		var healedInt, validInt int

		err := rows.Scan(
			&r.ID, &r.ToolName, &r.Status, &origStr, &healedStr, &diffsStr,
			&healedInt, &validInt, &r.TokensSaved, &r.LatencyUs, &r.CreatedAt,
		)
		if err != nil {
			continue
		}

		_ = json.Unmarshal([]byte(origStr), &r.OriginalArgs)
		_ = json.Unmarshal([]byte(healedStr), &r.HealedArgs)
		_ = json.Unmarshal([]byte(diffsStr), &r.Diffs)
		r.Healed = healedInt == 1
		r.Valid = validInt == 1

		records = append(records, &r)
	}

	return records, nil
}

// GetMetrics computes high-level KPIs across all invocations.
func (s *Store) GetMetrics() (*TelemetryMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT
		COUNT(*) as total,
		COALESCE(SUM(CASE WHEN healed = 1 THEN 1 ELSE 0 END), 0) as healed_count,
		COALESCE(SUM(tokens_saved), 0) as tokens_saved,
		COALESCE(AVG(latency_us), 0) as avg_latency
	FROM invocations;
	`

	var m TelemetryMetrics
	var avgLat float64

	row := s.db.QueryRow(query)
	err := row.Scan(&m.TotalInvocations, &m.HealedInvocations, &m.TokensSaved, &avgLat)
	if err != nil {
		return &TelemetryMetrics{}, nil
	}

	m.AverageLatencyUs = int64(avgLat)
	if m.TotalInvocations > 0 {
		m.RecoveryRate = (float64(m.HealedInvocations) / float64(m.TotalInvocations)) * 100.0
	} else {
		m.RecoveryRate = 100.0
	}

	return &m, nil
}

// GetDriftHeatmap returns top parameter drifts sorted by frequency.
func (s *Store) GetDriftHeatmap() ([]*DriftItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT tool_name, original_key, healed_key, fix_type, occurrences
	FROM schema_drifts
	ORDER BY occurrences DESC
	LIMIT 100;
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query drift heatmap: %w", err)
	}
	defer rows.Close()

	var drifts []*DriftItem
	for rows.Next() {
		var d DriftItem
		if err := rows.Scan(&d.ToolName, &d.OriginalKey, &d.HealedKey, &d.FixType, &d.Occurrences); err == nil {
			drifts = append(drifts, &d)
		}
	}

	return drifts, nil
}

// Close gracefully closes the SQLite database connection.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
