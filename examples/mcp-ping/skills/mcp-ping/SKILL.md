---
name: mcp-ping
description: Calls the echo-mcp ping tool and writes the result to ping-result.json.
---

# MCP Ping

When asked to ping via MCP:

1. Use the **echo-mcp** `ping` tool (no auth).
2. Write the tool's text result to `ping-result.json` at the workspace root as JSON, e.g. `{"result":"pong"}` (or include any echoed message).
3. Do not invent a result without calling the MCP tool.
