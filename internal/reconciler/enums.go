package reconciler

import (
	"strings"
)

// EnumResult holds the outcome of snapping an enum value.
type EnumResult struct {
	Value   string
	Snapped bool
	Score   float64
}

// Synonyms mapping common drifting model phrasing to standard enum values.
var enumSynonyms = map[string]string{
	"urgent":       "HIGH",
	"critical":     "CRITICAL",
	"blocker":      "CRITICAL",
	"high":         "HIGH",
	"medium":       "MEDIUM",
	"normal":       "MEDIUM",
	"low":          "LOW",
	"minor":        "LOW",
	"active":       "ACTIVE",
	"running":      "ACTIVE",
	"online":       "ACTIVE",
	"enabled":      "ACTIVE",
	"disabled":     "DISABLED",
	"inactive":     "DISABLED",
	"offline":      "DISABLED",
	"archived":     "ARCHIVED",
	"deleted":      "ARCHIVED",
	"draft":        "DRAFT",
	"admin":        "ADMIN",
	"administrator": "ADMIN",
	"member":       "MEMBER",
	"user":         "MEMBER",
	"guest":        "GUEST",
	"visitor":      "GUEST",
	"readonly":     "READONLY",
	"read_only":    "READONLY",
	"usd":          "USD",
	"dollar":       "USD",
	"eur":          "EUR",
	"euro":         "EUR",
	"gbp":          "GBP",
	"pound":        "GBP",
}

// SnapEnum resolves a drifting string value to the nearest valid enum variant.
func SnapEnum(input string, allowed []string, threshold float64) EnumResult {
	if len(allowed) == 0 {
		return EnumResult{Value: input, Snapped: false}
	}

	trimmed := strings.TrimSpace(input)

	// 1. Exact match
	for _, variant := range allowed {
		if variant == trimmed {
			return EnumResult{Value: variant, Snapped: false, Score: 1.0}
		}
	}

	// 2. Case-insensitive match
	for _, variant := range allowed {
		if strings.EqualFold(variant, trimmed) {
			return EnumResult{Value: variant, Snapped: true, Score: 1.0}
		}
	}

	// 3. Folded match (stripping delimiters and lowercasing)
	trimmedFolded := FoldKey(trimmed)
	for _, variant := range allowed {
		if FoldKey(variant) == trimmedFolded {
			return EnumResult{Value: variant, Snapped: true, Score: 1.0}
		}
	}

	// 4. Synonym dictionary check
	lower := strings.ToLower(trimmed)
	if target, exists := enumSynonyms[lower]; exists {
		for _, variant := range allowed {
			if strings.EqualFold(variant, target) {
				return EnumResult{Value: variant, Snapped: true, Score: 0.95}
			}
		}
	}

	// 5. Jaro-Winkler fuzzy nearest-neighbor matching
	bestVariant := ""
	bestScore := -1.0

	for _, variant := range allowed {
		score := JaroWinkler(strings.ToLower(trimmed), strings.ToLower(variant))
		if score > bestScore {
			bestScore = score
			bestVariant = variant
		}
	}

	if bestScore >= threshold && bestVariant != "" {
		return EnumResult{Value: bestVariant, Snapped: true, Score: bestScore}
	}

	return EnumResult{Value: input, Snapped: false, Score: bestScore}
}
