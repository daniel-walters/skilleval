import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { activatedSkillFromInput } from "./skills.mjs";

describe("activatedSkillFromInput", () => {
  it("reads skill_name from Agent SDK Skill tool input", () => {
    assert.equal(
      activatedSkillFromInput({ skill_name: "canary", rationale: "x" }),
      "canary",
    );
  });

  it("falls back to skill for alternate payloads", () => {
    assert.equal(activatedSkillFromInput({ skill: "canary" }), "canary");
  });

  it("prefers skill_name when both are set", () => {
    assert.equal(
      activatedSkillFromInput({ skill_name: "a", skill: "b" }),
      "a",
    );
  });

  it("returns null for missing or empty names", () => {
    assert.equal(activatedSkillFromInput(null), null);
    assert.equal(activatedSkillFromInput({}), null);
    assert.equal(activatedSkillFromInput({ skill_name: "" }), null);
  });
});
