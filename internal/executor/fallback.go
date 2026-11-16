package executor

import (
	"time"
)

// SemanticReceipt is returned to the AI agent runtime when a tool execution fails.
type SemanticReceipt struct {
	Status        string         `json:"status"` // "healed_fallback"
	ToolName      string         `json:"tool_name"`
	ExecutionMode string         `json:"execution_mode"`
	Timestamp     string         `json:"timestamp"`
	Reason        string         `json:"reason"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	Arguments     map[string]any `json:"arguments_attempted"`
	Guidance      string         `json:"agent_guidance"`
}

// GenerateFallbackReceipt creates a structured semantic receipt for unhandled tool execution failures.
func GenerateFallbackReceipt(toolName string, args map[string]any, err error, reason string) SemanticReceipt {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return SemanticReceipt{
		Status:        "healed_fallback",
		ToolName:      toolName,
		ExecutionMode: "semantic_fallback",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Reason:        reason,
		ErrorMessage:  errMsg,
		Arguments:     args,
		Guidance:      "The underlying tool endpoint is temporarily unreachable. You may proceed with alternative logic or present a graceful acknowledgment to the user.",
	}
}
