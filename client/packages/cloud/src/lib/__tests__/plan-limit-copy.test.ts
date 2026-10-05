import type { PlanLimitNotice } from "@trenova/shared/lib/plan-limit";
import { describe, expect, it } from "vitest";
import { cloudPlanLimitCopy } from "../plan-limit-copy";

const t = (message: string | null | undefined, ...args: unknown[]) =>
  (message ?? "").replace(/\{(\d+)\}/g, (_, index: string) => {
    const value = args[Number(index)];
    return typeof value === "string" || typeof value === "number" ? String(value) : "";
  });

const shipmentsQuota: PlanLimitNotice = {
  kind: "quota",
  meter: "shipments.total",
  limit: 12,
  used: 12,
  plan: "free_demo",
  message: "",
};

const readOnly: PlanLimitNotice = {
  kind: "restricted",
  capability: "",
  reason: "subscription_read_only",
  plan: "free_demo",
  message: "",
};

describe("cloudPlanLimitCopy", () => {
  it("names the meter and its figures for a lifetime quota, and how to free room", () => {
    const copy = cloudPlanLimitCopy(shipmentsQuota, t);
    expect(copy.title).toBe("Shipments limit reached");
    expect(copy.description).toBe("Shipments: 12 of 12 used (Free demo).");
    expect(copy.guidance).toContain("Delete records you no longer need");
    expect(copy.guidance).toContain("Paid plans with higher limits are coming soon.");
    expect(copy.usage).toEqual({ used: 12, limit: 12, usedLabel: "12", limitLabel: "12" });
  });

  it("says a monthly limit resets, and that paid plans are coming", () => {
    const spend = cloudPlanLimitCopy(
      { ...shipmentsQuota, meter: "ai.spend_cents", limit: 150, used: 150 },
      t,
    );
    expect(spend.usage?.limitLabel).toBe("$1.50");
    expect(spend.guidance).toBe(
      "This limit resets at the start of next month. Paid plans with higher limits are coming soon.",
    );
  });

  it("explains a per-file limit without a usage bar", () => {
    const copy = cloudPlanLimitCopy(
      { ...shipmentsQuota, meter: "documents.file_bytes", limit: 10485760, used: 25000000 },
      t,
    );
    expect(copy.description).toBe("Free demo allows files up to 10.0 MB. This one is larger.");
    expect(copy.guidance).toContain("Upload a smaller file");
    expect(copy.usage).toBeNull();
  });

  it("still explains a quota whose figures did not arrive", () => {
    const copy = cloudPlanLimitCopy({ ...shipmentsQuota, limit: null, used: null }, t);
    expect(copy.usage).toBeNull();
    expect(copy.description).toBe("Your organization has reached its Shipments limit (Free demo).");
  });

  it("explains that the trial ended rather than blaming the action", () => {
    const copy = cloudPlanLimitCopy(readOnly, t);
    expect(copy.title).toBe("Your free demo has ended");
    expect(copy.description).toContain("read-only");
  });

  it("names a restricted capability and what the demo leaves out", () => {
    const copy = cloudPlanLimitCopy(
      { ...readOnly, capability: "integrations", reason: "capability_restricted" },
      t,
    );
    expect(copy.title).toBe("Integrations not available");
    expect(copy.description).toContain("Samsara");
    expect(copy.guidance).toBe(
      "Free demo is for trying Trenova and leaves this out. Paid plans that include it are coming soon.",
    );
  });

  it("falls back to a readable label for a meter the client does not know", () => {
    expect(cloudPlanLimitCopy({ ...shipmentsQuota, meter: "edi.partners" }, t).title).toBe(
      "Edi partners limit reached",
    );
  });
});
