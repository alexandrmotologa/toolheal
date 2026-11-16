<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="toolheal Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">toolheal</h1>

<p align="center">
  <b>Autonomous Self-Healing Tool Calling and Schema Reconciliation Middleware for AI Agents</b>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/react-19-61DAFB?style=flat&logo=react" alt="React 19" />
  <img src="https://img.shields.io/badge/sqlite-pure--go-003B57?style=flat&logo=sqlite" alt="SQLite" />
  <img src="https://img.shields.io/badge/protocol-mcp-blue?style=flat" alt="Model Context Protocol" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat" alt="License" />
</p>

---

`toolheal` is a lightweight Go execution middleware (< 15MB RAM) with an embedded React 19 management console. It intercepts tool calls from autonomous agents (Claude Code, LangGraph, Cursor, CrewAI), repairs parameter naming discrepancies, coerces mismatched primitive types, snaps drifting enum values, and normalizes date strings against canonical OpenAPI and Model Context Protocol schemas in under 1 millisecond.

<p align="center">
  <img src="docs/images/toolheal_demo.gif?raw=true" alt="toolheal Live Execution Demo" width="100%" />
</p>

## Problem: The 400 Bad Request Re-Prompting Loop

When language models invoke tools in multi-step workflows, they frequently produce small formatting discrepancies:
1. Parameter naming collisions: passing `user_id` when the schema requires `userId`, or `limit` instead of `pageSize`.
2. Type casting errors: passing numbers or booleans encoded as strings (`{"amountCents": "5000"}` or `{"isActive": "true"}`).
3. Enum value drift: passing `"urgent"` or `"running"` when the schema accepts `["HIGH", "CRITICAL"]` or `["ACTIVE"]`.
4. Date format variations: outputting `15/11/2026` or `today` when endpoints expect strict RFC3339 timestamps.

Standard agent loops pass the raw 400 stack trace back to the LLM to re-generate arguments. Each retry loop burns 1,500 to 4,000 context tokens, adds 3 to 6 seconds of latency, and risks cascading execution failures.

`toolheal` resolves these discrepancies deterministically before the target tool executes, achieving over 85% automatic repair without triggering a single additional LLM call.

---

## Architecture and Core Pipeline

```
       +-------------------------------------------------------------+
       |               AI Agent Runtime / Harness                    |
       |         (Claude Code, LangGraph, AutoGen, CrewAI)           |
       +-------------------------------------------------------------+
                                      |
                     Tool Invocation: { name, arguments }
                                      v
       +-------------------------------------------------------------+
       |                   toolheal Interception Shim                |
       |                   (High-Performance Go CLI)                |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | Tool Schema Catalog |------->| Parameter Reconciler   |  |
       |  | (OpenAPI / MCP)     |        | & Fuzzy Key Matcher    |  |
       |  +---------------------+        +------------------------+  |
       |                                              |              |
       |                                              v              |
       |  +-------------------------------------------------------+  |
       |  |          Type Coercer & Enum Nearest-Neighbor         |  |
       |  |   - "100" -> 100, "true" -> true                      |  |
       |  |   - "urgent" -> "HIGH" (Enum Snapping)                |  |
       |  |   - Date string -> RFC3339 / ISO-8601                 |  |
       |  |   - Fast-Path O(1) Memory Cache                       |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |          Safe Tool Executor & Circuit Breaker         |  |
       |  |   - Dispatches Cleaned Arguments to Target Tool       |  |
       |  |   - Exponential Backoff on Transient Network Errors   |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |             Sanitized & Successful Tool Response            |
       |  1. Deterministic Output Returned to Agent Runtime          |
       |  2. Async Telemetry (Tokens Saved, Fix Diffs) to SQLite     |
       |  3. Embedded React 19 Tool Console (Local HTTP :7171)       |
       +-------------------------------------------------------------+
```

---

## Live Management Console

`toolheal` embeds a React 19 web interface directly into the single compiled binary via `//go:embed`.

### Real-Time Telemetry Dashboard
Displays auto-recovery success rates, prompt tokens saved, financial ROI calculation based on avoided re-prompt loops, and average execution latency.

<p align="center">
  <img src="docs/images/dashboard_desktop.png?raw=true" alt="toolheal Live Dashboard" width="100%" />
</p>

### Interactive Parameter Healing Sandbox
Test raw or hallucinated JSON payloads against registered schemas. View side-by-side execution output, sub-millisecond timing, and color-coded diff badges.

<p align="center">
  <img src="docs/images/sandbox_healed.png?raw=true" alt="toolheal Interactive Sandbox" width="100%" />
</p>

### Live Interception Stream and Fix Inspection
Inspect every intercepted tool call in real time via Server-Sent Events, complete with confidence scores and audit trail export in CSV and JSON formats.

<p align="center">
  <img src="docs/images/invocations_stream.png?raw=true" alt="toolheal Invocations Stream" width="100%" />
</p>

### Responsive Mobile Interface
Full layout adaptability across desktop, tablet, and mobile displays (tested down to 390px viewport width).

<p align="center">
  <img src="docs/images/mobile_responsive.png?raw=true" alt="toolheal Mobile Responsive UI" width="340" />
</p>

---

## Benchmark Results

Evaluated against 60 real-world parameter failure scenarios from production agent loops:

| Metric | Result | Target / SLA | Status |
| :--- | :--- | :--- | :--- |
| **Auto-Recovery Rate** | **100.00%** | > 85.0% | PASSED |
| **Average Processing Latency** | **0.008 ms (8 µs)** | < 2.0 ms | PASSED |
| **Prompt Tokens Saved** | **144,000 tokens** | > 50,000 | PASSED |
| **Circuit Breaker Trip Time** | **< 1 ms** | < 10 ms | PASSED |
| **Binary Memory Footprint** | **< 15 MB RAM** | < 30 MB | PASSED |

---

## Quick Start

### 1. Build Single Binary

```bash
git clone https://github.com/alexandrmotologa/toolheal.git
cd toolheal
go build -o toolheal.exe ./cmd/toolheal
```

### 2. Start Middleware Daemon and Console

```bash
./toolheal.exe serve --port 7171
```

- Web Console: `http://localhost:7171`
- Tool Execution Endpoint: `POST http://localhost:7171/v1/tools/execute`
- Dry-Run Reconciliation: `POST http://localhost:7171/v1/heal`
- Model Context Protocol Endpoint: `POST http://localhost:7171/rpc`

### 3. Command Line Pipe Mode

Heal tool arguments via standard input in scripts and terminal pipelines:

```bash
echo '{"tool":"create_user","arguments":{"user_id":"u_001","email":"alex@corp.com","role":"admin","active":"true"}}' | ./toolheal.exe heal
```

### 4. Transparent MCP Stdio Wrapper

Wrap any existing stdio MCP server for Claude Code or Cursor with automatic parameter healing:

```bash
toolheal mcp-wrap npx -y @modelcontextprotocol/server-github
```

### 5. Run the Verification Benchmark

Execute the automated benchmark suite against test scenarios:

```bash
./toolheal.exe bench --file examples/failing_invocations.jsonl
```

---

## Documentation

- [Architecture Specification](docs/architecture.md): Deep dive into Damerau-Levenshtein calculations, recursive nested object healing, and circuit breaker states.
- [Agent Integration Guide](docs/integration_guide.md): Code examples for LangGraph, Claude Code, CrewAI, and custom loops.
- [Model Context Protocol Specification](docs/mcp_spec.md): JSON-RPC 2.0 framing and error mapping specifications.

---

## License

MIT License. Developed by Alexandr Motologa.
