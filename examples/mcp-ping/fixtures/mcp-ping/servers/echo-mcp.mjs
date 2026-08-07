#!/usr/bin/env node
// Minimal no-auth MCP stdio server with a single `ping` tool.
// Protocol: JSON-RPC 2.0, one message per line on stdin/stdout.

import readline from "node:readline";

const PROTOCOL_VERSION = "2024-11-05";
const SERVER_INFO = { name: "echo-mcp", version: "1.0.0" };

const tools = [
  {
    name: "ping",
    description: "Returns pong (optional message echoed back).",
    inputSchema: {
      type: "object",
      properties: {
        message: { type: "string", description: "Optional text to echo" },
      },
    },
  },
];

function send(msg) {
  process.stdout.write(JSON.stringify(msg) + "\n");
}

function handle(msg) {
  if (!msg || typeof msg !== "object") {
    return;
  }
  const { id, method, params } = msg;
  if (method === undefined) {
    return;
  }
  // Notifications have no id and must not get a response.
  if (id === undefined) {
    return;
  }

  try {
    if (method === "initialize") {
      send({
        jsonrpc: "2.0",
        id,
        result: {
          protocolVersion: PROTOCOL_VERSION,
          capabilities: { tools: {} },
          serverInfo: SERVER_INFO,
        },
      });
      return;
    }
    if (method === "ping") {
      send({ jsonrpc: "2.0", id, result: {} });
      return;
    }
    if (method === "tools/list") {
      send({ jsonrpc: "2.0", id, result: { tools } });
      return;
    }
    if (method === "tools/call") {
      const name = params?.name;
      const args = params?.arguments ?? {};
      if (name !== "ping") {
        send({
          jsonrpc: "2.0",
          id,
          error: { code: -32601, message: `Unknown tool: ${name}` },
        });
        return;
      }
      const message = args.message != null ? String(args.message) : "";
      const text = message ? `pong: ${message}` : "pong";
      send({
        jsonrpc: "2.0",
        id,
        result: { content: [{ type: "text", text }], isError: false },
      });
      return;
    }
    send({
      jsonrpc: "2.0",
      id,
      error: { code: -32601, message: `Method not found: ${method}` },
    });
  } catch (err) {
    send({
      jsonrpc: "2.0",
      id,
      error: { code: -32000, message: String(err?.message || err) },
    });
  }
}

const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
rl.on("line", (line) => {
  const trimmed = line.trim();
  if (!trimmed) {
    return;
  }
  try {
    handle(JSON.parse(trimmed));
  } catch (err) {
    process.stderr.write(`echo-mcp: parse error: ${err}\n`);
  }
});
