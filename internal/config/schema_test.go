package config

import (
	"testing"
)

func TestCatalogLoadFromJSON(t *testing.T) {
	catalog := NewCatalog()
	err := catalog.LoadFromFile("../../examples/tool_definitions.json")
	if err != nil {
		t.Fatalf("failed to load tool definitions: %v", err)
	}

	tools := catalog.List()
	if len(tools) < 5 {
		t.Fatalf("expected at least 5 tools, got %d", len(tools))
	}

	userTool, exists := catalog.Get("create_user")
	if !exists {
		t.Fatalf("expected create_user to exist in catalog")
	}

	if len(userTool.Parameters.Required) != 2 {
		t.Fatalf("expected 2 required parameters, got %d", len(userTool.Parameters.Required))
	}

	prop, hasEmail := userTool.Parameters.Properties["userEmail"]
	if !hasEmail {
		t.Fatalf("expected userEmail property")
	}
	if prop.Format != "email" {
		t.Errorf("expected email format, got %s", prop.Format)
	}
}
