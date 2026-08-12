import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  emptyUsage,
  addUsage,
  addCost,
  userMessages,
} from "./legagg.mjs";

describe("legagg", () => {
  it("sums usage fields", () => {
    assert.deepEqual(
      addUsage(
        { inputTokens: 1, outputTokens: 2, cacheReadTokens: 3, cacheWriteTokens: 4, totalTokens: 10 },
        { inputTokens: 10, outputTokens: 20, cacheReadTokens: 30, cacheWriteTokens: 40, totalTokens: 100 },
      ),
      {
        inputTokens: 11,
        outputTokens: 22,
        cacheReadTokens: 33,
        cacheWriteTokens: 44,
        totalTokens: 110,
      },
    );
  });

  it("treats missing usage as zeros", () => {
    assert.deepEqual(addUsage(null, emptyUsage()), emptyUsage());
  });

  it("sums cost when either side is set", () => {
    assert.equal(addCost(null, null), null);
    assert.equal(addCost(1.5, null), 1.5);
    assert.equal(addCost(null, 2), 2);
    assert.equal(addCost(1.25, 0.75), 2);
  });

  it("builds ordered user messages from prompt and replies", () => {
    assert.deepEqual(userMessages("hi", undefined), ["hi"]);
    assert.deepEqual(userMessages("hi", []), ["hi"]);
    assert.deepEqual(userMessages("hi", ["yes", "go"]), ["hi", "yes", "go"]);
    assert.deepEqual(userMessages("hi", ["yes", "", "go"]), ["hi", "yes", "go"]);
  });
});
