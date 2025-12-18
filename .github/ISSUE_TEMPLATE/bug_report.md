---
name: Bug Report
about: Report a bug or unexpected behavior
title: '[BUG] '
labels: bug
assignees: ''
---

## Description

A clear and concise description of what the bug is.

## Steps to Reproduce

1. Configure MCP server with '...'
2. Run query '...'
3. See error

## Expected Behavior

What you expected to happen.

## Actual Behavior

What actually happened.

## Environment

- **mcp-loki Version**: [e.g., v0.1.0]
- **Loki Version**: [e.g., 3.0.0]
- **MCP Client**: [e.g., Claude Desktop, custom]
- **Deployment Method**: [Container/Binary]

## Logs

<details>
<summary>MCP server logs</summary>

```text
Paste logs here
```

</details>

## Configuration

<details>
<summary>MCP config (redact secrets!)</summary>

```json
{
  "mcpServers": {
    "loki": {
      "command": "...",
      "args": ["..."]
    }
  }
}
```

</details>

## Additional Context

Add any other context about the problem here.
