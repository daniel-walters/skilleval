import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { countTurns } from "./turns.mjs";

describe("countTurns", () => {
  it("counts assistantMessage steps across a multi-round conversation", () => {
    const conversation = [
      {
        type: "agentConversationTurn",
        turn: {
          steps: [
            { type: "assistantMessage", message: { text: "I'll read the file" } },
            { type: "toolCall", message: { name: "read" } },
            { type: "assistantMessage", message: { text: "I'll edit next" } },
            { type: "toolCall", message: { name: "write" } },
            { type: "assistantMessage", message: { text: "Done" } },
          ],
        },
      },
    ];
    assert.equal(countTurns(1, conversation), 3);
  });

  it("reports 1 for a single assistantMessage round", () => {
    const conversation = [
      {
        type: "agentConversationTurn",
        turn: {
          steps: [{ type: "assistantMessage", message: { text: "Hello" } }],
        },
      },
    ];
    assert.equal(countTurns(9, conversation), 1);
  });

  it("falls back to streamTurns when conversation is missing", () => {
    assert.equal(countTurns(4, null), 4);
    assert.equal(countTurns(4, undefined), 4);
  });

  it("falls back to streamTurns when conversation is empty", () => {
    assert.equal(countTurns(2, []), 2);
  });

  it("falls back when agent turns have no assistantMessage steps", () => {
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
    assert.equal(countTurns(5, conversation), 5);
  });

  it("ignores shellConversationTurn entries", () => {
    const conversation = [
      {
        type: "shellConversationTurn",
        turn: { shellCommand: { command: "ls" } },
      },
      {
        type: "agentConversationTurn",
        turn: {
          steps: [
            { type: "assistantMessage", message: { text: "a" } },
            { type: "assistantMessage", message: { text: "b" } },
          ],
        },
      },
    ];
    assert.equal(countTurns(1, conversation), 2);
  });
});
