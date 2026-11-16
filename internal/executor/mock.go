package executor

import (
	"fmt"
	"time"
)

// MockServer provides deterministic in-memory execution for tools configured with target: mock.
type MockServer struct{}

// NewMockServer creates an instance of the mock tool execution server.
func NewMockServer() *MockServer {
	return &MockServer{}
}

// Execute handles mock execution for known demo tools or generic fallback.
func (m *MockServer) Execute(toolName string, args map[string]any) (map[string]any, error) {
	switch toolName {
	case "create_user":
		return map[string]any{
			"status":       "created",
			"user_id":      args["userId"],
			"email":        args["userEmail"],
			"role":         args["role"],
			"is_active":    args["isActive"],
			"created_date": args["createdDate"],
			"tags":         args["tags"],
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		}, nil

	case "process_payment":
		amountCents, _ := args["amountCents"].(int)
		return map[string]any{
			"transaction_id": fmt.Sprintf("txn_%d", time.Now().UnixNano()%1000000),
			"customer_id":    args["customerId"],
			"amount_cents":   amountCents,
			"currency":       args["currency"],
			"status":         "succeeded",
			"captured":       args["captureNow"],
			"descriptor":     args["statementDescriptor"],
		}, nil

	case "search_database":
		pageSize := 20
		if p, ok := args["pageSize"].(int); ok && p > 0 {
			pageSize = p
		}
		items := make([]map[string]any, 0, min(pageSize, 3))
		for i := 1; i <= min(pageSize, 3); i++ {
			items = append(items, map[string]any{
				"id":     fmt.Sprintf("rec_%d", i),
				"title":  fmt.Sprintf("Match %d for %v", i, args["query"]),
				"status": args["filterStatus"],
			})
		}
		return map[string]any{
			"query":        args["query"],
			"page_number":  args["pageNumber"],
			"page_size":    pageSize,
			"total_hits":   42,
			"records":      items,
			"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		}, nil

	case "create_github_issue":
		return map[string]any{
			"issue_number": 1084,
			"repository":   args["repository"],
			"title":        args["issueTitle"],
			"state":        "open",
			"labels":       args["labels"],
			"assignees":    args["assignees"],
			"priority":     args["priority"],
			"url":          fmt.Sprintf("https://github.com/%v/issues/1084", args["repository"]),
		}, nil

	case "send_slack_notification":
		return map[string]any{
			"ok":        true,
			"channel":   args["channelId"],
			"ts":        fmt.Sprintf("%d.000100", time.Now().Unix()),
			"message":   args["messageText"],
			"is_urgent": args["isUrgent"],
		}, nil

	default:
		return map[string]any{
			"tool":         toolName,
			"status":       "executed",
			"echo_args":    args,
			"timestamp_us": time.Now().UnixMicro(),
		}, nil
	}
}
