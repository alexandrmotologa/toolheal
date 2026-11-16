# toolheal Model Context Protocol (MCP) Specification

`toolheal` implements the Model Context Protocol (MCP) specification for tool discovery and execution. It acts as an intelligent intermediary proxy that normalizes LLM function calls before they reach upstream MCP servers.

## Supported MCP Methods

### 1. `tools/list`
Returns the active catalog of registered tools and their canonical input schemas.

#### Request
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list"
}
```

#### Response
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "create_user",
        "description": "Provision a new user account with role permissions.",
        "inputSchema": {
          "type": "object",
          "required": ["userId", "userEmail"],
          "properties": {
            "userId": { "type": "string", "description": "Unique user identifier" },
            "userEmail": { "type": "string", "format": "email" },
            "role": { "type": "string", "enum": ["ADMIN", "MEMBER", "GUEST", "READONLY"] }
          }
        }
      }
    ]
  }
}
```

### 2. `tools/call`
Executes a specific tool after passing arguments through the self-healing reconciliation pipeline.

#### Request with Hallucinated Parameters
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "create_user",
    "arguments": {
      "user_id": "usr_7721",
      "email": "sarah@acme.org",
      "role": "admin",
      "is_active": "true"
    }
  }
}
```

#### Reconciled Execution Response
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\n  \"created_date\": \"2026-11-16T14:00:00Z\",\n  \"email\": \"sarah@acme.org\",\n  \"is_active\": true,\n  \"role\": \"ADMIN\",\n  \"status\": \"created\",\n  \"user_id\": \"usr_7721\"\n}"
      }
    ],
    "isError": false
  }
}
```

## Error Codes

When a tool call cannot be repaired (e.g. required parameters are completely absent), `toolheal` emits standard JSON-RPC 2.0 error payloads:

| Code | Message | Description |
|---|---|---|
| `-32700` | Parse error | Invalid JSON received by the server |
| `-32600` | Invalid Request | Payload does not adhere to JSON-RPC 2.0 structure |
| `-32601` | Method not found | Method other than `tools/list` or `tools/call` requested |
| `-32602` | Invalid params | Missing required parameters even after healing pass |
| `-32603` | Internal error | Unhandled internal exception during execution |
