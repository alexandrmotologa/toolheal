# toolheal Integration Guide

This guide details how to integrate `toolheal` into autonomous agent loops, including LangGraph, Claude Code, CrewAI, and direct HTTP/JSON-RPC pipelines.

## 1. LangGraph Integration (Python)

Route tool execution calls through the `toolheal` interception endpoint. When the LLM generates a tool call, send the payload to `toolheal` before dispatching to the target handler:

```python
import requests
from langchain_core.tools import tool

TOOLHEAL_URL = "http://127.0.0.1:7171/v1/tools/execute"

def execute_with_healing(tool_name: str, arguments: dict):
    response = requests.post(
        TOOLHEAL_URL,
        json={
            "tool": tool_name,
            "arguments": arguments
        },
        timeout=10
    )
    response.raise_for_status()
    payload = response.json()
    
    # payload["result"] contains the sanitized execution output
    # payload["diffs"] contains records of any repaired parameter names or types
    return payload["result"]

# Example in LangGraph tool node
def safe_tool_node(state):
    messages = state["messages"]
    last_message = messages[-1]
    
    outputs = []
    for tool_call in last_message.tool_calls:
        result = execute_with_healing(tool_call["name"], tool_call["args"])
        outputs.append(result)
        
    return {"messages": outputs}
```

## 2. Model Context Protocol (MCP) Integration

`toolheal` speaks JSON-RPC 2.0 and supports MCP `tools/list` and `tools/call`. Configure your agent client to proxy through `toolheal`:

### Claude Desktop / Agent MCP Configuration
In your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "toolheal": {
      "command": "toolheal",
      "args": ["serve", "--port", "7171"]
    }
  }
}
```

To invoke tools through the HTTP JSON-RPC endpoint:
```bash
curl -X POST http://127.0.0.1:7171/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "create_user",
      "arguments": {
        "user_id": "usr_991",
        "email": "dev@corp.local",
        "role": "admin"
      }
    }
  }'
```

`toolheal` reconciles `user_id` to `userId`, `email` to `userEmail`, and snaps `admin` to uppercase `ADMIN`, returning a standard MCP content block.

## 3. CrewAI Tool Execution

Wrap standard CrewAI tools with a lightweight decorator that points to `toolheal`:

```python
import json
import requests
from crewai.tools import BaseTool

class SafeCreateUserTool(BaseTool):
    name: str = "create_user"
    description: str = "Create a new user account with role permissions."

    def _run(self, **kwargs) -> str:
        resp = requests.post(
            "http://127.0.0.1:7171/v1/tools/execute",
            json={"tool": "create_user", "arguments": kwargs}
        )
        data = resp.json()
        return json.dumps(data.get("result", {}))
```

## 4. Direct CLI Pipe Mode (Shell / Bash Scripts)

You can heal tool call payloads directly from the command line without running a background server:

```bash
echo '{"tool":"create_user","arguments":{"user_id":"u1","email":"a@b.c","role":"admin"}}' \
  | toolheal heal --schemas ./examples/tool_definitions.json
```

Output:
```json
{
  "tool_name": "create_user",
  "healed_arguments": {
    "isActive": true,
    "role": "ADMIN",
    "userEmail": "a@b.c",
    "userId": "u1"
  },
  "diffs": [
    {
      "original_key": "user_id",
      "healed_key": "userId",
      "fix_type": "KEY_ALIAS",
      "confidence": 0.98
    },
    {
      "original_key": "email",
      "healed_key": "userEmail",
      "fix_type": "KEY_ALIAS",
      "confidence": 0.98
    },
    {
      "original_key": "role",
      "healed_key": "role",
      "fix_type": "ENUM_SNAP",
      "confidence": 1.0
    }
  ],
  "healed": true,
  "valid": true,
  "latency_us": 12
}
```
