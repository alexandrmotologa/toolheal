package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// PropertySchema defines JSON schema validation and healing metadata for a single parameter.
type PropertySchema struct {
	Type        string                    `json:"type" yaml:"type"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Enum        []string                  `json:"enum,omitempty" yaml:"enum,omitempty"`
	Format      string                    `json:"format,omitempty" yaml:"format,omitempty"`
	Default     any                       `json:"default,omitempty" yaml:"default,omitempty"`
	Items       *PropertySchema           `json:"items,omitempty" yaml:"items,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty" yaml:"required,omitempty"`
	Aliases     []string                  `json:"aliases,omitempty" yaml:"aliases,omitempty"`
}

// ToolParameters defines the root parameter schema for a tool.
type ToolParameters struct {
	Type       string                    `json:"type" yaml:"type"`
	Properties map[string]PropertySchema `json:"properties" yaml:"properties"`
	Required   []string                  `json:"required,omitempty" yaml:"required,omitempty"`
}

// ExecutionTarget configures how tool calls are dispatched if toolheal executes them.
type ExecutionTarget struct {
	Type     string            `json:"type" yaml:"type"` // "mock", "http", "cli", "passthrough"
	Endpoint string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Method   string            `json:"method,omitempty" yaml:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Command  []string          `json:"command,omitempty" yaml:"command,omitempty"`
	TimeoutMs int              `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`
}

// ToolDefinition represents a registered tool with its parameter schema and target.
type ToolDefinition struct {
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	Parameters  ToolParameters  `json:"parameters" yaml:"parameters"`
	Target      ExecutionTarget `json:"target,omitempty" yaml:"target,omitempty"`
}

// MCPToolDefinition represents the JSON structure from Model Context Protocol tools/list.
type MCPToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema ToolParameters `json:"inputSchema"`
}

// Catalog is a thread-safe registry of canonical tool schemas.
type Catalog struct {
	mu    sync.RWMutex
	tools map[string]*ToolDefinition
}

// NewCatalog creates an empty tool catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		tools: make(map[string]*ToolDefinition),
	}
}

// Register adds or updates a tool definition in the catalog.
func (c *Catalog) Register(tool *ToolDefinition) {
	if tool == nil || tool.Name == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tools[tool.Name] = tool
}

// Get retrieves a tool definition by name.
func (c *Catalog) Get(name string) (*ToolDefinition, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tool, exists := c.tools[name]
	return tool, exists
}

// List returns a slice of all registered tool definitions.
func (c *Catalog) List() []*ToolDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*ToolDefinition, 0, len(c.tools))
	for _, tool := range c.tools {
		out = append(out, tool)
	}
	return out
}

// LoadFromJSON parses a JSON array or envelope of tool definitions into the catalog.
func (c *Catalog) LoadFromJSON(data []byte) error {
	// Try standard tool definition slice
	var tools []*ToolDefinition
	if err := json.Unmarshal(data, &tools); err == nil && len(tools) > 0 && tools[0].Name != "" {
		for _, t := range tools {
			c.Register(t)
		}
		return nil
	}

	// Try MCP tools envelope {"tools": [...]}
	var mcpEnvelope struct {
		Tools []MCPToolDefinition `json:"tools"`
	}
	if err := json.Unmarshal(data, &mcpEnvelope); err == nil && len(mcpEnvelope.Tools) > 0 {
		for _, mt := range mcpEnvelope.Tools {
			c.Register(&ToolDefinition{
				Name:        mt.Name,
				Description: mt.Description,
				Parameters:  mt.InputSchema,
				Target: ExecutionTarget{
					Type: "passthrough",
				},
			})
		}
		return nil
	}

	// Try single tool definition
	var single ToolDefinition
	if err := json.Unmarshal(data, &single); err == nil && single.Name != "" {
		c.Register(&single)
		return nil
	}

	return fmt.Errorf("unrecognized tool definition JSON schema format")
}

// LoadFromFile loads tool definitions from a JSON file path.
func (c *Catalog) LoadFromFile(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tool definitions file: %w", err)
	}
	return c.LoadFromJSON(bytes)
}
