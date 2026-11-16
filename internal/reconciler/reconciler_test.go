package reconciler

import (
	"testing"

	"github.com/alexandrmotologa/toolheal/internal/config"
)

func TestFoldKeyAndCasing(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"user_id", "userid"},
		{"userId", "userid"},
		{"user-id", "userid"},
		{"User_Id", "userid"},
		{"page.number", "pagenumber"},
	}

	for _, tc := range cases {
		folded := FoldKey(tc.input)
		if folded != tc.expected {
			t.Errorf("FoldKey(%q) = %q; expected %q", tc.input, folded, tc.expected)
		}
	}

	if ToCamelCase("user_id") != "userId" {
		t.Errorf("ToCamelCase(user_id) expected userId, got %s", ToCamelCase("user_id"))
	}
	if ToSnakeCase("userId") != "user_id" {
		t.Errorf("ToSnakeCase(userId) expected user_id, got %s", ToSnakeCase("userId"))
	}
}

func TestDamerauLevenshteinAndFuzzy(t *testing.T) {
	// Transposition
	dist := DamerauLevenshtein("usrId", "userId")
	if dist > 2 {
		t.Errorf("DamerauLevenshtein(usrId, userId) expected <= 2, got %d", dist)
	}

	candidates := []string{"userId", "userEmail", "role", "isActive", "createdDate"}
	match := FindBestFuzzyMatch("user_id", candidates, 0.80, 2)
	if !match.Matched || match.CanonicalKey != "userId" {
		t.Fatalf("expected userId match, got %+v", match)
	}

	matchTypo := FindBestFuzzyMatch("usreId", candidates, 0.80, 2)
	if !matchTypo.Matched || matchTypo.CanonicalKey != "userId" {
		t.Fatalf("expected userId match for typo usreId, got %+v", matchTypo)
	}
}

func TestTypeCoercion(t *testing.T) {
	// String to int
	intProp := config.PropertySchema{Type: "integer"}
	res1 := CoerceValue("42", intProp)
	if res1.Value != 42 || !res1.Changed {
		t.Errorf("expected 42, got %v (%T)", res1.Value, res1.Value)
	}

	// String to bool
	boolProp := config.PropertySchema{Type: "boolean"}
	res2 := CoerceValue("true", boolProp)
	if res2.Value != true || !res2.Changed {
		t.Errorf("expected true, got %v", res2.Value)
	}

	// Scalar to array
	arrProp := config.PropertySchema{Type: "array"}
	res3 := CoerceValue("single_tag", arrProp)
	slice, ok := res3.Value.([]any)
	if !ok || len(slice) != 1 || slice[0] != "single_tag" {
		t.Errorf("expected []any{'single_tag'}, got %v", res3.Value)
	}

	// Comma separated string to array
	res4 := CoerceValue("tag1, tag2, tag3", arrProp)
	slice4, ok4 := res4.Value.([]any)
	if !ok4 || len(slice4) != 3 {
		t.Errorf("expected 3 items in slice, got %v", res4.Value)
	}
}

func TestEnumSnapping(t *testing.T) {
	allowed := []string{"ADMIN", "MEMBER", "GUEST", "READONLY"}

	// Case folding
	res1 := SnapEnum("admin", allowed, 0.70)
	if !res1.Snapped || res1.Value != "ADMIN" {
		t.Errorf("expected ADMIN, got %+v", res1)
	}

	// Synonym
	res2 := SnapEnum("user", allowed, 0.70)
	if !res2.Snapped || res2.Value != "MEMBER" {
		t.Errorf("expected MEMBER, got %+v", res2)
	}

	// Priority enum
	priorityEnums := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}
	res3 := SnapEnum("urgent", priorityEnums, 0.70)
	if !res3.Snapped || res3.Value != "HIGH" {
		t.Errorf("expected HIGH, got %+v", res3)
	}
}

func TestDateNormalization(t *testing.T) {
	// DD/MM/YYYY to RFC3339
	res1 := NormalizeDate("15/11/2026", "date-time")
	if !res1.Normalized || res1.Value != "2026-11-15T00:00:00Z" {
		t.Errorf("expected 2026-11-15T00:00:00Z, got %+v", res1)
	}

	// Date only target format
	res2 := NormalizeDate("2026/11/15", "date")
	if !res2.Normalized || res2.Value != "2026-11-15" {
		t.Errorf("expected 2026-11-15, got %+v", res2)
	}

	// Relative 'today'
	res3 := NormalizeDate("today", "date")
	if !res3.Normalized {
		t.Errorf("expected today to normalize, got %+v", res3)
	}
}

