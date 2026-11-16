package reconciler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/toolheal/internal/config"
)

// ParameterDiff records a single repair performed on an argument.
type ParameterDiff struct {
	OriginalKey   string  `json:"original_key"`
	HealedKey     string  `json:"healed_key"`
	OriginalValue any     `json:"original_value"`
	HealedValue   any     `json:"healed_value"`
	FixType       string  `json:"fix_type"` // KEY_CASE, KEY_ALIAS, KEY_FUZZY, TYPE_COERCE, ENUM_SNAP, DATE_NORMALIZE, DEFAULT_INJECT, ENVELOPE_UNWRAP, NESTED_OBJECT
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

// CachedResolution stores pre-computed key mappings for fast-path O(1) resolution.
type CachedResolution struct {
	ToolName     string  `json:"tool_name"`
	OriginalKey  string  `json:"original_key"`
	CanonicalKey string  `json:"canonical_key"`
	FixType      string  `json:"fix_type"`
	Confidence   float64 `json:"confidence"`
	Hits         int64   `json:"hits"`
}

// FastPathCache accelerates repeated parameter resolutions to sub-microsecond O(1) speed.
type FastPathCache struct {
	mu      sync.RWMutex
	entries map[string]*CachedResolution
}

// NewFastPathCache initializes the fast-path memory cache.
func NewFastPathCache() *FastPathCache {
	return &FastPathCache{
		entries: make(map[string]*CachedResolution),
	}
}

// Get retrieves a pre-computed key resolution if available.
func (c *FastPathCache) Get(toolName, originalKey string) (*CachedResolution, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	lookup := toolName + ":" + originalKey
	res, ok := c.entries[lookup]
	if ok {
		res.Hits++
		return res, true
	}
	return nil, false
}

// Set stores a newly computed resolution.
func (c *FastPathCache) Set(toolName, originalKey, canonicalKey, fixType string, confidence float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	lookup := toolName + ":" + originalKey
	c.entries[lookup] = &CachedResolution{
		ToolName:     toolName,
		OriginalKey:  originalKey,
		CanonicalKey: canonicalKey,
		FixType:      fixType,
		Confidence:   confidence,
		Hits:         1,
	}
}

// ListSuggestedPatches returns resolutions with high hit counts suitable for schema updates.
func (c *FastPathCache) ListSuggestedPatches(minHits int64) []*CachedResolution {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*CachedResolution, 0)
	for _, res := range c.entries {
		if res.Hits >= minHits && res.OriginalKey != res.CanonicalKey {
			out = append(out, res)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Hits > out[j].Hits
	})
	return out
}

// Engine coordinates the multi-stage fuzzy schema reconciliation pipeline.
type Engine struct {
	config    *config.ReconciliationConfig
	fastCache *FastPathCache
}

// NewEngine creates a new reconciliation engine instance.
func NewEngine(cfg *config.ReconciliationConfig) *Engine {
	if cfg == nil {
		cfg = &config.DefaultConfig().Reconciliation
	}
	return &Engine{
		config:    cfg,
		fastCache: NewFastPathCache(),
	}
}

// FastCache returns the engine's fast-path resolution cache.
func (e *Engine) FastCache() *FastPathCache {
	return e.fastCache
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

	// 3. Reconcile root level properties recursively
	healedMap, diffs, valid, missing := e.reconcileProperties(toolName, "", inputMap, schema.Parameters.Properties, schema.Parameters.Required)

	report.HealedArguments = healedMap
	report.Diffs = append(report.Diffs, diffs...)
	if len(diffs) > 0 {
		report.Healed = true
	}
	report.Valid = valid
	report.MissingRequired = missing
	report.LatencyUs = time.Since(start).Microseconds()

	return report
}

// reconcileProperties performs recursive reconciliation over a map of properties.
func (e *Engine) reconcileProperties(
	toolName string,
	prefix string,
	inputMap map[string]any,
	canonicalProps map[string]config.PropertySchema,
	requiredKeys []string,
) (map[string]any, []ParameterDiff, bool, []string) {
	diffs := make([]ParameterDiff, 0)
	canonicalKeys := make([]string, 0, len(canonicalProps))
	for k := range canonicalProps {
		canonicalKeys = append(canonicalKeys, k)
	}
	sort.Strings(canonicalKeys)

	healedMap := make(map[string]any)
	matchedCanonical := make(map[string]bool)

	// A. Match and heal provided parameters
	for inKey, inVal := range inputMap {
		canonicalKey, fixType, conf, matched := e.resolveKeyCached(toolName, inKey, canonicalKeys, canonicalProps)

		fullInKey := inKey
		fullCanKey := canonicalKey
		if prefix != "" {
			fullInKey = prefix + "." + inKey
			fullCanKey = prefix + "." + canonicalKey
		}

		if matched {
			matchedCanonical[canonicalKey] = true
			prop := canonicalProps[canonicalKey]

			// Coerce and recursively heal value
			healedVal, valDiffs := e.healValueRecursive(toolName, fullCanKey, inVal, prop)

			healedMap[canonicalKey] = healedVal

			// If key renamed, record diff
			if inKey != canonicalKey {
				diffs = append(diffs, ParameterDiff{
					OriginalKey:   fullInKey,
					HealedKey:     fullCanKey,
					OriginalValue: inVal,
					HealedValue:   healedVal,
					FixType:       fixType,
					Description:   fmt.Sprintf("Renamed hallucinated parameter '%s' to '%s'", fullInKey, fullCanKey),
					Confidence:    conf,
				})
			}

			// Add value coercion & nested diffs
			diffs = append(diffs, valDiffs...)
		} else {
			// Unrecognized parameter passed through unchanged
			healedMap[inKey] = inVal
		}
	}

	// B. Default value injection
	if e.config.EnableDefaultInjection {
		for cKey, prop := range canonicalProps {
			if _, exists := healedMap[cKey]; !exists {
				if prop.Default != nil {
					fullCKey := cKey
					if prefix != "" {
						fullCKey = prefix + "." + cKey
					}
					healedMap[cKey] = prop.Default
					diffs = append(diffs, ParameterDiff{
						OriginalKey:   fullCKey,
						HealedKey:     fullCKey,
						OriginalValue: nil,
						HealedValue:   prop.Default,
						FixType:       "DEFAULT_INJECT",
						Description:   fmt.Sprintf("Injected default value for missing property '%s'", fullCKey),
						Confidence:    1.0,
					})
				}
			}
		}
	}

	// C. Required check
	isValid := true
	missing := make([]string, 0)
	for _, reqKey := range requiredKeys {
		if _, ok := healedMap[reqKey]; !ok {
			missing = append(missing, reqKey)
			isValid = false
		}
	}

	return healedMap, diffs, isValid, missing
}

