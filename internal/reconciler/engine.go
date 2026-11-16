package reconciler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alexandrmotologa/toolheal/internal/config"
)

// ParameterDiff records a single repair performed on an argument.
type ParameterDiff struct {
	OriginalKey   string  `json:"original_key"`
	HealedKey     string  `json:"healed_key"`
	OriginalValue any     `json:"original_value"`
	HealedValue   any     `json:"healed_value"`
	FixType       string  `json:"fix_type"` // KEY_CASE, KEY_ALIAS, KEY_FUZZY, TYPE_COERCE, ENUM_SNAP, DATE_NORMALIZE, DEFAULT_INJECT, ENVELOPE_UNWRAP
	Description   string  `json:"description"`
	Confidence    float64 `json:"confidence"`
}

// ReconciliationReport contains the complete outcome of the healing pass.
type ReconciliationReport struct {
	ToolName          string          `json:"tool_name"`
	OriginalArguments map[string]any  `json:"original_arguments"`
	HealedArguments   map[string]any  `json:"healed_arguments"`
	Diffs             []ParameterDiff `json:"diffs"`
	Healed            bool            `json:"healed"`
	Valid             bool            `json:"valid"`
	MissingRequired   []string        `json:"missing_required,omitempty"`
	LatencyUs         int64           `json:"latency_us"`
}

// Engine coordinates the multi-stage fuzzy schema reconciliation pipeline.
type Engine struct {
	config *config.ReconciliationConfig
}

// NewEngine creates a new reconciliation engine instance.
func NewEngine(cfg *config.ReconciliationConfig) *Engine {
	if cfg == nil {
		cfg = &config.DefaultConfig().Reconciliation
	}
	return &Engine{config: cfg}
}

// Reconcile executes the healing pipeline on raw tool arguments against a tool schema.
func (e *Engine) Reconcile(toolName string, rawArgs any, schema *config.ToolDefinition) *ReconciliationReport {
	start := time.Now()

	report := &ReconciliationReport{
		ToolName:          toolName,
		OriginalArguments: make(map[string]any),
		HealedArguments:   make(map[string]any),
		Diffs:             make([]ParameterDiff, 0),
		Healed:            false,
		Valid:             true,
	}

	// 1. Normalize input arguments into a map[string]any
	inputMap := normalizeRawArguments(rawArgs)
	for k, v := range inputMap {
		report.OriginalArguments[k] = v
	}

	if schema == nil {
		report.HealedArguments = inputMap
		report.LatencyUs = time.Since(start).Microseconds()
		return report
	}

	// 2. Envelope unwrapping (if model passed {"arguments": {...}} or {"params": {...}})
	if e.config.EnableEnvelopeUnwrapping {
		inputMap = e.unwrapEnvelope(inputMap, schema, report)
	}

	canonicalProps := schema.Parameters.Properties
	canonicalKeys := make([]string, 0, len(canonicalProps))
	for k := range canonicalProps {
		canonicalKeys = append(canonicalKeys, k)
	}
	sort.Strings(canonicalKeys)

	matchedCanonical := make(map[string]bool)
	healedMap := make(map[string]any)

	// 3. Match each input key to a canonical property
	for inKey, inVal := range inputMap {
		canonicalKey, fixType, conf, matched := e.resolveKey(inKey, canonicalKeys, canonicalProps)

		if matched {
			matchedCanonical[canonicalKey] = true
			prop := canonicalProps[canonicalKey]

			// Coerce value according to schema property rules
			healedVal, valDiffs := e.healValue(inVal, canonicalKey, prop)

			healedMap[canonicalKey] = healedVal

			// If key changed, record diff
			if inKey != canonicalKey {
				report.Diffs = append(report.Diffs, ParameterDiff{
					OriginalKey:   inKey,
					HealedKey:     canonicalKey,
					OriginalValue: inVal,
					HealedValue:   healedVal,
					FixType:       fixType,
					Description:   fmt.Sprintf("Renamed hallucinated parameter '%s' to '%s'", inKey, canonicalKey),
					Confidence:    conf,
				})
				report.Healed = true
			}

			// Add value coercion diffs
			for _, vd := range valDiffs {
				vd.OriginalKey = inKey
				vd.HealedKey = canonicalKey
				report.Diffs = append(report.Diffs, vd)
				report.Healed = true
			}
		} else {
			// Unrecognized parameter passed through unchanged
			healedMap[inKey] = inVal
		}
	}

	// 4. Inject default values for missing required or optional parameters
	if e.config.EnableDefaultInjection {
		for cKey, prop := range canonicalProps {
			if _, exists := healedMap[cKey]; !exists {
				if prop.Default != nil {
					healedMap[cKey] = prop.Default
					report.Diffs = append(report.Diffs, ParameterDiff{
						OriginalKey:   cKey,
						HealedKey:     cKey,
						OriginalValue: nil,
						HealedValue:   prop.Default,
						FixType:       "DEFAULT_INJECT",
						Description:   fmt.Sprintf("Injected default value for missing property '%s'", cKey),
						Confidence:    1.0,
					})
					report.Healed = true
				}
			}
		}
	}

	// 5. Check missing required properties
	missing := make([]string, 0)
	for _, reqKey := range schema.Parameters.Required {
		if _, ok := healedMap[reqKey]; !ok {
			missing = append(missing, reqKey)
			report.Valid = false
		}
	}
	report.MissingRequired = missing
	report.HealedArguments = healedMap
	report.LatencyUs = time.Since(start).Microseconds()

	return report
}