func TestEngineReconciliation(t *testing.T) {
	catalog := config.NewCatalog()
	err := catalog.LoadFromFile("../../examples/tool_definitions.json")
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	userTool, _ := catalog.Get("create_user")
	engine := NewEngine(nil)

	// Simulated hallucinated payload from an LLM:
	// - user_id instead of userId
	// - email instead of userEmail
	// - role as lowercase "admin"
	// - is_active as string "true"
	// - created_at in DD/MM/YYYY format
	// - tags as scalar string "dev"
	hallucinatedInput := map[string]any{
		"user_id":    "usr_9981",
		"email":      "alex@example.com",
		"role":       "admin",
		"is_active":  "true",
		"created_at": "15/11/2026",
		"tags":       "developer",
	}

	report := engine.Reconcile("create_user", hallucinatedInput, userTool)

	if !report.Valid {
		t.Fatalf("expected report to be valid, missing: %v", report.MissingRequired)
	}
	if !report.Healed {
		t.Fatalf("expected report to have healed arguments")
	}

	healed := report.HealedArguments
	if healed["userId"] != "usr_9981" {
		t.Errorf("expected userId=usr_9981, got %v", healed["userId"])
	}
	if healed["userEmail"] != "alex@example.com" {
		t.Errorf("expected userEmail=alex@example.com, got %v", healed["userEmail"])
	}
	if healed["role"] != "ADMIN" {
		t.Errorf("expected role=ADMIN, got %v", healed["role"])
	}
	if healed["isActive"] != true {
		t.Errorf("expected isActive=true, got %v", healed["isActive"])
	}
	if healed["createdDate"] != "2026-11-15T00:00:00Z" {
		t.Errorf("expected createdDate=2026-11-15T00:00:00Z, got %v", healed["createdDate"])
	}

	tags, ok := healed["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "developer" {
		t.Errorf("expected tags wrapped in slice, got %v", healed["tags"])
	}

	// Verify latency is well below 1ms (1000 microseconds)
	if report.LatencyUs > 2000 {
		t.Errorf("latency %d us exceeded 2000 us limit", report.LatencyUs)
	}
}

func TestRecursiveNestedHealingAndFastPathCache(t *testing.T) {
	// Define a tool with nested object and array of objects
	nestedTool := &config.ToolDefinition{
		Name: "complex_query",
		Parameters: config.ToolParameters{
			Type: "object",
			Properties: map[string]config.PropertySchema{
				"filterSettings": {
					Type: "object",
					Properties: map[string]config.PropertySchema{
						"startDate": {Type: "string", Format: "date"},
						"status":    {Type: "string", Enum: []string{"ACTIVE", "ARCHIVED"}},
					},
					Required: []string{"startDate"},
				},
				"userList": {
					Type: "array",
					Items: &config.PropertySchema{
						Type: "object",
						Properties: map[string]config.PropertySchema{
							"userId":   {Type: "string"},
							"isActive": {Type: "boolean"},
						},
						Required: []string{"userId"},
					},
				},
			},
			Required: []string{"filterSettings"},
		},
	}

	engine := NewEngine(nil)

	// Hallucinated payload:
	// - filter_settings instead of filterSettings
	// - start_date instead of startDate with DD/MM/YYYY
	// - status lowercase 'active'
	// - user_list instead of userList
	// - items inside have user_id instead of userId, active string 'true'
	input := map[string]any{
		"filter_settings": map[string]any{
			"start_date": "15/11/2026",
			"status":     "active",
		},
		"user_list": []any{
			map[string]any{
				"user_id": "usr_100",
				"active":  "true",
			},
		},
	}

	// First pass: populates FastPathCache
	report := engine.Reconcile("complex_query", input, nestedTool)
	if !report.Valid {
		t.Fatalf("expected report to be valid, missing: %v", report.MissingRequired)
	}

	filter, ok := report.HealedArguments["filterSettings"].(map[string]any)
	if !ok {
		t.Fatalf("expected filterSettings object, got %T", report.HealedArguments["filterSettings"])
	}
	if filter["startDate"] != "2026-11-15" {
		t.Errorf("expected startDate=2026-11-15, got %v", filter["startDate"])
	}
	if filter["status"] != "ACTIVE" {
		t.Errorf("expected status=ACTIVE, got %v", filter["status"])
	}

	users, ok := report.HealedArguments["userList"].([]any)
	if !ok || len(users) != 1 {
		t.Fatalf("expected userList slice with 1 item, got %v", report.HealedArguments["userList"])
	}
	user0, ok := users[0].(map[string]any)
	if !ok {
		t.Fatalf("expected user object, got %T", users[0])
	}
	if user0["userId"] != "usr_100" {
		t.Errorf("expected userId=usr_100, got %v", user0["userId"])
	}
	if user0["isActive"] != true {
		t.Errorf("expected isActive=true, got %v", user0["isActive"])
	}

	// Verify FastPathCache recorded hits on subsequent run
	report2 := engine.Reconcile("complex_query", input, nestedTool)
	if !report2.Valid {
		t.Fatalf("expected report2 to be valid")
	}

	cached, found := engine.FastCache().Get("complex_query", "filter_settings")
	if !found || cached.Hits < 2 {
		t.Errorf("expected FastPathCache hits >= 2 for filter_settings, got %+v", cached)
	}

	patches := engine.FastCache().ListSuggestedPatches(1)
	if len(patches) < 1 {
		t.Errorf("expected suggested patches, got %d", len(patches))
	}
}
