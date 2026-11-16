# toolheal Architecture Specification

`toolheal` is an execution middleware written in Go for autonomous AI agent workflows. It sits between agent execution loops and tool providers (REST APIs, Model Context Protocol servers, and local command line utilities). Its primary role is intercepting invalid or malformed tool calls, repairing parameter discrepancies against canonical schemas in under 1 millisecond, and eliminating context-burning LLM retry loops.

## System Topology

```
                  +----------------------------------------------+
                  |         AI Agent Execution Runtime           |
                  |     (Claude Code, LangGraph, CrewAI)         |
                  +----------------------------------------------+
                                         |
                       POST /v1/tools/execute or /rpc
                                         v
+--------------------------------------------------------------------------------+
|                           toolheal Middleware                                  |
|                                                                                |
|  +--------------------+   +-------------------------------------------------+  |
|  | Tool Schema        |   | Multi-Stage Reconciliation Pipeline             |  |
|  | Catalog            |   |                                                 |  |
|  | - OpenAPI 3.1      |-->| 1. Envelope Unwrapping                          |  |
|  | - MCP tool schemas |   | 2. Case & Delimiter Normalization               |  |
|  +--------------------+   | 3. Global & Explicit Alias Resolution           |  |
|                           | 4. Damerau-Levenshtein & Jaro-Winkler Matching  |  |
|                           | 5. Primitive Type Coercion & Array Wrapping     |  |
|                           | 6. Enum Nearest-Neighbor Snapping               |  |
|                           | 7. Temporal & ISO-8601 Date Normalization       |  |
|                           | 8. Schema Default & Missing Key Injection       |  |
|                           +-------------------------------------------------+  |
|                                                    |                           |
|                                                    v                           |
|                           +-------------------------------------------------+  |
|                           | Safe Tool Dispatcher & Circuit Breaker          |  |
|                           | - State tracking: CLOSED, OPEN, HALF_OPEN       |  |
|                           | - Exponential backoff with jitter on 429/5xx    |  |
|                           | - Deterministic semantic fallback receipts      |  |
|                           +-------------------------------------------------+  |
|                                                    |                           |
|                                                    v                           |
|  +--------------------+   +-------------------------------------------------+  |
|  | Telemetry Store    |<--| Execution Output & Audit Logging                |  |
|  | - Pure-Go SQLite   |   | - Latency microsecond recording                 |  |
|  | - modernc.org/sqlite   | - Tokens saved accounting                       |  |
|  | - SSE Event Stream |   | - Parameter drift frequency matrix              |  |
|  +--------------------+   +-------------------------------------------------+  |
+--------------------------------------------------------------------------------+
                                         |
                                         v
                  +----------------------------------------------+
                  |               Target Service                 |
                  |     (Mock Server, External REST, MCP)        |
                  +----------------------------------------------+
```

## Reconciliation Pipeline Stages

### Stage 1: Envelope Unwrapping
Language models frequently wrap arguments inside structural objects, such as `{"arguments": {...}}`, `{"params": {...}}`, or `{"input": {...}}`. The engine unpacks these wrapper keys if only a single nested object is present, extracting direct key-value pairs before schema evaluation.

### Stage 2: Case and Delimiter Folding
Parameter names vary across models and prompt instructions (`user_id`, `userId`, `user-id`, `User_Id`). The engine strips all underscores, hyphens, dots, and whitespace while converting characters to lowercase. If the folded string matches a folded schema property key, the parameter is immediately renamed to the canonical casing.

### Stage 3: Semantic and Explicit Alias Resolution
The engine maintains bidirectional equivalence groups for standard enterprise parameter concepts:
- Identifiers: `user_id`, `userId`, `id`, `uid`, `customer_id`
- Pagination: `page_size`, `pageSize`, `limit`, `per_page`, `count`, `max_results`
- Offsets: `page_number`, `pageNumber`, `page`, `page_index`, `offset`
- Search queries: `query`, `q`, `search`, `search_term`, `keyword`
- Timestamps: `created_date`, `created_date`, `created_at`, `timestamp`, `since`

