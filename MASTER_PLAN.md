# toolheal: Architecture and Engineering Master Plan
> Autonomous Self-Healing Tool Calling and Schema Reconciliation Middleware for AI Agents
> Target: High-Performance Single-Binary Go Middleware (< 15MB RAM, < 2ms execution) with Fuzzy Schema Reconciliation and Embedded React 19 Tool Console

---

## 1. Problem Statement and Motivation

### 1.1 The Brittleness of Function Calling in Autonomous AI Agents
Autonomous coding, research, and enterprise agents (such as Claude Code, LangGraph, AutoGen, and custom agentic loops) interact with external systems through tool calls (function calling). In real-world multi-step execution, tool calls suffer from high failure rates caused by subtle model hallucinations and formatting discrepancies:
1. **Hallucinated Parameter Names**: Models frequently swap naming conventions or hallucinate parameter keys (e.g., passing `user_id` when the schema requires `userId`, or `limit` instead of `pageSize`).
2. **Type Casting Collisions**: Models routinely pass numbers or booleans encoded as strings (e.g., `{"count": "10"}` instead of `{"count": 10}`) or wrap scalar values inside single-element arrays, triggering immediate `400 Bad Request` schema rejections.
3. **Enum Value Drift**: Models substitute near-synonyms for strict enum definitions (e.g., passing `"active_user"` or `"running"` when the schema strictly permits `["ACTIVE", "PENDING", "DISABLED"]`).
4. **Date and Timestamp Formatting Inconsistencies**: Models produce unstructured date strings (e.g., `"15/11/2026"` or `"yesterday"`) when the underlying API requires strict ISO-8601 (`"2026-11-15T00:00:00Z"`).
5. **The Expensive Re-Prompting Loop**: When a tool call fails, standard agent loops pass the raw stack trace back to the LLM, prompting it to "fix the arguments." Each retry burns 1,000 to 4,000 context tokens, adds 3 to 6 seconds of latency, and frequently leads the agent into repetitive deadlock loops.

### 1.2 The toolheal Solution
`toolheal` is a lightweight, zero-dependency middleware written in Go that intercepts tool invocations before they are executed. Operating between agent loops and underlying APIs or Model Context Protocol (MCP) servers:
- **Zero-Token Fuzzy Parameter Reconciliation**: Uses Levenshtein distance, case-insensitivity mapping (camelCase <-> snake_case <-> kebab-case), and semantic similarity to automatically map hallucinated parameter names to canonical schema keys in under 1 millisecond.
- **Deterministic Type and Enum Snapping**: Coerces mismatched primitives (string -> integer, scalar -> array) and snaps drifting enum strings to their closest valid schema definition using fast string metrics.
- **Temporal Normalization**: Parses natural and varied date formats, automatically emitting valid ISO-8601 and UNIX epoch timestamps.
- **Resilient Execution and Smart Fallbacks**: Catches transient 429/500 API errors with exponential backoff; if a non-critical tool is permanently unavailable, synthesizes a structured semantic receipt so the agent can proceed without crashing.
- **Embedded React 19 Tool Console**: Displays live parameter healing operations, schema drift heatmaps, and token savings metrics.

---

## 2. High-Level System Architecture

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

## 3. Fuzzy Reconciliation and Parameter Repair Engine

```
+-----------------------------------+-----------------------------------------------------------------+
| Reconciliation Operation          | Technical Action                                                |
+-----------------------------------+-----------------------------------------------------------------+
| 1. Key Case & Delimiter Fold      | Converts keys to lowercase and strips underscores/hyphens       |
|                                   | (user_id, userId, user-id -> userid).                           |
| 2. Levenshtein Fuzzy Alignment    | If key not found, computes edit distance D(kin, k).             |
|                                   | Automatically renames key if D <= 2 or similarity >= 0.80.      |
| 3. Strict Enum Snapping           | Computes Jaro-Winkler similarity between provided value and     |
|                                   | permitted enum set; snaps to closest valid enum variant.        |
| 4. Primitive Type Casting         | Safely unquotes strings into integers/floats/booleans; wraps    |
|                                   | isolated scalar into array if schema defines type: array.       |
| 5. ISO-8601 Date Normalization    | Detects diverse date strings; parses and formats into RFC3339   |
|                                   | (2026-11-15T00:00:00Z) required by backend endpoints.           |
| 6. Missing Optional Null Fill     | Injects default schema values or null for unprovided            |
|                                   | optional arguments expected by strict JSON-RPC handlers.        |
| 7. Structural Envelope Unwrap     | Extracts parameters if wrapped in {"params": ...} or similar.   |
+-----------------------------------+-----------------------------------------------------------------+
```

