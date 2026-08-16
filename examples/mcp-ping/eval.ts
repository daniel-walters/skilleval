/**
 * TypeScript authoring equivalent of eval.yaml.
 * toolsUsed "mcp" is Cursor; Claude records mcp__echo-mcp__ping.
 *
 *   skilleval run ./eval.ts
 */
import { run, expect } from "@danielwaltersdev/skilleval";

const { result, workspace } = await run({
  name: "mcp-ping",
  prompt: `Use the mcp-ping skill.

Call the echo-mcp ping tool, then write the result to ping-result.json
at the workspace root (JSON object with a "result" field).`,
  skill: "./skills/mcp-ping",
  input: "./fixtures/mcp-ping",
  mcp: "./mcp.json",
  model: "composer-2.5",
});

expect(result).turns.toBeLessThanOrEqual(20);
expect(result).costUSD.toBeLessThanOrEqual(1);
expect(result).toolsUsed.toInclude("mcp");
expect(result).skills.activated.toInclude("mcp-ping");
expect(result, workspace)
  .file("ping-result.json")
  .toHaveBeenCreated()
  .toContain(/pong/);
