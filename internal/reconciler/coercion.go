package reconciler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alexandrmotologa/toolheal/internal/config"
)

// CoerceResult holds the coerced value and whether an alteration occurred.
type CoerceResult struct {
	Value       any
	Changed     bool
	CoercionType string
}

// CoerceValue coerces an input value to match the expected property schema type.
func CoerceValue(val any, prop config.PropertySchema) CoerceResult {
	if val == nil {
		if prop.Default != nil {
			return CoerceResult{Value: prop.Default, Changed: true, CoercionType: "DEFAULT_VALUE"}
		}
		return CoerceResult{Value: nil, Changed: false}
	}

	expectedType := strings.ToLower(prop.Type)

	// Check if input is a stringified JSON object or array
	if strVal, isStr := val.(string); isStr && (expectedType == "object" || expectedType == "array") {
		trimmed := strings.TrimSpace(strVal)
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			var parsed any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				return CoerceValue(parsed, prop)
			}
		}
	}

	switch expectedType {
	case "integer":
		return coerceToInteger(val)
	case "number":
		return coerceToFloat(val)
	case "boolean":
		return coerceToBoolean(val)
	case "string":
		return coerceToString(val)
	case "array":
		return coerceToArray(val, prop.Items)
	default:
		return CoerceResult{Value: val, Changed: false}
	}
}

func coerceToInteger(val any) CoerceResult {
	switch v := val.(type) {
	case int:
		return CoerceResult{Value: v, Changed: false}
	case int64:
		return CoerceResult{Value: int(v), Changed: false}
	case float64:
		// JSON unmarshals all numbers as float64
		intVal := int(v)
		return CoerceResult{Value: intVal, Changed: false}
	case string:
		trimmed := strings.TrimSpace(v)
		if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return CoerceResult{Value: int(i), Changed: true, CoercionType: "STRING_TO_INT"}
		}
		// If float representation e.g. "100.0"
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return CoerceResult{Value: int(f), Changed: true, CoercionType: "STRING_TO_INT"}
		}
	case bool:
		if v {
			return CoerceResult{Value: 1, Changed: true, CoercionType: "BOOL_TO_INT"}
		}
		return CoerceResult{Value: 0, Changed: true, CoercionType: "BOOL_TO_INT"}
	case []any:
		if len(v) == 1 {
			res := coerceToInteger(v[0])
			return CoerceResult{Value: res.Value, Changed: true, CoercionType: "ARRAY_UNWRAP"}
		}
	}
	return CoerceResult{Value: val, Changed: false}
}

func coerceToFloat(val any) CoerceResult {
	switch v := val.(type) {
	case float64:
		return CoerceResult{Value: v, Changed: false}
	case int:
		return CoerceResult{Value: float64(v), Changed: false}
	case string:
		trimmed := strings.TrimSpace(v)
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return CoerceResult{Value: f, Changed: true, CoercionType: "STRING_TO_FLOAT"}
		}
	case []any:
		if len(v) == 1 {
			res := coerceToFloat(v[0])
			return CoerceResult{Value: res.Value, Changed: true, CoercionType: "ARRAY_UNWRAP"}
		}
	}
	return CoerceResult{Value: val, Changed: false}
}

func coerceToBoolean(val any) CoerceResult {
	switch v := val.(type) {
	case bool:
		return CoerceResult{Value: v, Changed: false}
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "true", "1", "yes", "y", "t", "on":
			return CoerceResult{Value: true, Changed: true, CoercionType: "STRING_TO_BOOL"}
		case "false", "0", "no", "n", "f", "off":
			return CoerceResult{Value: false, Changed: true, CoercionType: "STRING_TO_BOOL"}
		}
	case float64:
		return CoerceResult{Value: v != 0, Changed: true, CoercionType: "NUMBER_TO_BOOL"}
	case int:
		return CoerceResult{Value: v != 0, Changed: true, CoercionType: "NUMBER_TO_BOOL"}
	case []any:
		if len(v) == 1 {
			res := coerceToBoolean(v[0])
			return CoerceResult{Value: res.Value, Changed: true, CoercionType: "ARRAY_UNWRAP"}
		}
	}
	return CoerceResult{Value: val, Changed: false}
}

func coerceToString(val any) CoerceResult {
	switch v := val.(type) {
	case string:
		return CoerceResult{Value: v, Changed: false}
	case float64:
		// If whole integer
		if v == float64(int64(v)) {
			return CoerceResult{Value: strconv.FormatInt(int64(v), 10), Changed: true, CoercionType: "NUM_TO_STRING"}
		}
		return CoerceResult{Value: strconv.FormatFloat(v, 'f', -1, 64), Changed: true, CoercionType: "NUM_TO_STRING"}
	case int:
		return CoerceResult{Value: strconv.Itoa(v), Changed: true, CoercionType: "NUM_TO_STRING"}
	case bool:
		return CoerceResult{Value: strconv.FormatBool(v), Changed: true, CoercionType: "BOOL_TO_STRING"}
	case []any:
		if len(v) == 1 {
			res := coerceToString(v[0])
			return CoerceResult{Value: res.Value, Changed: true, CoercionType: "ARRAY_UNWRAP"}
		}
	}
	return CoerceResult{Value: fmt.Sprintf("%v", val), Changed: true, CoercionType: "ANY_TO_STRING"}
}

func coerceToArray(val any, itemSchema *config.PropertySchema) CoerceResult {
	switch v := val.(type) {
	case []any:
		// Array is already an array, optionally coerce individual items
		if itemSchema != nil && len(v) > 0 {
			changedAny := false
			items := make([]any, len(v))
			for i, item := range v {
				c := CoerceValue(item, *itemSchema)
				items[i] = c.Value
				if c.Changed {
					changedAny = true
				}
			}
			return CoerceResult{Value: items, Changed: changedAny, CoercionType: "ITEM_COERCE"}
		}
		return CoerceResult{Value: v, Changed: false}
	case string:
		trimmed := strings.TrimSpace(v)
		// Check if comma-separated values string e.g. "tag1, tag2, tag3"
		if strings.Contains(trimmed, ",") {
			parts := strings.Split(trimmed, ",")
			slice := make([]any, len(parts))
			for i, p := range parts {
				slice[i] = strings.TrimSpace(p)
			}
			return CoerceResult{Value: slice, Changed: true, CoercionType: "COMMA_STRING_TO_ARRAY"}
		}
		// Wrap scalar string into array
		return CoerceResult{Value: []any{trimmed}, Changed: true, CoercionType: "SCALAR_WRAP"}
	default:
		// Wrap any scalar into slice
		return CoerceResult{Value: []any{val}, Changed: true, CoercionType: "SCALAR_WRAP"}
	}
}
