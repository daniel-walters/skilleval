#!/usr/bin/env node
/**
 * Private Cursor agent helper for skilleval.
 * Prints one JSON observables blob to stdout; logs go to stderr.
 */

// Route console noise away from stdout before loading the SDK.
for (const method of ["log", "info", "warn", "debug"]) {
  console[method] = (...args) => {
    console.error(...args);
  };
}

const { Agent, CursorAgentError } = await import("@cursor/sdk");
const { countTurns } = await import("./turns.mjs");
const { noteActivatedSkill } = await import("./skills.mjs");

function parseArgs(argv) {
  const out = { cwd: "", model: "", prompt: "" };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--cwd") out.cwd = argv[++i] ?? "";
    else if (a === "--model") out.model = argv[++i] ?? "";
    else if (a === "--prompt") out.prompt = argv[++i] ?? "";
  }
  return out;
}

function emptyUsage() {
  return {
    inputTokens: 0,
    outputTokens: 0,
    cacheReadTokens: 0,
    cacheWriteTokens: 0,
    totalTokens: 0,
  };
}

function mapUsage(u) {
  if (!u) return emptyUsage();
  return {
    inputTokens: u.inputTokens ?? 0,
    outputTokens: u.outputTokens ?? 0,
    cacheReadTokens: u.cacheReadTokens ?? 0,
    cacheWriteTokens: u.cacheWriteTokens ?? 0,
    totalTokens: u.totalTokens ?? 0,
  };
}

/** Record a tool_call stream event, coalescing start/complete for the same invocation. */
function recordToolCall(toolCalls, toolsUsedSet, event) {
  const name = event.name ?? "unknown";
  const status = event.status ?? "completed";
  const callId = event.call_id;
  toolsUsedSet.add(name);

  if (callId) {
    const idx = toolCalls.findIndex((t) => t.callId === callId);
    if (idx >= 0) {
      toolCalls[idx] = { callId, name, status };
      return;
    }
  }

  // Completion/error without a matching call_id: close the latest running call of this name.
  if (status !== "running") {
    for (let i = toolCalls.length - 1; i >= 0; i--) {
      if (toolCalls[i].name === name && toolCalls[i].status === "running") {
        toolCalls[i] = {
          callId: callId ?? toolCalls[i].callId,
          name,
          status,
        };
        return;
      }
    }
  }

  toolCalls.push({ callId, name, status });
}

/** After the run ends, no tool should still be "running". */
function finalizeToolCalls(toolCalls, runStatus) {
  const fallback =
    runStatus === "error" || runStatus === "cancelled" ? "error" : "completed";
  for (const t of toolCalls) {
    if (t.status === "running") {
      t.status = fallback;
    }
  }
}

async function main() {
  const { cwd, model, prompt } = parseArgs(process.argv.slice(2));
  if (!cwd || !model || !prompt) {
    console.error("usage: run.mjs --cwd <dir> --model <id> --prompt <text>");
    process.exit(2);
  }

  const apiKey = process.env.CURSOR_API_KEY;
  if (!apiKey) {
    console.error("CURSOR_API_KEY is required");
    process.exit(2);
  }

  const toolCalls = [];
  const toolsUsedSet = new Set();
  const activatedSkills = new Set();
  let turns = 0;

  let agent;
  try {
    agent = await Agent.create({
      apiKey,
      model: { id: model },
      local: {
        cwd,
        settingSources: ["project"],
      },
    });

    const run = await agent.send(prompt);
    for await (const event of run.stream()) {
      if (event.type === "tool_call") {
        recordToolCall(toolCalls, toolsUsedSet, event);
        noteActivatedSkill(activatedSkills, event);
      } else if (event.type === "assistant") {
        turns += 1;
      }
    }

    const result = await run.wait();
    let conversationTurns = turns;
    try {
      if (run.supports?.("conversation")) {
        const conv = await run.conversation();
        conversationTurns = countTurns(turns, conv);
      }
    } catch {
      // keep stream-derived turns
    }

    const runStatus = result.status ?? "finished";
    finalizeToolCalls(toolCalls, runStatus);

    const out = {
      id: result.id ?? run.id ?? "",
      status: runStatus,
      finalMessage: result.result ?? "",
      error: result.error?.message ? result.error.message : null,
      durationMs: result.durationMs ?? 0,
      turns: conversationTurns,
      toolsUsed: [...toolsUsedSet],
      toolCalls: toolCalls.map(({ name, status }) => ({ name, status })),
      usage: mapUsage(result.usage),
      skills: { activated: [...activatedSkills] },
    };
    process.stdout.write(JSON.stringify(out) + "\n");
  } catch (err) {
    if (err instanceof CursorAgentError) {
      const out = {
        id: "",
        status: "error",
        finalMessage: "",
        error: err.message ?? String(err),
        durationMs: 0,
        turns: 0,
        toolsUsed: [],
        toolCalls: [],
        usage: emptyUsage(),
        skills: { activated: [] },
      };
      process.stdout.write(JSON.stringify(out) + "\n");
      return;
    }
    console.error(err);
    process.exit(1);
  } finally {
    if (agent && typeof agent[Symbol.asyncDispose] === "function") {
      await agent[Symbol.asyncDispose]();
    }
  }
}

main();
