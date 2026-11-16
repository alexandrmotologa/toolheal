package reconciler

import (
	"strings"
	"unicode"
)

// FoldKey strips all delimiters (underscores, hyphens, spaces, dots) and converts to lowercase.
// For example: "user_id", "userId", "user-id", "User_Id" all become "userid".
func FoldKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '_' || r == '-' || r == ' ' || r == '.' || r == '/' {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// ToCamelCase converts snake_case, kebab-case, or spaced strings to camelCase.
func ToCamelCase(s string) string {
	parts := splitWords(s)
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for i, part := range parts {
		lower := strings.ToLower(part)
		if i == 0 {
			b.WriteString(lower)
		} else if len(lower) > 0 {
			runes := []rune(lower)
			runes[0] = unicode.ToUpper(runes[0])
			b.WriteString(string(runes))
		}
	}
	return b.String()
}

// ToSnakeCase converts camelCase or kebab-case to snake_case.
func ToSnakeCase(s string) string {
	parts := splitWords(s)
	if len(parts) == 0 {
		return ""
	}
	lowerParts := make([]string, len(parts))
	for i, p := range parts {
		lowerParts[i] = strings.ToLower(p)
	}
	return strings.Join(lowerParts, "_")
}

// ToKebabCase converts camelCase or snake_case to kebab-case.
func ToKebabCase(s string) string {
	parts := splitWords(s)
	if len(parts) == 0 {
		return ""
	}
	lowerParts := make([]string, len(parts))
	for i, p := range parts {
		lowerParts[i] = strings.ToLower(p)
	}
	return strings.Join(lowerParts, "-")
}

// splitWords splits an identifier by delimiters and camelCase boundaries.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if r == '_' || r == '-' || r == ' ' || r == '.' || r == '/' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}

		// Detect camelCase boundary: lower followed by upper, or upper followed by lower
		if unicode.IsUpper(r) && current.Len() > 0 {
			prev := runes[i-1]
			if unicode.IsLower(prev) || (i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				words = append(words, current.String())
				current.Reset()
			}
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}
