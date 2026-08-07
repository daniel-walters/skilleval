import { fail } from "./error.js";

export class ToolsUsedMatchers {
  constructor(private readonly used: string[]) {}

  toInclude(...tools: string[]): this {
    const set = new Set(this.used);
    for (const tool of tools) {
      if (!set.has(tool)) {
        fail("toolsUsed.includes", `missing required tool ${JSON.stringify(tool)}`);
      }
    }
    return this;
  }

  get not(): { toInclude: (...tools: string[]) => ToolsUsedMatchers } {
    return {
      toInclude: (...tools: string[]) => {
        const set = new Set(this.used);
        for (const tool of tools) {
          if (set.has(tool)) {
            fail("toolsUsed.excludes", `forbidden tool ${JSON.stringify(tool)} was used`);
          }
        }
        return this;
      },
    };
  }
}

export class SkillsActivatedMatchers {
  constructor(private readonly activated: string[]) {}

  toInclude(...skills: string[]): this {
    const set = new Set(this.activated);
    for (const skill of skills) {
      if (!set.has(skill)) {
        fail(
          "skills.activated.includes",
          `missing required activated skill ${JSON.stringify(skill)}`,
        );
      }
    }
    return this;
  }
}
