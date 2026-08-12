import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  makeLog,
  userEvent,
  errorEvent,
  appendStreamEvent,
  eventsFromConversation,
  appendClaudeMessage,
  finalizeLogEvents,
  assistantTextFromStreamEvent,
} from "./agentlog.mjs";

describe("makeLog", () => {
  it("wraps events with schemaVersion 1", () => {
    assert.deepEqual(makeLog([{ type: "user", text: "hi" }]), {
      schemaVersion: 1,
      events: [{ type: "user", text: "hi" }],
    });
  });
});

describe("eventsFromConversation", () => {
  it("converts assistant and toolCall steps", () => {
    const conversation = [
      {
        type: "agentConversationTurn",
        turn: {
          steps: [
            { type: "assistantMessage", message: { text: "I'll read" } },
            {
              type: "toolCall",
              message: { name: "read", status: "completed", args: { path: "a.go" } },
            },
            { type: "assistantMessage", message: { text: "Done" } },
          ],
        },
      },
    ];
    const events = eventsFromConversation(conversation, "refactor please");
    assert.deepEqual(events, [
      { type: "user", text: "refactor please" },
      { type: "assistant", text: "I'll read" },
      { type: "tool_call", name: "read", status: "completed", args: { path: "a.go" } },
      { type: "assistant", text: "Done" },
    ]);
  });

  it("returns null when conversation has no usable steps", () => {
    assert.equal(eventsFromConversation([], "p"), null);
    assert.equal(eventsFromConversation(null, "p"), null);
    assert.equal(
      eventsFromConversation(
        [{ type: "shellConversationTurn", turn: {} }],
        "p",
      ),
      null,
    );
  });

  it("returns null for tool-only conversation (keep stream log)", () => {
    const conversation = [
      {
        type: "agentConversationTurn",
        turn: {
          steps: [
            { type: "thinkingMessage", message: { text: "..." } },
            { type: "toolCall", message: { name: "read" } },
          ],
        },
      },
    ];
    assert.equal(eventsFromConversation(conversation, "p"), null);
  });

  it("emits multiple user messages across conversation turns", () => {
    const conversation = [
      {
        type: "agentConversationTurn",
        turn: {
          userMessage: { text: "start" },
          steps: [{ type: "assistantMessage", message: { text: "Confirm?" } }],
        },
      },
      {
        type: "agentConversationTurn",
        turn: {
          userMessage: { text: "yes" },
          steps: [
            {
              type: "toolCall",
              message: { name: "delete", status: "completed", args: { path: "x" } },
            },
            { type: "assistantMessage", message: { text: "Deleted" } },
          ],
        },
      },
    ];
    const events = eventsFromConversation(conversation, "start");
    assert.deepEqual(events, [
      { type: "user", text: "start" },
      { type: "assistant", text: "Confirm?" },
      { type: "user", text: "yes" },
      { type: "tool_call", name: "delete", status: "completed", args: { path: "x" } },
      { type: "assistant", text: "Deleted" },
    ]);
  });
});

describe("appendStreamEvent", () => {
  it("records assistant text from content blocks", () => {
    const events = [userEvent("hi")];
    appendStreamEvent(events, {
      type: "assistant",
      message: { content: [{ type: "text", text: "Working…" }] },
    });
    assert.deepEqual(events[1], { type: "assistant", text: "Working…" });
  });

  it("coalesces tool_call by call_id", () => {
    const events = [];
    appendStreamEvent(events, {
      type: "tool_call",
      call_id: "c1",
      name: "read",
      status: "running",
      args: { path: "x.go" },
    });
    appendStreamEvent(events, {
      type: "tool_call",
      call_id: "c1",
      name: "read",
      status: "completed",
    });
    assert.equal(events.length, 1);
    assert.equal(events[0].status, "completed");
    assert.equal(events[0].args.path, "x.go");
    assert.deepEqual(finalizeLogEvents(events, "finished"), [
      { type: "tool_call", name: "read", status: "completed", args: { path: "x.go" } },
    ]);
  });
});

describe("finalizeLogEvents", () => {
  it("closes leftover running tools from run status", () => {
    const events = [
      { type: "tool_call", callId: "c1", name: "shell", status: "running" },
    ];
    assert.deepEqual(finalizeLogEvents(events, "cancelled"), [
      { type: "tool_call", name: "shell", status: "error" },
    ]);
    assert.deepEqual(finalizeLogEvents(events, "finished"), [
      { type: "tool_call", name: "shell", status: "completed" },
    ]);
  });
});

describe("appendClaudeMessage", () => {
  it("emits text and tool_use blocks", () => {
    const events = [userEvent("go")];
    appendClaudeMessage(events, {
      type: "assistant",
      message: {
        content: [
          { type: "text", text: "Looking" },
          { type: "tool_use", name: "Read", input: { file_path: "a.ts" } },
        ],
      },
    });
    assert.deepEqual(events.slice(1), [
      { type: "assistant", text: "Looking" },
      { type: "tool_call", name: "Read", status: "completed", args: { path: "a.ts" } },
    ]);
  });
});

describe("helpers", () => {
  it("builds user and error events", () => {
    assert.deepEqual(userEvent("a"), { type: "user", text: "a" });
    assert.deepEqual(errorEvent("boom"), { type: "error", text: "boom" });
  });

  it("reads assistant text from stream events", () => {
    assert.equal(
      assistantTextFromStreamEvent({
        message: { content: [{ type: "text", text: "hi" }] },
      }),
      "hi",
    );
  });
});
