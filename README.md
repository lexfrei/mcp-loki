# MCP Loki

[![Go Version](https://img.shields.io/github/go-mod/go-version/lexfrei/mcp-loki)](https://go.dev/)
[![License](https://img.shields.io/github/license/lexfrei/mcp-loki)](LICENSE)
[![Release](https://img.shields.io/github/v/release/lexfrei/mcp-loki)](https://github.com/lexfrei/mcp-loki/releases)
[![CI](https://github.com/lexfrei/mcp-loki/actions/workflows/pr.yaml/badge.svg)](https://github.com/lexfrei/mcp-loki/actions/workflows/pr.yaml)

MCP server for querying Grafana Loki logs. Enables LLMs to search and analyze logs via the Model Context Protocol.

## Features

- **LogQL Queries** — Execute range queries with flexible time ranges
- **Label Discovery** — List labels and their values for query building
- **Series Exploration** — Find log streams matching label selectors
- **Index Statistics** — Get cardinality and size metrics
- **Multiple Auth Methods** — Basic auth, Bearer token, multi-tenant (X-Scope-OrgID)
- **Multi-arch Images** — `linux/amd64` and `linux/arm64`
- **Signed Images** — Verified with cosign keyless signing

## Quick Start

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "loki": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LOKI_URL=http://host.docker.internal:3100",
        "ghcr.io/lexfrei/mcp-loki:latest"
      ]
    }
  }
}
```

## Configuration

All configuration is done via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LOKI_URL` | No | `http://localhost:3100` | Loki server URL |
| `LOKI_USERNAME` | No | — | Basic auth username |
| `LOKI_PASSWORD` | No | — | Basic auth password |
| `LOKI_TOKEN` | No | — | Bearer token (alternative to basic auth) |
| `LOKI_ORG_ID` | No | — | X-Scope-OrgID header for multi-tenant Loki |
| `MCP_HTTP_PORT` | No | — | Enable HTTP SSE transport on this port |

### Authentication Examples

**No authentication (local Loki):**

```json
{
  "args": [
    "run", "--rm", "-i",
    "-e", "LOKI_URL=http://host.docker.internal:3100",
    "ghcr.io/lexfrei/mcp-loki:latest"
  ]
}
```

**Basic authentication:**

```json
{
  "args": [
    "run", "--rm", "-i",
    "-e", "LOKI_URL=https://loki.example.com",
    "-e", "LOKI_USERNAME=admin",
    "-e", "LOKI_PASSWORD=secret",
    "ghcr.io/lexfrei/mcp-loki:latest"
  ]
}
```

**Bearer token (Grafana Cloud):**

```json
{
  "args": [
    "run", "--rm", "-i",
    "-e", "LOKI_URL=https://logs-prod-us-central1.grafana.net",
    "-e", "LOKI_TOKEN=glc_xxx",
    "-e", "LOKI_ORG_ID=123456",
    "ghcr.io/lexfrei/mcp-loki:latest"
  ]
}
```

## Available Tools

### loki_query

Execute LogQL queries against Loki.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | LogQL query string |
| `start` | string | No | Start time (RFC3339, relative like `1h`, or `now`) |
| `end` | string | No | End time (RFC3339, relative, or `now`) |
| `limit` | int | No | Maximum entries to return (default: 100) |
| `direction` | string | No | `forward` or `backward` (default: `backward`) |

**Example:**

```text
Query the last hour of nginx error logs:
- query: {app="nginx"} |= "error"
- start: 1h
- limit: 50
```

### loki_labels

Get label names or values for a specific label.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | No | Label name to get values for (omit for all labels) |
| `start` | string | No | Start time |
| `end` | string | No | End time |

**Example:**

```text
Get all label names: (no parameters)
Get values for "app" label: name=app
```

### loki_series

Find log streams matching label selectors.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `match` | []string | Yes | Label selector(s), e.g., `{app="nginx"}` |
| `start` | string | No | Start time |
| `end` | string | No | End time |

**Example:**

```text
Find all nginx streams: match=["{app=\"nginx\"}"]
```

### loki_stats

Get index statistics for a query.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | LogQL selector |
| `start` | string | No | Start time |
| `end` | string | No | End time |

**Example:**

```text
Get stats for nginx logs: query={app="nginx"}
```

## Time Formats

All time parameters accept:

- **RFC3339**: `2024-01-15T10:30:00Z`
- **Relative**: `30s`, `5m`, `1h`, `7d` (seconds, minutes, hours, days ago)
- **Keyword**: `now`

## Verification

Container images are signed with cosign keyless signing:

```bash
cosign verify ghcr.io/lexfrei/mcp-loki:latest \
  --certificate-identity-regexp=https://github.com/lexfrei/mcp-loki \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com
```

## License

[BSD-3-Clause](LICENSE)
