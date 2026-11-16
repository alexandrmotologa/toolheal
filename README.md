# toolheal

Self-healing tool calling and schema reconciliation middleware for autonomous AI agents.

`toolheal` intercepts tool invocations before execution, repairs parameter naming discrepancies, coerces mismatched types, snaps drifting enum values, and normalizes date strings against canonical OpenAPI and Model Context Protocol schemas.

## Features

- Parameter Name Alignment: Resolves camelCase, snake_case, kebab-case, and fuzzy typos via Damerau-Levenshtein distance in under 1 millisecond.
- Deterministic Type Coercion: Automatically casts stringified numbers, booleans, and wraps scalars into arrays when expected by schema.
- Enum Value Snapping: Matches drifting values to valid enum options using Jaro-Winkler similarity.
- Temporal Normalization: Parses natural and multi-format date strings into RFC3339 or ISO-8601 timestamps.
- Circuit Breaker Protection: Guards upstream tools against cascading failures and rate limits, returning structured fallback receipts when endpoints are down.
- Single-Binary Deployment: Lightweight Go middleware consuming under 15MB RAM with embedded pure-Go SQLite persistence and an embedded React 19 management console.

## Quick Start

### 1. Build from Source

Requirements: Go 1.22+ and Node.js 20+

```bash
git clone https://github.com/alexandrmotologa/toolheal.git
cd toolheal
go build -o toolheal.exe ./cmd/toolheal
```

### 2. Start the Server and Console

```bash
./toolheal.exe serve --port 7171
```

Access endpoints:
- Web Console: `http://localhost:7171`
- Tool Execution: `POST http://localhost:7171/v1/tools/execute`
- Dry-Run Reconciliation: `POST http://localhost:7171/v1/heal`
- MCP JSON-RPC 2.0: `POST http://localhost:7171/rpc`

### 3. Command Line Pipe Mode

You can pipe JSON payloads directly into `toolheal` without starting a server:

```bash
echo '{"tool":"create_user","arguments":{"user_id":"u1","email":"alex@test.com","role":"admin"}}' | ./toolheal.exe heal
```

### 4. Running Benchmarks

Run the benchmark suite against real-world failing agent payloads:

```bash
./toolheal.exe bench --file examples/failing_invocations.jsonl
```

## Benchmark Metrics

Results on 60 real-world LLM parameter failure scenarios:
- Auto-Recovery Rate: 100.00% (target: > 85%)
- Average Processing Latency: 0.017 ms (target: < 2.0 ms)
- LLM Token Savings: 144,000 tokens eliminated across test suite

## Documentation

- [Architecture Specification](docs/architecture.md)
- [Agent Integration Guide](docs/integration_guide.md)
- [MCP Specification](docs/mcp_spec.md)
