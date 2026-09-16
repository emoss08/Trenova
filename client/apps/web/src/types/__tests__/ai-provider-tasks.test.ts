import { aiTaskSchema } from "@/types/ai-provider";
import { describe, expect, it } from "vitest";

/**
 * The assistant, the scope guard and the insight narrator each route through
 * their own task. A provider form that cannot name those tasks cannot assign a
 * provider to them, and every one of those features then fails at call time
 * with "no provider configured" no matter what an administrator does.
 */
describe("aiTaskSchema", () => {
  it.each(["ScopeClassification", "AssistantChat", "OperationalInsights"])(
    "accepts the %s task the server routes",
    (task) => {
      expect(aiTaskSchema.safeParse(task).success).toBe(true);
    },
  );

  it("still refuses a task the server does not route", () => {
    expect(aiTaskSchema.safeParse("Telepathy").success).toBe(false);
  });
});
