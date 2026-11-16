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

func TestLoadFromOpenAPI(t *testing.T) {
	openAPISample := []byte(`{
		"openapi": "3.0.3",
		"info": { "title": "Store API", "version": "1.0" },
		"servers": [{ "url": "https://api.store.com/v1" }],
		"paths": {
			"/orders/{orderId}": {
				"get": {
					"operationId": "getOrder",
					"summary": "Fetch an existing order",
					"parameters": [
						{
							"name": "orderId",
							"in": "path",
							"required": true,
							"schema": { "type": "string" }
						}
					]
				},
				"post": {
					"operationId": "createOrder",
					"summary": "Create a new purchase order",
					"requestBody": {
						"required": true,
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"required": ["customerId", "totalAmount"],
									"properties": {
										"customerId": { "type": "string" },
										"totalAmount": { "type": "number" },
										"currency": { "type": "string", "enum": ["USD", "EUR"] }
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	catalog := NewCatalog()
	count, err := catalog.LoadFromOpenAPI(openAPISample)
	if err != nil {
		t.Fatalf("LoadFromOpenAPI failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 tools imported, got %d", count)
	}

	orderTool, exists := catalog.Get("createOrder")
	if !exists {
		t.Fatalf("expected createOrder to be registered")
	}
	if len(orderTool.Parameters.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(orderTool.Parameters.Required))
	}
	if orderTool.Target.Endpoint != "https://api.store.com/v1/orders/{orderId}" {
		t.Errorf("expected target endpoint with server prefix, got %s", orderTool.Target.Endpoint)
	}
}