// resolveKey finds the canonical schema key for an input parameter using layered heuristics.
func (e *Engine) resolveKey(inKey string, canonicalKeys []string, props map[string]config.PropertySchema) (string, string, float64, bool) {
	// Level 1: Exact match
	if _, ok := props[inKey]; ok {
		return inKey, "EXACT", 1.0, true
	}

	// Level 2: Property-level explicit aliases
	for cKey, prop := range props {
		if FindExplicitAlias(inKey, prop.Aliases) {
			return cKey, "KEY_ALIAS", 0.98, true
		}
	}

	// Level 3: Delimiter & Case folding (user_id -> userid -> userId)
	if e.config.EnableCaseFolding {
		inFolded := FoldKey(inKey)
		for _, cKey := range canonicalKeys {
			if FoldKey(cKey) == inFolded {
				return cKey, "KEY_CASE", 0.95, true
			}
		}
	}

	// Level 4: Global alias dictionary lookup
	if aliasKey, found := FindGlobalAlias(inKey, canonicalKeys); found {
		return aliasKey, "KEY_ALIAS", 0.90, true
	}

	// Level 5: Fuzzy Levenshtein and Jaro-Winkler alignment
	fuzzyRes := FindBestFuzzyMatch(inKey, canonicalKeys, e.config.LevenshteinThreshold, e.config.MaxEditDistance)
	if fuzzyRes.Matched {
		return fuzzyRes.CanonicalKey, "KEY_FUZZY", fuzzyRes.Score, true
	}

	return inKey, "", 0.0, false
}

// healValue validates and repairs parameter types, enums, and timestamps.
func (e *Engine) healValue(val any, key string, prop config.PropertySchema) (any, []ParameterDiff) {
	diffs := make([]ParameterDiff, 0)
	current := val

	// 1. Primitive type coercion
	if e.config.EnableTypeCoercion && prop.Type != "" {
		res := CoerceValue(current, prop)
		if res.Changed {
			diffs = append(diffs, ParameterDiff{
				OriginalValue: current,
				HealedValue:   res.Value,
				FixType:       "TYPE_COERCE",
				Description:   fmt.Sprintf("Coerced parameter '%s' via %s", key, res.CoercionType),
				Confidence:    0.95,
			})
			current = res.Value
		}
	}

	// 2. Strict enum snapping
	if e.config.EnableEnumSnapping && len(prop.Enum) > 0 {
		if strVal, isStr := current.(string); isStr {
			snapRes := SnapEnum(strVal, prop.Enum, e.config.JaroWinklerThreshold)
			if snapRes.Snapped {
				diffs = append(diffs, ParameterDiff{
					OriginalValue: current,
					HealedValue:   snapRes.Value,
					FixType:       "ENUM_SNAP",
					Description:   fmt.Sprintf("Snapped drifting enum '%s' to '%s'", strVal, snapRes.Value),
					Confidence:    snapRes.Score,
				})
				current = snapRes.Value
			}
		}
	}

	// 3. Temporal / Date normalization
	if e.config.EnableDateNormalization && (prop.Format == "date" || prop.Format == "date-time") {
		dateRes := NormalizeDate(current, prop.Format)
		if dateRes.Normalized && dateRes.Value != current {
			diffs = append(diffs, ParameterDiff{
				OriginalValue: current,
				HealedValue:   dateRes.Value,
				FixType:       "DATE_NORMALIZE",
				Description:   fmt.Sprintf("Normalized date/timestamp into '%s' format", prop.Format),
				Confidence:    0.95,
			})
			current = dateRes.Value
		}
	}

	return current, diffs
}

// unwrapEnvelope extracts arguments if wrapped inside an envelope key.
func (e *Engine) unwrapEnvelope(input map[string]any, schema *config.ToolDefinition, report *ReconciliationReport) map[string]any {
	envelopeKeys := []string{"arguments", "params", "parameters", "input", "payload"}

	for _, ek := range envelopeKeys {
		if raw, ok := input[ek]; ok && len(input) == 1 {
			// Check if raw is a map
			if innerMap, isMap := raw.(map[string]any); isMap {
				report.Diffs = append(report.Diffs, ParameterDiff{
					OriginalKey:   ek,
					HealedKey:     "*",
					OriginalValue: input,
					HealedValue:   innerMap,
					FixType:       "ENVELOPE_UNWRAP",
					Description:   fmt.Sprintf("Unwrapped nested payload from '%s' envelope", ek),
					Confidence:    1.0,
				})
				report.Healed = true
				return innerMap
			}

			// Check if raw is a JSON string
			if strVal, isStr := raw.(string); isStr {
				var parsed map[string]any
				if err := json.Unmarshal([]byte(strVal), &parsed); err == nil {
					report.Diffs = append(report.Diffs, ParameterDiff{
						OriginalKey:   ek,
						HealedKey:     "*",
						OriginalValue: strVal,
						HealedValue:   parsed,
						FixType:       "ENVELOPE_UNWRAP",
						Description:   fmt.Sprintf("Parsed stringified JSON payload from '%s' envelope", ek),
						Confidence:    1.0,
					})
					report.Healed = true
					return parsed
				}
			}
		}
	}

	return input
}

// normalizeRawArguments converts stringified JSON or arbitrary structs into map[string]any.
func normalizeRawArguments(raw any) map[string]any {
	if raw == nil {
		return make(map[string]any)
	}

	switch v := raw.(type) {
	case map[string]any:
		return v
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return make(map[string]any)
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return parsed
		}
	case []byte:
		var parsed map[string]any
		if err := json.Unmarshal(v, &parsed); err == nil {
			return parsed
		}
	}

	// Try json roundtrip for structs
	bytes, err := json.Marshal(raw)
	if err == nil {
		var parsed map[string]any
		if err := json.Unmarshal(bytes, &parsed); err == nil {
			return parsed
		}
	}

	return make(map[string]any)
}
