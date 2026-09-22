import { goEnumValues } from "@/test/go-source";
import { aiTaskSchema } from "@/types/ai-provider";
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
  return goEnumValues({
    file: "services/tms/internal/core/domain/aiprovider/enums.go",
    typeName: "Task",
  });
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