### 3.1 Mathematical Key Alignment Metric
Given an unknown input parameter key $u$ and a canonical schema property key $k \in \mathcal{K}$:

$$\text{Sim}(u, k) = 1 - \frac{\text{Levenshtein}(u, k)}{\max(|u|, |k|)}$$

The optimal key replacement $k^*$ is selected via:

$$k^* = \arg\max_{k \in \mathcal{K}} \text{Sim}(u, k), \quad \text{subject to } \text{Sim}(u, k^*) \ge 0.80$$

---

## 4. Repository Structure

```
toolheal/
├── cmd/
│   └── toolheal/
│       └── main.go                 # CLI entry point, subcommands: serve, proxy, heal, bench
├── internal/
│   ├── config/
│   │   ├── config.go               # Configuration parser for tool registries and tolerances
│   │   └── schema.go               # Schema catalog definitions and execution timeouts
│   ├── middleware/
│   │   ├── handler.go              # Tool invocation interceptor for MCP and REST backends
│   │   ├── rpc.go                  # JSON-RPC 2.0 parser and serializer
│   │   └── mock_server.go          # Local mock tool server for testing
│   ├── reconciler/
│   │   ├── engine.go               # Master reconciliation coordinator
│   │   ├── fuzzy.go                # Levenshtein and Jaro-Winkler key matcher
│   │   ├── casing.go               # camelCase, snake_case, and kebab-case normalizer
│   │   ├── enums.go                # Enum variant nearest-neighbor resolver
│   │   ├── dates.go                # Natural language and multi-format ISO-8601 parser
│   │   ├── coercion.go             # Primitive type casting and array wrapping
│   │   └── aliases.go              # Standard semantic aliases map (e.g. limit <-> pageSize)
│   ├── executor/
│   │   ├── dispatcher.go           # Tool runner (Local CLI, REST HTTP, MCP stdio, Mock)
│   │   ├── circuit.go              # Circuit breaker with exponential backoff on 429/500
│   │   └── fallback.go             # Semantic receipt generator for unavailable non-critical tools
│   ├── telemetry/
│   │   ├── store.go                # Embedded SQLite store (modernc.org/sqlite) for repair logs
│   │   ├── metrics.go              # Cumulative tokens saved, healing success rate, latency delta
│   │   └── sse_feed.go             # Live broadcast feed for web console
│   └── server/
│       ├── api.go                  # REST API for tool registry inspection, healing, and metrics
│       ├── embed.go                # Go 1.16+ embed filesystem for React 19 console
│       └── console_handler.go      # Web dashboard routes
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── HealingGauge.tsx    # Tool invocation auto-recovery percentage dial
│   │   │   ├── ParameterDiff.tsx   # Visual diff of hallucinated args vs. healed arguments
│   │   │   ├── DriftHeatmap.tsx    # Heatmap of tools and parameters most prone to hallucination
│   │   │   ├── ToolSandbox.tsx     # Interactive playground to test parameter fuzzy healing
│   │   │   ├── LiveInvocationLog.tsx# Real-time stream of intercepted tool calls
│   │   │   └── ToolRegistryView.tsx# Catalog of registered tool schemas
│   │   ├── index.css
│   │   └── main.tsx
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── toolheal.yaml               # Standard configuration file
│   ├── tool_definitions.json       # Sample OpenAPI / MCP tool schemas (Stripe, GitHub, Jira)
│   └── failing_invocations.jsonl   # Real-world agent tool failure payloads for benchmarking
├── docs/
│   ├── architecture.md             # Detailed engineering and algorithm specifications
│   ├── integration_guide.md        # LangGraph, Claude Code, CrewAI integration recipes
│   └── mcp_spec.md                 # MCP interception and proxying protocol details
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Step-by-Step Implementation Roadmap

### Phase 1: Go Module Initialization, Config, and Schema Catalog
- [x] Initialize Git repository and create public GitHub repo (`alexandrmotologa/toolheal`) with description and 13 targeted topics.
- [ ] Initialize Go module (`go.mod`) with modernc.org/sqlite and test dependencies.
- [ ] Implement `internal/config/config.go` and `internal/config/schema.go` supporting OpenAPI 3.0/3.1 and MCP tools schema specifications.
- [ ] Build `examples/tool_definitions.json` with realistic enterprise tools (payment charges, user creation, github issues, analytics query).

### Phase 2: Reconciliation Engine and String Algorithms
- [ ] Implement `casing.go`: fast delimiter folding, camelCase, snake_case, kebab-case, PascalCase, SCREAMING_SNAKE_CASE conversion.
- [ ] Implement `fuzzy.go`: Damerau-Levenshtein distance and Jaro-Winkler similarity algorithms with benchmark sub-millisecond execution.
- [ ] Implement `aliases.go`: built-in semantic dictionary for ubiquitous parameter synonyms (`user_id` <-> `userId`, `limit` <-> `pageSize`, `query` <-> `q`).
- [ ] Implement `coercion.go`: primitive type coercer (string to int/float/bool, array wrapping/unwrapping, stringified JSON unwrapping).
- [ ] Implement `enums.go`: nearest-neighbor enum resolver with case folding and distance ranking.
- [ ] Implement `dates.go`: multi-format date parser supporting ISO-8601, RFC3339, YYYY-MM-DD, DD/MM/YYYY, epoch seconds/millis, and common relative dates.
- [ ] Implement `engine.go`: coordinator tying all healing stages together into a single pipeline with granular diff tracking.
- [ ] Write comprehensive unit tests for all reconciler modules.

### Phase 3: Resilient Tool Executor and Circuit Breaker
- [ ] Implement `internal/executor/circuit.go`: three-state circuit breaker (Closed, Open, Half-Open) with exponential backoff on 429 and 5xx errors.
- [ ] Implement `internal/executor/dispatcher.go`: multi-target tool runner supporting internal mock tool executions, HTTP REST requests, and local commands.
- [ ] Implement `internal/executor/fallback.go`: deterministic semantic receipt synthesizer for gracefully degraded agent executions.
- [ ] Implement `internal/middleware/mock_server.go`: integrated mock server for end-to-end testing without external network dependencies.

### Phase 4: Telemetry, SQLite Storage, and Live Streaming
- [ ] Implement `internal/telemetry/store.go`: pure-Go embedded SQLite database schema for invocations, fixes, and schema drift counts.
- [ ] Implement `internal/telemetry/metrics.go`: cumulative tokens saved, healing success rate percentage, and latency percentiles.
- [ ] Implement `internal/telemetry/sse_feed.go`: high-throughput Server-Sent Events broker publishing live healing events to connected clients.

### Phase 5: Middleware Interceptor and HTTP Server
- [ ] Implement `internal/middleware/handler.go`: HTTP handler intercepting OpenAI/Anthropic/MCP format requests (`POST /v1/tools/execute`, `POST /v1/heal`).
- [ ] Implement `internal/middleware/rpc.go`: JSON-RPC 2.0 message parsing and framing.
- [ ] Implement `internal/server/api.go`: REST endpoints for dashboard metrics (`/api/metrics`), invocation history (`/api/invocations`), schema catalog (`/api/tools`), and live sandbox test (`/api/heal`).
- [ ] Implement `internal/server/embed.go` and `console_handler.go`: static asset serving for embedded web console.

### Phase 6: Embedded React 19 Tool Console
- [ ] Scaffold `web/` with Vite, React 19, TypeScript, and Tailwind CSS.
- [ ] Build `HealingGauge.tsx`: recovery rate dial with animated SVG arc.
- [ ] Build `ParameterDiff.tsx`: side-by-side and patch-badge view of original hallucinated arguments vs healed arguments.
- [ ] Build `DriftHeatmap.tsx`: visual frequency table and bar chart of hallucination-prone keys.
- [ ] Build `ToolSandbox.tsx`: interactive playground to paste invalid JSON and test healing against registered schemas with instant feedback.
- [ ] Build `LiveInvocationLog.tsx`: real-time SSE stream of intercepted calls with filters.
- [ ] Build `ToolRegistryView.tsx`: inspection catalog of all active tool schemas.
- [ ] Build production assets (`npm run build`) and link via Go `embed.FS`.

### Phase 7: CLI Subcommands, Benchmark Suite, and Integration Tests
- [ ] Build `cmd/toolheal/main.go` supporting `serve`, `proxy`, `heal`, and `bench` subcommands.
- [ ] Build `examples/failing_invocations.jsonl` containing 50+ real-world agent tool failure scenarios.
- [ ] Implement benchmark runner in `toolheal bench` proving > 85% auto-repair rate and < 2ms latency.
- [ ] Write integration tests verifying end-to-end proxy behavior and CLI pipe mode.

### Phase 8: Documentation, Recipes, and Final Verification
- [ ] Write clear, factual `README.md` following humanizer discipline (no em dashes, active voice, zero empty superlatives).
- [ ] Author `docs/architecture.md`, `docs/integration_guide.md`, and `docs/mcp_spec.md`.
- [ ] Add `LICENSE` (Apache-2.0 or MIT) and `Makefile`.
- [ ] Commit all code with semantic commit messages and push to GitHub repository.
- [ ] Verify clean build, zero compiler warnings, and all tests passing.
