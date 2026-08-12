import { run, expect } from "@danielwaltersdev/skilleval";

const { result, workspace } = await run({
  name: "interactive-confirm",
  prompt:
    "Use the interactive-confirm skill to clean up obsolete files in this workspace.",
  replies: ["yes"],
  skill: "./skills/interactive-confirm",
  input: "./fixtures/interactive-confirm",
  model: "composer-2.5",
});

expect(result).turns.toBeLessThanOrEqual(20);
expect(result).costUSD.toBeLessThanOrEqual(1);
expect(result).skills.activated.toInclude("interactive-confirm");
expect(result, workspace).file("obsolete.txt").toHaveBeenDeleted();
expect(result).finalMessage.toMatch(/deleted|removed|yes/i);