// resolveKeyCached checks the fast-path cache before executing full resolution algorithms.
func (e *Engine) resolveKeyCached(
	toolName, inKey string,
	canonicalKeys []string,
	props map[string]config.PropertySchema,
) (string, string, float64, bool) {
	// 1. Check FastPathCache
	if e.fastCache != nil {
		if cached, ok := e.fastCache.Get(toolName, inKey); ok {
			return cached.CanonicalKey, cached.FixType, cached.Confidence, true
		}
	}

	// 2. Full resolution pipeline
	cKey, fixType, conf, matched := e.resolveKey(inKey, canonicalKeys, props)

	// 3. Store in FastPathCache for subsequent calls
	if matched && e.fastCache != nil {
		e.fastCache.Set(toolName, inKey, cKey, fixType, conf)
	}

	return cKey, fixType, conf, matched
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

// healValueRecursive validates and repairs parameter types, enums, dates, and nested structures.
func (e *Engine) healValueRecursive(toolName, fullKey string, val any, prop config.PropertySchema) (any, []ParameterDiff) {
	diffs := make([]ParameterDiff, 0)
	current := val

	// 1. Primitive type coercion
	if e.config.EnableTypeCoercion && prop.Type != "" {
		res := CoerceValue(current, prop)
		if res.Changed {
			diffs = append(diffs, ParameterDiff{
				OriginalKey:   fullKey,
				HealedKey:     fullKey,
				OriginalValue: current,
				HealedValue:   res.Value,
				FixType:       "TYPE_COERCE",
				Description:   fmt.Sprintf("Coerced parameter '%s' via %s", fullKey, res.CoercionType),
				Confidence:    0.95,
			})
			current = res.Value
		}
	}

	// 2. Recursive Nested Object Healing
	if prop.Type == "object" && len(prop.Properties) > 0 {
		if objMap, ok := current.(map[string]any); ok {
			healedNested, nestedDiffs, _, _ := e.reconcileProperties(toolName, fullKey, objMap, prop.Properties, prop.Required)
			diffs = append(diffs, nestedDiffs...)
			current = healedNested
		}
	}

	// 3. Recursive Array of Objects Healing
	if prop.Type == "array" && prop.Items != nil {
		if slice, ok := current.([]any); ok {
			// If array of objects
			if prop.Items.Type == "object" && len(prop.Items.Properties) > 0 {
				newSlice := make([]any, len(slice))
				for i, item := range slice {
					if itemMap, isMap := item.(map[string]any); isMap {
						itemPrefix := fmt.Sprintf("%s[%d]", fullKey, i)
						healedItem, itemDiffs, _, _ := e.reconcileProperties(toolName, itemPrefix, itemMap, prop.Items.Properties, prop.Items.Required)
						diffs = append(diffs, itemDiffs...)
						newSlice[i] = healedItem
					} else {
						newSlice[i] = item
					}
				}
				current = newSlice
			}
		}
	}

	// 4. Strict enum snapping
	if e.config.EnableEnumSnapping && len(prop.Enum) > 0 {
		if strVal, isStr := current.(string); isStr {
			snapRes := SnapEnum(strVal, prop.Enum, e.config.JaroWinklerThreshold)
			if snapRes.Snapped {
				diffs = append(diffs, ParameterDiff{
					OriginalKey:   fullKey,
					HealedKey:     fullKey,
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

	// 5. Temporal / Date normalization
	if e.config.EnableDateNormalization && (prop.Format == "date" || prop.Format == "date-time") {
		dateRes := NormalizeDate(current, prop.Format)
		if dateRes.Normalized && dateRes.Value != current {
			diffs = append(diffs, ParameterDiff{
				OriginalKey:   fullKey,
				HealedKey:     fullKey,
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
