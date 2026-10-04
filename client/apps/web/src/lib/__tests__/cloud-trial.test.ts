import { describe, expect, it } from "vitest";
import { cloudTrialStatus } from "@/lib/cloud-trial";
import type { BillingSummary } from "@/types/platform-billing";

const DAY = 86_400;
const NOW = 1_800_000_000;

function summary(
  subscription: Partial<NonNullable<BillingSummary["subscription"]>> | null,
  planKey = "free_demo",
) {
  return {
    plan: { id: "", key: planKey, name: "Free demo", status: "" },
    subscription: subscription
      ? {
          id: "",
          planId: "",
          planKey: "",
          status: "trialing",
          trialEndsAt: null,
          readOnlyUntil: null,
          currentPeriodStart: null,
          currentPeriodEnd: null,
          ...subscription,
        }
      : null,
  } satisfies Pick<BillingSummary, "plan" | "subscription">;
}

describe("cloudTrialStatus", () => {
  it("counts whole days left in the trial, rounding a part day up", () => {
    expect(cloudTrialStatus(summary({ trialEndsAt: NOW + 10 * DAY - 60 }), NOW)).toEqual({
      kind: "trialing",
      daysRemaining: 10,
      trialEndsAt: NOW + 10 * DAY - 60,
    });
  });

  it("treats a trial whose end has passed as read-only before the sweep catches up", () => {
    expect(
      cloudTrialStatus(summary({ trialEndsAt: NOW - 1, readOnlyUntil: NOW + 14 * DAY }), NOW),
    ).toEqual({ kind: "read_only", readOnlyUntil: NOW + 14 * DAY, daysUntilPurge: 14 });
  });

  it("reports read-only and expired subscriptions", () => {
    expect(cloudTrialStatus(summary({ status: "read_only", readOnlyUntil: null }), NOW)).toEqual({
      kind: "read_only",
      readOnlyUntil: null,
      daysUntilPurge: null,
    });
    expect(cloudTrialStatus(summary({ status: "expired" }), NOW)).toEqual({ kind: "expired" });
  });

  it("is null for an internal organization with no subscription and for other plans", () => {
    expect(cloudTrialStatus(summary(null), NOW)).toBeNull();
    expect(cloudTrialStatus(summary({ status: "active" }, "pro"), NOW)).toBeNull();
    expect(cloudTrialStatus(undefined, NOW)).toBeNull();
  });

  it("does not invent a deadline the server did not send", () => {
    expect(cloudTrialStatus(summary({ trialEndsAt: null }), NOW)).toMatchObject({
      kind: "trialing",
      daysRemaining: null,
    });
  });
});
