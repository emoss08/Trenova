import { aiTaskSchema } from "@/types/ai-provider";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * Every routable task has to be nameable here.
 *
 * The provider catalog arrives from the server with one entry per task, and
 * the whole response is parsed through this enum — so a task the client has
 * not heard of does not degrade to an unknown row, it throws the entire
 * catalog away. The AI Control page then shows nothing at all, and the
 * feature behind the new task can never be assigned a provider.
 *
 * That is what happened when DailyBriefing landed on the server: the enum
 * was eight values, the catalog had nine, and the page went blank with a
 * ZodError about index 7. The old version of this test listed the tasks it
 * cared about by hand, so it passed throughout.
 *
 * It reads the server's own list now. A task added in Go without being added
 * here fails this test rather than the page.
 */
function serverTasks(): string[] {
  const source = readFileSync(
    join(repoRoot(), "services/tms/internal/core/domain/aiprovider/enums.go"),
    "utf8",
  );
  const block = /func AllTasks\(\) \[\]Task \{\s*return \[\]Task\{([\s\S]*?)\}/.exec(source);
  if (block === null) {
    throw new Error("could not find AllTasks() in the aiprovider enums");
  }

  const names = [...block[1].matchAll(/Task([A-Za-z]+),/g)].map((match) => match[1]);
  const constants = new Map(
    [...source.matchAll(/Task([A-Za-z]+)\s*=\s*Task\("([^"]+)"\)/g)].map((match) => [
      match[1],
      match[2],
    ]),
  );

  return names.map((name) => {
    const value = constants.get(name);
    if (value === undefined) {
      throw new Error(`AllTasks() names Task${name}, which has no constant`);
    }

    return value;
  });
}

/** The nearest ancestor holding both halves of the monorepo. */
function repoRoot(): string {
  let current = process.cwd();
  for (let depth = 0; depth < 8; depth++) {
    try {
      readFileSync(join(current, "go.work"), "utf8");

      return current;
    } catch {
      current = dirname(current);
    }
  }

  throw new Error("could not find the repository root from " + process.cwd());
}

describe("aiTaskSchema", () => {
  const tasks = serverTasks();

  it("finds the tasks the server routes", () => {
    expect(tasks.length).toBeGreaterThan(5);
  });

  it.each(tasks)("accepts the %s task the server routes", (task) => {
    expect(aiTaskSchema.safeParse(task).success).toBe(true);
  });

  it("names no task the server does not route", () => {
    expect([...aiTaskSchema.options].sort()).toEqual([...tasks].sort());
  });

  it("still refuses a task the server does not route", () => {
    expect(aiTaskSchema.safeParse("Telepathy").success).toBe(false);
  });
});
