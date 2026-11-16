package reconciler

import (
	"strconv"
	"strings"
	"time"
)

// DateResult holds the normalized date string and whether parsing/reformatting succeeded.
type DateResult struct {
	Value      string
	Normalized bool
}

var knownDateLayouts = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"2006/01/02",
	"02/01/2006",
	"01/02/2006",
	"02-01-2006",
	"02 Jan 2006",
	"2 Jan 2006",
	"Jan 2, 2006",
	"January 2, 2006",
	"2006.01.02",
}

// NormalizeDate converts dates, relative keywords, and epoch timestamps to RFC3339 or YYYY-MM-DD.
func NormalizeDate(input any, targetFormat string) DateResult {
	if input == nil {
		return DateResult{Value: "", Normalized: false}
	}

	now := time.Now().UTC()
	var parsedTime time.Time
	var parseSuccess bool

	switch v := input.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return DateResult{Value: "", Normalized: false}
		}

		// 1. Relative keywords
		lower := strings.ToLower(trimmed)
		switch lower {
		case "now", "current":
			parsedTime = now
			parseSuccess = true
		case "today":
			parsedTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			parseSuccess = true
		case "yesterday":
			parsedTime = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)
			parseSuccess = true
		case "tomorrow":
			parsedTime = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
			parseSuccess = true
		}

		// 2. Numeric epoch string e.g. "1731682800"
		if !parseSuccess {
			if num, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
				parsedTime = parseEpoch(num)
				parseSuccess = true
			}
		}

		// 3. Known layout matching
		if !parseSuccess {
			for _, layout := range knownDateLayouts {
				if t, err := time.Parse(layout, trimmed); err == nil {
					parsedTime = t
					parseSuccess = true
					break
				}
			}
		}

	case float64:
		parsedTime = parseEpoch(int64(v))
		parseSuccess = true
	case int:
		parsedTime = parseEpoch(int64(v))
		parseSuccess = true
	case int64:
		parsedTime = parseEpoch(v)
		parseSuccess = true
	}

	if !parseSuccess {
		return DateResult{Value: "", Normalized: false}
	}

	// Format output according to targetFormat specification
	if targetFormat == "date" {
		formatted := parsedTime.Format("2006-01-02")
		return DateResult{Value: formatted, Normalized: true}
	}

	// Default to RFC3339 for date-time
	formatted := parsedTime.Format(time.RFC3339)
	return DateResult{Value: formatted, Normalized: true}
}

func parseEpoch(epoch int64) time.Time {
	// Detect if epoch is in milliseconds (greater than year 2100 in seconds: 4102444800)
	if epoch > 4102444800 {
		return time.UnixMilli(epoch).UTC()
	}
	return time.Unix(epoch, 0).UTC()
}
