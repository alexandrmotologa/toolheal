package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// OpenAPIDoc represents an OpenAPI 3.0/3.1 document structure.
type OpenAPIDoc struct {
	OpenAPI string                                  `json:"openapi" yaml:"openapi"`
	Swagger string                                  `json:"swagger" yaml:"swagger"`
	Info    OpenAPIInfo                             `json:"info" yaml:"info"`
	Servers []OpenAPIServer                         `json:"servers" yaml:"servers"`
	Paths   map[string]map[string]OpenAPIOperation  `json:"paths" yaml:"paths"`
}

type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

type OpenAPIServer struct {
	URL string `json:"url" yaml:"url"`
}

type OpenAPIOperation struct {
	OperationID string             `json:"operationId" yaml:"operationId"`
	Summary     string             `json:"summary" yaml:"summary"`
	Description string             `json:"description" yaml:"description"`
	Parameters  []OpenAPIParam     `json:"parameters" yaml:"parameters"`
	RequestBody *OpenAPIReqBody    `json:"requestBody" yaml:"requestBody"`
}

type OpenAPIParam struct {
	Name        string          `json:"name" yaml:"name"`
	In          string          `json:"in" yaml:"in"` // "query", "path", "header"
	Required    bool            `json:"required" yaml:"required"`
	Description string          `json:"description" yaml:"description"`
	Schema      *PropertySchema `json:"schema" yaml:"schema"`
}

type OpenAPIReqBody struct {
	Required bool                          `json:"required" yaml:"required"`
	Content  map[string]OpenAPIMediaObject `json:"content" yaml:"content"`
}

type OpenAPIMediaObject struct {
	Schema *struct {
		Type       string                    `json:"type" yaml:"type"`
		Properties map[string]PropertySchema `json:"properties" yaml:"properties"`
		Required   []string                  `json:"required" yaml:"required"`
	} `json:"schema" yaml:"schema"`
}

// LoadFromOpenAPI parses an OpenAPI 3.0/3.1 JSON or YAML specification and registers tools.
func (c *Catalog) LoadFromOpenAPI(data []byte) (int, error) {
	var doc OpenAPIDoc

	// Try JSON first, fallback to YAML
	if err := json.Unmarshal(data, &doc); err != nil {
		if yamlErr := yaml.Unmarshal(data, &doc); yamlErr != nil {
			return 0, fmt.Errorf("failed to parse document as OpenAPI JSON or YAML: %v", err)
		}
	}

	if len(doc.Paths) == 0 {
		return 0, fmt.Errorf("no paths found in OpenAPI document")
	}

	baseURL := ""
	if len(doc.Servers) > 0 {
		baseURL = doc.Servers[0].URL
	}

	importedCount := 0

	for path, methods := range doc.Paths {
		for method, op := range methods {
			lowerMethod := strings.ToLower(method)
			if lowerMethod != "get" && lowerMethod != "post" && lowerMethod != "put" &&
				lowerMethod != "delete" && lowerMethod != "patch" {
				continue
			}

			// Generate canonical tool name
			toolName := op.OperationID
			if toolName == "" {
				toolName = fmt.Sprintf("%s_%s", lowerMethod, sanitizePath(path))
			}

			desc := op.Summary
			if desc == "" {
				desc = op.Description
			}
			if desc == "" {
				desc = fmt.Sprintf("%s %s", strings.ToUpper(lowerMethod), path)
			}

			props := make(map[string]PropertySchema)
			required := make([]string, 0)

			// 1. Ingest query & path parameters
			for _, p := range op.Parameters {
				prop := PropertySchema{
					Type:        "string",
					Description: p.Description,
				}
				if p.Schema != nil {
					prop.Type = p.Schema.Type
					prop.Format = p.Schema.Format
					prop.Enum = p.Schema.Enum
					prop.Default = p.Schema.Default
				}
				props[p.Name] = prop
				if p.Required {
					required = append(required, p.Name)
				}
			}

			// 2. Ingest requestBody JSON properties
			if op.RequestBody != nil {
				for _, media := range op.RequestBody.Content {
					if media.Schema != nil && len(media.Schema.Properties) > 0 {
						for k, v := range media.Schema.Properties {
							props[k] = v
						}
						required = append(required, media.Schema.Required...)
					}
				}
			}

			fullEndpoint := path
			if baseURL != "" {
				fullEndpoint = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
			}

			tool := &ToolDefinition{
				Name:        toolName,
				Description: desc,
				Parameters: ToolParameters{
					Type:       "object",
					Properties: props,
					Required:   required,
				},
				Target: ExecutionTarget{
					Type:     "http",
					Endpoint: fullEndpoint,
					Method:   strings.ToUpper(lowerMethod),
				},
			}

			c.Register(tool)
			importedCount++
		}
	}

	return importedCount, nil
}

// FetchAndLoadOpenAPI downloads an OpenAPI specification from a URL and registers its tools.
func (c *Catalog) FetchAndLoadOpenAPI(url string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch OpenAPI URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("remote server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read OpenAPI response body: %w", err)
	}

	return c.LoadFromOpenAPI(body)
}

func sanitizePath(p string) string {
	p = strings.Trim(p, "/")
	p = strings.ReplaceAll(p, "/", "_")
	p = strings.ReplaceAll(p, "{", "")
	p = strings.ReplaceAll(p, "}", "")
	p = strings.ReplaceAll(p, "-", "_")
	return p
}
