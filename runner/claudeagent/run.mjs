#!/usr/bin/env node
/**
 * Private Claude agent helper for skilleval.
 * Prints one JSON observables blob to stdout; logs go to stderr.
 */

for (const method of ["log", "info", "warn", "debug"]) {
  console[method] = (...args) => {
    console.error(...args);
  };
}

const { query } = await import("@anthropic-ai/claude-agent-sdk");
const { activatedSkillFromInput } = await import("./skills.mjs");
const { normalizeToolArgs } = await import("./toolargs.mjs");
const {
  makeLog,
  userEvent,
  errorEvent,
  appendClaudeMessage,
} = await import("./agentlog.mjs");
const { emptyUsage, addUsage, addCost, userMessages } = await import(
  "./legagg.mjs"
);

function parseArgs(argv) {
  const out = { cwd: "", model: "", prompt: "", skill: "", replies: [] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--cwd") out.cwd = argv[++i] ?? "";
    else if (a === "--model") out.model = argv[++i] ?? "";
    else if (a === "--prompt") out.prompt = argv[++i] ?? "";
    else if (a === "--skill") out.skill = argv[++i] ?? "";
    else if (a === "--reply") out.replies.push(argv[++i] ?? "");
  }
  return out;
}

function mapUsage(u) {
  if (!u) return emptyUsage();
  const input = u.input_tokens ?? u.inputTokens ?? 0;
  const output = u.output_tokens ?? u.outputTokens ?? 0;
  const cacheRead = u.cache_read_input_tokens ?? u.cacheReadTokens ?? 0;
  const cacheWrite = u.cache_creation_input_tokens ?? u.cacheWriteTokens ?? 0;
  return {
    inputTokens: input,
    outputTokens: output,
    cacheReadTokens: cacheRead,
    cacheWriteTokens: cacheWrite,
    totalTokens: input + output + cacheRead + cacheWrite,
  };
}

function mapStatus(resultMsg) {
  if (!resultMsg) return "error";
  if (resultMsg.is_error) return "error";
  switch (resultMsg.subtype) {
    case "success":
      return "finished";
    case "error_max_turns":
    case "error_during_execution":
    case "error_max_budget_usd":
    case "error_max_structured_output_retries":
      return "error";
    default:
      return resultMsg.is_error === false ? "finished" : "error";
  }
}

async function main() {
  const { cwd, model, prompt, replies } = parseArgs(process.argv.slice(2));
  if (!cwd || !model || !prompt) {
    console.error(
      "usage: run.mjs --cwd <dir> --model <id> --prompt <text> [--reply <text> ...] [--skill <name>]",
    );
    process.exit(2);
  }

  const messages = userMessages(prompt, replies);
  const toolsUsedSet = new Set();
  const toolCalls = [];
  const activatedSkills = new Set();
  const logEvents = [];
  let usage = emptyUsage();
  let durationMs = 0;
  let turns = 0;
  let costUSD = null;
  let firstId = "";
  let sessionId = undefined;
  let lastResultMsg = null;
  let status = "finished";
  let error = null;

  try {
    for (let i = 0; i < messages.length; i++) {
      const text = messages[i];
      logEvents.push(userEvent(text));

      const options = {
        cwd,
        model,
        settingSources: ["project"],
        permissionMode: "bypassPermissions",
        allowDangerouslySkipPermissions: true,
        skills: "all",
      };
      if (i > 0 && sessionId) {
        options.resume = sessionId;
      }

      let resultMsg = null;
      try {
        for await (const message of query({ prompt: text, options })) {
          if (message.type === "assistant" && message.message?.content) {
            appendClaudeMessage(logEvents, message, cwd);
            for (const block of message.message.content) {
              if (block?.type !== "tool_use") continue;
              const name = block.name ?? "unknown";
              toolsUsedSet.add(name);
              const entry = { name, status: "completed" };
              const args = normalizeToolArgs(block.input, cwd);
              if (args) entry.args = args;
              toolCalls.push(entry);
              if (name === "Skill") {
                const skillName = activatedSkillFromInput(block.input);
                if (skillName) {
                  activatedSkills.add(skillName);
                }
              }
            }
          }
          if (message.type === "result") {
            resultMsg = message;
          }
        }
      } catch (err) {
        // Single-shot query() may throw after yielding an error result.
        if (!resultMsg) {
          throw err;
        }
      }

      lastResultMsg = resultMsg;
      if (!resultMsg) {
        status = "error";
        error = "claude agent: no result message";
        logEvents.push(errorEvent(error));
        break;
      }

      if (!firstId) {
        firstId = resultMsg.session_id ?? "";
      }
      sessionId = resultMsg.session_id ?? sessionId;
      usage = addUsage(usage, mapUsage(resultMsg.usage));
      durationMs += resultMsg.duration_ms ?? 0;
      turns += resultMsg.num_turns ?? 0;
      costUSD = addCost(
        costUSD,
        typeof resultMsg.total_cost_usd === "number"
          ? resultMsg.total_cost_usd
          : null,
      );

      status = mapStatus(resultMsg);
      error = null;
      if (status === "error") {
        if (Array.isArray(resultMsg.errors) && resultMsg.errors.length > 0) {
          error = resultMsg.errors.join("; ");
        } else if (typeof resultMsg.result === "string" && resultMsg.result) {
          error = resultMsg.result;
        } else {
          error = resultMsg.subtype ?? "error";
        }
        logEvents.push(errorEvent(error));
        break;
      }
    }

    const out = {
      id: firstId || lastResultMsg?.session_id || "",
      status,
      finalMessage:
        status === "finished" ? (lastResultMsg?.result ?? "") : "",
      error,
      durationMs,
      turns,
      toolsUsed: [...toolsUsedSet],
      toolCalls,
      usage,
      skills: { activated: [...activatedSkills] },
      costUSD,
      log: makeLog(logEvents),
    };
    process.stdout.write(JSON.stringify(out) + "\n");
  } catch (err) {
    const errText = err?.message ?? String(err);
    logEvents.push(errorEvent(errText));
    const out = {
      id: firstId || lastResultMsg?.session_id || "",
      status: "error",
      finalMessage: "",
      error: errText,
      durationMs,
      turns,
      toolsUsed: [...toolsUsedSet],
      toolCalls,
      usage,
      skills: { activated: [...activatedSkills] },
      costUSD,
      log: makeLog(logEvents),
    };
    process.stdout.write(JSON.stringify(out) + "\n");
  }
}

main();
