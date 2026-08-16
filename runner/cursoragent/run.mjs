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
const { normalizeToolArgs } = await import("./toolargs.mjs");
const { exitCodeFromToolEvent } = await import("./exitcode.mjs");
const {
  makeLog,
  userEvent,
  errorEvent,
  appendStreamEvent,
  eventsFromConversation,
  finalizeLogEvents,
} = await import("./agentlog.mjs");
const { emptyUsage, addUsage, userMessages, preferConversationTranscript, composeLocalFollowUpPrompt } =
  await import("./legagg.mjs");

function parseArgs(argv) {
  const out = { cwd: "", model: "", prompt: "", replies: [] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--cwd") out.cwd = argv[++i] ?? "";
    else if (a === "--model") out.model = argv[++i] ?? "";
    else if (a === "--prompt") out.prompt = argv[++i] ?? "";
    else if (a === "--reply") out.replies.push(argv[++i] ?? "");
  }
  return out;
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
function recordToolCall(toolCalls, toolsUsedSet, event, cwd) {
  const name = event.name ?? "unknown";
  const status = event.status ?? "completed";
  const callId = event.call_id;
  const args = normalizeToolArgs(event.args, cwd);
  const exitCode = exitCodeFromToolEvent(name, event);
  toolsUsedSet.add(name);

  if (callId) {
    const idx = toolCalls.findIndex((t) => t.callId === callId);
    if (idx >= 0) {
      const prev = toolCalls[idx];
      const next = {
        callId,
        name,
        status,
        args: args ?? prev.args,
      };
      const code = exitCode ?? prev.exitCode;
      if (typeof code === "number") next.exitCode = code;
      toolCalls[idx] = next;
      return;
    }
  }

  // Completion/error without a matching call_id: close the latest running call of this name.
  if (status !== "running") {
    for (let i = toolCalls.length - 1; i >= 0; i--) {
      if (toolCalls[i].name === name && toolCalls[i].status === "running") {
        const next = {
          callId: callId ?? toolCalls[i].callId,
          name,
          status,
          args: args ?? toolCalls[i].args,
        };
        const code = exitCode ?? toolCalls[i].exitCode;
        if (typeof code === "number") next.exitCode = code;
        toolCalls[i] = next;
        return;
      }
    }
  }

  const row = { callId, name, status, args };
  if (typeof exitCode === "number") row.exitCode = exitCode;
  toolCalls.push(row);
}

/** Serialize tool calls for helper stdout (omit empty args). */
function emitToolCalls(toolCalls) {
  return toolCalls.map(({ name, status, args, exitCode }) => {
    const out = { name, status };
    if (args && Object.keys(args).length > 0) {
      out.args = args;
    }
    if (typeof exitCode === "number") {
      out.exitCode = exitCode;
    }
    return out;
  });
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
  const { cwd, model, prompt, replies } = parseArgs(process.argv.slice(2));
  if (!cwd || !model || !prompt) {
    console.error(
      "usage: run.mjs --cwd <dir> --model <id> --prompt <text> [--reply <text> ...]",
    );
    process.exit(2);
  }

  const apiKey = process.env.CURSOR_API_KEY;
  if (!apiKey) {
    console.error("CURSOR_API_KEY is required");
    process.exit(2);
  }

  const messages = userMessages(prompt, replies);
  const toolCalls = [];
  const toolsUsedSet = new Set();
  const activatedSkills = new Set();
  const streamLogEvents = [];
  let streamTurns = 0;
  let usage = emptyUsage();
  let durationMs = 0;
  let firstId = "";
  let lastResult = null;
  let lastRun = null;
  let runStatus = "finished";

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

    for (let i = 0; i < messages.length; i++) {
      const text = messages[i];
      // Agent log keeps the clean reply; local SDK may need history in send().
      streamLogEvents.push(userEvent(text));
      const sendText =
        i === 0
          ? text
          : composeLocalFollowUpPrompt(text, streamLogEvents.slice(0, -1));

      const run = await agent.send(sendText);
      lastRun = run;
      for await (const event of run.stream()) {
        if (event.type === "tool_call") {
          recordToolCall(toolCalls, toolsUsedSet, event, cwd);
          noteActivatedSkill(activatedSkills, event);
          appendStreamEvent(streamLogEvents, event, cwd);
        } else if (event.type === "assistant") {
          streamTurns += 1;
          appendStreamEvent(streamLogEvents, event, cwd);
        }
      }

      const result = await run.wait();
      lastResult = result;
      if (!firstId) {
        firstId = result.id ?? run.id ?? "";
      }
      usage = addUsage(usage, mapUsage(result.usage));
      durationMs += result.durationMs ?? 0;
      runStatus = result.status ?? "finished";
      finalizeToolCalls(toolCalls, runStatus);

      if (runStatus !== "finished") {
        break;
      }
    }

    let conversationTurns = streamTurns;
    let logEvents = streamLogEvents;
    // Each Cursor Run owns one prompt's conversation. Multi-leg attempts must
    // keep stream-aggregated turns/log; lastRun.conversation() would truncate.
    try {
      if (
        preferConversationTranscript(messages.length) &&
        lastRun?.supports?.("conversation")
      ) {
        const conv = await lastRun.conversation();
        conversationTurns = countTurns(streamTurns, conv);
        const fromConv = eventsFromConversation(conv, prompt, cwd);
        if (fromConv) {
          logEvents = fromConv;
        }
      }
    } catch {
      // keep stream-derived turns and log
    }

    if (lastResult?.error?.message) {
      logEvents.push(errorEvent(lastResult.error.message));
    }

    const out = {
      id: firstId || lastResult?.id || lastRun?.id || "",
      status: runStatus,
      finalMessage: lastResult?.result ?? "",
      error: lastResult?.error?.message ? lastResult.error.message : null,
      durationMs,
      turns: conversationTurns,
      toolsUsed: [...toolsUsedSet],
      toolCalls: emitToolCalls(toolCalls),
      usage,
      skills: { activated: [...activatedSkills] },
      log: makeLog(finalizeLogEvents(logEvents, runStatus)),
    };
    process.stdout.write(JSON.stringify(out) + "\n");
  } catch (err) {
    if (err instanceof CursorAgentError) {
      const out = {
        id: firstId,
        status: "error",
        finalMessage: "",
        error: err.message ?? String(err),
        durationMs,
        turns: streamTurns,
        toolsUsed: [...toolsUsedSet],
        toolCalls: emitToolCalls(toolCalls),
        usage,
        skills: { activated: [...activatedSkills] },
        log: makeLog(
          finalizeLogEvents(
            [...streamLogEvents, errorEvent(err.message ?? String(err))],
            "error",
          ),
        ),
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
