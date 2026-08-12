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

function parseArgs(argv) {
  const out = { cwd: "", model: "", prompt: "", skill: "" };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--cwd") out.cwd = argv[++i] ?? "";
    else if (a === "--model") out.model = argv[++i] ?? "";
    else if (a === "--prompt") out.prompt = argv[++i] ?? "";
    else if (a === "--skill") out.skill = argv[++i] ?? "";
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
  const { cwd, model, prompt } = parseArgs(process.argv.slice(2));
  if (!cwd || !model || !prompt) {
    console.error(
      "usage: run.mjs --cwd <dir> --model <id> --prompt <text> [--skill <name>]",
    );
    process.exit(2);
  }

  const toolsUsedSet = new Set();
  const toolCalls = [];
  const activatedSkills = new Set();
  const logEvents = [userEvent(prompt)];
  let resultMsg = null;

  try {
    for await (const message of query({
      prompt,
      options: {
        cwd,
        model,
        settingSources: ["project"],
        permissionMode: "bypassPermissions",
        allowDangerouslySkipPermissions: true,
        skills: "all",
      },
    })) {
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

    if (!resultMsg) {
      const errText = "claude agent: no result message";
      logEvents.push(errorEvent(errText));
      const out = {
        id: "",
        status: "error",
        finalMessage: "",
        error: errText,
        durationMs: 0,
        turns: 0,
        toolsUsed: [...toolsUsedSet],
        toolCalls,
        usage: emptyUsage(),
        skills: { activated: [...activatedSkills] },
        costUSD: null,
        log: makeLog(logEvents),
      };
      process.stdout.write(JSON.stringify(out) + "\n");
      return;
    }

    const status = mapStatus(resultMsg);
    let error = null;
    if (status === "error") {
      if (Array.isArray(resultMsg.errors) && resultMsg.errors.length > 0) {
        error = resultMsg.errors.join("; ");
      } else if (typeof resultMsg.result === "string" && resultMsg.result) {
        error = resultMsg.result;
      } else {
        error = resultMsg.subtype ?? "error";
      }
      logEvents.push(errorEvent(error));
    }

    const out = {
      id: resultMsg.session_id ?? "",
      status,
      finalMessage: status === "finished" ? (resultMsg.result ?? "") : "",
      error,
      durationMs: resultMsg.duration_ms ?? 0,
      turns: resultMsg.num_turns ?? 0,
      toolsUsed: [...toolsUsedSet],
      toolCalls,
      usage: mapUsage(resultMsg.usage),
      skills: { activated: [...activatedSkills] },
      costUSD:
        typeof resultMsg.total_cost_usd === "number"
          ? resultMsg.total_cost_usd
          : null,
      log: makeLog(logEvents),
    };
    process.stdout.write(JSON.stringify(out) + "\n");
  } catch (err) {
    const errText = err?.message ?? String(err);
    logEvents.push(errorEvent(errText));
    const out = {
      id: resultMsg?.session_id ?? "",
      status: "error",
      finalMessage: "",
      error: errText,
      durationMs: resultMsg?.duration_ms ?? 0,
      turns: resultMsg?.num_turns ?? 0,
      toolsUsed: [...toolsUsedSet],
      toolCalls,
      usage: mapUsage(resultMsg?.usage),
      skills: { activated: [...activatedSkills] },
      costUSD:
        typeof resultMsg?.total_cost_usd === "number"
          ? resultMsg.total_cost_usd
          : null,
      log: makeLog(logEvents),
    };
    process.stdout.write(JSON.stringify(out) + "\n");
  }
}

main();