Individual tool definitions may also declare explicit property aliases in their schema definitions.

### Stage 4: Damerau-Levenshtein and Jaro-Winkler Matching
For unmatched keys, edit distance and prefix-weighted similarity are evaluated against canonical schema keys:

$$\text{Sim}_{\text{Lev}}(u, k) = 1 - \frac{\text{DamerauLevenshtein}(u, k)}{\max(|u|, |k|)}$$

Combined score:

$$\text{Score}(u, k) = 0.5 \cdot \text{Sim}_{\text{Lev}}(u, k) + 0.5 \cdot \text{JaroWinkler}(u, k)$$

The key is mapped to the canonical candidate with the highest score, provided the score meets or exceeds 0.80 or the edit distance is within 2 edits.

### Stage 5: Primitive Type Coercion
- Strings representing integers or floats (`"100"`, `"19.99"`) are parsed into numeric primitives.
- String booleans (`"true"`, `"false"`, `"yes"`, `"no"`, `"1"`, `"0"`) are converted to Go boolean types.
- Scalar values passed where `type: "array"` is declared are wrapped into a single-element slice.
- Arrays of length 1 passed where a scalar primitive is expected are unpacked.
- Comma-delimited strings passed to array fields (`"admin, operator, reviewer"`) are split into slices.
- Embedded JSON strings are unpacked into nested maps or slices.

### Stage 6: Nearest-Neighbor Enum Snapping
When a string parameter fails exact match against a schema enum list:
1. Case-insensitive comparison is attempted.
2. Delimiter-folded matching is evaluated.
3. Common synonyms are resolved (e.g. `urgent` to `HIGH`, `administrator` to `ADMIN`).
4. Jaro-Winkler string similarity snaps drifting values to the closest permitted enum variant when similarity meets or exceeds 0.70.

### Stage 7: Temporal Normalization
API endpoints frequently reject timestamps due to layout differences. The temporal normalizer accepts RFC3339, ISO-8601, European date notation (`DD/MM/YYYY`), US notation (`MM/DD/YYYY`), Unix epoch seconds, Unix epoch milliseconds, and relative references (`today`, `yesterday`, `now`). Outputs are formatted strictly into RFC3339 (`2026-11-15T00:00:00Z`) or `YYYY-MM-DD` depending on the schema format field.

### Stage 8: Default Value Injection
Missing optional or required parameters that declare defaults in the schema are populated with those default values.

## Circuit Breaker and Failure Recovery

To prevent cascading latency in multi-agent environments, each tool target is protected by an independent circuit breaker:
- `CLOSED`: Normal operation. Invocations dispatch directly to the upstream tool target.
- `OPEN`: Tripped after consecutive upstream failures exceed the threshold (default: 5). Invocations immediately synthesize a structured fallback receipt without hitting the failed network endpoint.
- `HALF_OPEN`: After the reset timeout expires (default: 10 seconds), a trial invocation is dispatched. If successful, the circuit returns to `CLOSED`.

### Fallback Receipts
When a tool is unavailable, `toolheal` generates a semantic receipt:
```json
{
  "status": "healed_fallback",
  "tool_name": "payment_gateway",
  "execution_mode": "semantic_fallback",
  "timestamp": "2026-11-16T14:00:00Z",
  "reason": "upstream tool service unavailable",
  "arguments_attempted": { ... },
  "agent_guidance": "The underlying tool endpoint is temporarily unreachable. You may proceed with alternative logic or present a graceful acknowledgment to the user."
}
```
This structured receipt prevents unhandled exceptions from terminating the outer agent loop.

## Telemetry and Persistence

Persistent auditing is implemented via pure-Go SQLite (`modernc.org/sqlite`), requiring zero CGO compilation.
- `invocations`: Stores call ID, tool name, raw input, repaired output, applied diffs, tokens saved, and microsecond latency.
- `schema_drifts`: Tracks cumulative occurrences of specific hallucinated parameter names per tool.
- Real-time updates are distributed over Server-Sent Events (`GET /api/stream`) to connected dashboard instances.
