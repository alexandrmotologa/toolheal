package middleware

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/alexandrmotologa/toolheal/internal/config"
	"github.com/alexandrmotologa/toolheal/internal/reconciler"
)

// MCPWrapBridge wraps an upstream MCP server running via stdio and transparently heals tool calls.
type MCPWrapBridge struct {
	catalog *config.Catalog
	engine  *reconciler.Engine
}

// NewMCPWrapBridge creates an MCP stdio interception bridge.
func NewMCPWrapBridge(catalog *config.Catalog, engine *reconciler.Engine) *MCPWrapBridge {
	return &MCPWrapBridge{
		catalog: catalog,
		engine:  engine,
	}
}

// Run executes the child MCP command, intercepting stdin and stdout streams.
func (b *MCPWrapBridge) Run(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr

	childIn, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open child stdin pipe: %w", err)
	}

	childOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open child stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start child MCP process: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Stream 1: Parent os.Stdin -> Child childIn (Intercept tools/call)
	go func() {
		defer wg.Done()
		defer childIn.Close()

		scanner := bufio.NewScanner(os.Stdin)
		// Support large MCP JSON payloads (up to 16MB)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 16*1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			healedLine := b.interceptParentRequest(line)
			_, _ = childIn.Write(healedLine)
			_, _ = childIn.Write([]byte("\n"))
		}
	}()

	// Stream 2: Child childOut -> Parent os.Stdout (Intercept tools/list)
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(childOut)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 16*1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			b.interceptChildResponse(line)
			_, _ = os.Stdout.Write(line)
			_, _ = os.Stdout.Write([]byte("\n"))
		}
	}()

	wg.Wait()
	return cmd.Wait()
}

// interceptParentRequest intercepts incoming JSON-RPC calls, healing arguments on tools/call.
func (b *MCPWrapBridge) interceptParentRequest(line []byte) []byte {
	var req JSONRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return line
	}

	if req.Method != "tools/call" {
		return line
	}

	toolName, args, err := ParseMCPToolCall(req.Params)
	if err != nil {
		return line
	}

	toolDef, _ := b.catalog.Get(toolName)
	report := b.engine.Reconcile(toolName, args, toolDef)

	if !report.Healed {
		return line
	}

	// Re-encode modified arguments back into JSON-RPC payload
	newParams := map[string]any{
		"name":      toolName,
		"arguments": report.HealedArguments,
	}
	paramsJSON, err := json.Marshal(newParams)
	if err != nil {
		return line
	}

	req.Params = paramsJSON
	newReqBytes, err := json.Marshal(req)
	if err != nil {
		return line
	}

	return newReqBytes
}

// interceptChildResponse captures schema catalog from tools/list responses.
func (b *MCPWrapBridge) interceptChildResponse(line []byte) {
	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
		Result  struct {
			Tools []config.MCPToolDefinition `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal(line, &resp); err == nil && len(resp.Result.Tools) > 0 {
		for _, mt := range resp.Result.Tools {
			b.catalog.Register(&config.ToolDefinition{
				Name:        mt.Name,
				Description: mt.Description,
				Parameters:  mt.InputSchema,
				Target: config.ExecutionTarget{
					Type: "passthrough",
				},
			})
		}
	}
}

// WriteSafeStderr is a helper to log diagnostic messages without corrupting stdio JSON-RPC.
func WriteSafeStderr(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[toolheal-mcp] "+format+"\n", a...)
}
