import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  activatedSkillFromReadPath,
  noteActivatedSkill,
} from "./skills.mjs";

describe("activatedSkillFromReadPath", () => {
  it("extracts name from .cursor/skills paths", () => {
    assert.equal(
      activatedSkillFromReadPath(
        "/tmp/ws/.cursor/skills/skilleval-canary/SKILL.md",
      ),
      "skilleval-canary",
    );
  });

  it("extracts name from .agents/skills paths", () => {
    assert.equal(
      activatedSkillFromReadPath("project/.agents/skills/helper/SKILL.md"),
      "helper",
    );
  });

  it("accepts Windows-style separators", () => {
    assert.equal(
      activatedSkillFromReadPath(
        "C:\\ws\\.cursor\\skills\\demo\\SKILL.md",
      ),
      "demo",
    );
  });

  it("returns null for non-skill reads", () => {
    assert.equal(activatedSkillFromReadPath("src/foo.go"), null);
    assert.equal(
      activatedSkillFromReadPath(".cursor/skills/demo/references/x.md"),
      null,
    );
    assert.equal(activatedSkillFromReadPath(""), null);
    assert.equal(activatedSkillFromReadPath(null), null);
  });
});

describe("noteActivatedSkill", () => {
  it("records completed skill reads only", () => {
    const activated = new Set();
    noteActivatedSkill(activated, {
      name: "read",
      status: "running",
      args: { path: "/ws/.cursor/skills/demo/SKILL.md" },
    });
    assert.deepEqual([...activated], []);

    noteActivatedSkill(activated, {
      name: "read",
      status: "completed",
      args: { path: "/ws/.cursor/skills/demo/SKILL.md" },
    });
    noteActivatedSkill(activated, {
      name: "read",
      status: "completed",
      args: { path: "/ws/.cursor/skills/demo/SKILL.md" },
    });
    noteActivatedSkill(activated, {
      name: "read",
      status: "completed",
      args: { path: "/ws/src/main.go" },
    });
    noteActivatedSkill(activated, {
      name: "edit",
      status: "completed",
      args: { path: "/ws/.cursor/skills/other/SKILL.md" },
    });
    assert.deepEqual([...activated], ["demo"]);
  });
});
