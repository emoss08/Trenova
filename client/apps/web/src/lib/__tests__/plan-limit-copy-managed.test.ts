import { describe, expect, it } from "vitest";
import type { PlanLimitNotice } from "@trenova/shared/lib/plan-limit";
import { planLimitCopy } from "@/lib/plan-limit-copy";

const t = (message: string | null | undefined, ...args: unknown[]) =>
  (message ?? "")
    .replace(/\{0, plural, one \{([^}]*)\} other \{([^}]*)\}\}/g, (_, one: string, other: string) =>
      (args[0] === 1 ? one : other).replace("#", String(args[0])),
    )
    .replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)] ?? ""));

function quota(meter: string, limit: number, plan = "managed"): PlanLimitNotice {
  return { kind: "quota", meter, limit, used: limit, plan, message: "" };
}

describe("plan-limit copy on a managed instance", () => {
  it("offers overage or deleting documents when storage is full", () => {
    const copy = planLimitCopy(quota("documents.storage_bytes", 1073741824), t);

    expect(copy.title).toBe("Document storage limit reached");
    expect(copy.guidance).toBe(
      "Turn on overage from your Trenova account to keep uploading, or delete documents you no longer need.",
    );
  });

  it("names how many organizations the plan includes", () => {
    expect(planLimitCopy(quota("organizations.total", 1), t).guidance).toBe(
      "Your plan includes 1 organization. Add organizations to your plan from your Trenova account to create another.",
    );
    expect(planLimitCopy(quota("organizations.total", 5), t).guidance).toBe(
      "Your plan includes 5 organizations. Add organizations to your plan from your Trenova account to create another.",
    );
  });

  it("keeps the general guidance for the same meter on another plan", () => {
    expect(planLimitCopy(quota("documents.storage_bytes", 100, "free_demo"), t).guidance).toBe(
      "Delete records you no longer need to free room.",
    );
  });
});
