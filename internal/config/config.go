package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig represents global settings for the toolheal middleware daemon and CLI.
type AppConfig struct {
	Server         ServerConfig         `json:"server" yaml:"server"`
	Reconciliation ReconciliationConfig `json:"reconciliation" yaml:"reconciliation"`
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker" yaml:"circuit_breaker"`
	Telemetry      TelemetryConfig      `json:"telemetry" yaml:"telemetry"`
	ToolSchemas    string               `json:"tool_schemas" yaml:"tool_schemas"`
}

// ServerConfig configures the local HTTP and SSE daemon.
type ServerConfig struct {
	Host             string `json:"host" yaml:"host"`
	Port             int    `json:"port" yaml:"port"`
	EnableConsole    bool   `json:"enable_console" yaml:"enable_console"`
	ReadTimeoutSec   int    `json:"read_timeout_sec" yaml:"read_timeout_sec"`
	WriteTimeoutSec  int    `json:"write_timeout_sec" yaml:"write_timeout_sec"`
	TargetUpstreamURL string `json:"target_upstream_url,omitempty" yaml:"target_upstream_url,omitempty"`
}

// ReconciliationConfig controls the sensitivity and rules of fuzzy parameter matching.
type ReconciliationConfig struct {
	LevenshteinThreshold     float64 `json:"levenshtein_threshold" yaml:"levenshtein_threshold"`
	JaroWinklerThreshold     float64 `json:"jaro_winkler_threshold" yaml:"jaro_winkler_threshold"`
	MaxEditDistance          int     `json:"max_edit_distance" yaml:"max_edit_distance"`
	EnableCaseFolding        bool    `json:"enable_case_folding" yaml:"enable_case_folding"`
	EnableTypeCoercion       bool    `json:"enable_type_coercion" yaml:"enable_type_coercion"`
	EnableEnumSnapping       bool    `json:"enable_enum_snapping" yaml:"enable_enum_snapping"`
	EnableDateNormalization  bool    `json:"enable_date_normalization" yaml:"enable_date_normalization"`
	EnableDefaultInjection   bool    `json:"enable_default_injection" yaml:"enable_default_injection"`
	EnableEnvelopeUnwrapping bool    `json:"enable_envelope_unwrapping" yaml:"enable_envelope_unwrapping"`
}

// CircuitBreakerConfig controls retry behavior and circuit breaking on upstream tool failures.
type CircuitBreakerConfig struct {
	MaxConsecutiveFailures int `json:"max_consecutive_failures" yaml:"max_consecutive_failures"`
	ResetTimeoutMs         int `json:"reset_timeout_ms" yaml:"reset_timeout_ms"`
	BackoffBaseMs          int `json:"backoff_base_ms" yaml:"backoff_base_ms"`
	BackoffMaxMs           int `json:"backoff_max_ms" yaml:"backoff_max_ms"`
}

// TelemetryConfig configures SQLite storage and token savings heuristics.
type TelemetryConfig struct {
	DatabasePath           string `json:"database_path" yaml:"database_path"`
	TokensSavedPerHealing  int    `json:"tokens_saved_per_healing" yaml:"tokens_saved_per_healing"`
	MaxHistoryItems        int    `json:"max_history_items" yaml:"max_history_items"`
}

// DefaultConfig provides production-ready default values.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Host:            "127.0.0.1",
			Port:            7171,
			EnableConsole:   true,
			ReadTimeoutSec:  15,
			WriteTimeoutSec: 15,
		},
		Reconciliation: ReconciliationConfig{
			LevenshteinThreshold:     0.80,
			JaroWinklerThreshold:     0.82,
			MaxEditDistance:          2,
			EnableCaseFolding:        true,
			EnableTypeCoercion:       true,
			EnableEnumSnapping:       true,
			EnableDateNormalization:  true,
			EnableDefaultInjection:   true,
			EnableEnvelopeUnwrapping: true,
		},
		CircuitBreaker: CircuitBreakerConfig{
			MaxConsecutiveFailures: 5,
			ResetTimeoutMs:         10000,
			BackoffBaseMs:          200,
			BackoffMaxMs:           3000,
		},
		Telemetry: TelemetryConfig{
			DatabasePath:          "toolheal.db",
			TokensSavedPerHealing: 2400,
			MaxHistoryItems:       5000,
		},
		ToolSchemas: "examples/tool_definitions.json",
	}
}

// LoadConfig loads an AppConfig from a YAML file, falling back to defaults if not found.
func LoadConfig(path string) (*AppConfig, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	return cfg, nil
}
