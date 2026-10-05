import { describe, expect, it } from "vitest";
import {
  cloudTrialStatus,
  demoBannerMessages,
  shipmentsRemaining,
  trialEndingLimit,
  type CloudTrialStatus,
} from "../cloud-trial";
import type { BillingSummary } from "../../types/platform-billing";

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

describe("trialEndingLimit", () => {
  const usage = (meterKey: string, limit: number) => ({
    meterKey,
    unit: "",
    limit,
    used: 0,
    remaining: limit,
    windowStart: 0,
    windowEnd: 0,
  });

  it("reads the shipment allowance that ends the trial", () => {
    expect(
      trialEndingLimit({ usage: [usage("trailers.total", 3), usage("shipments.total", 12)] }),
    ).toBe(12);
  });

  it("is null without a shipment limit or a summary", () => {
    expect(trialEndingLimit({ usage: [usage("trailers.total", 3)] })).toBeNull();
    expect(trialEndingLimit({ usage: [usage("shipments.total", 0)] })).toBeNull();
    expect(trialEndingLimit(undefined)).toBeNull();
  });
});

describe("demo banner counts", () => {
  const meter = (meterKey: string, limit: number, used: number) => ({
    meterKey,
    unit: "",
    limit,
    used,
    remaining: Math.max(0, limit - used),
    windowStart: 0,
    windowEnd: 0,
  });
  const trialing = (daysRemaining: number | null): CloudTrialStatus => ({
    kind: "trialing",
    daysRemaining,
    trialEndsAt: daysRemaining === null ? null : NOW + daysRemaining * DAY,
  });

  it("counts shipments left from limit minus used, never below zero", () => {
    expect(shipmentsRemaining({ usage: [meter("shipments.total", 12, 5)] })).toBe(7);
    expect(shipmentsRemaining({ usage: [meter("shipments.total", 12, 15)] })).toBe(0);
    expect(shipmentsRemaining({ usage: [meter("customers.total", 8, 1)] })).toBeNull();
    expect(shipmentsRemaining({ usage: [meter("shipments.total", 0, 0)] })).toBeNull();
    expect(shipmentsRemaining(undefined)).toBeNull();
  });

  it("shows days then shipments while the trial runs", () => {
    expect(
      demoBannerMessages(trialing(6), {
        usage: [meter("trailers.total", 3, 0), meter("shipments.total", 12, 2)],
      }),
    ).toEqual([
      { kind: "days", value: 6 },
      { kind: "shipments", value: 10 },
    ]);
  });

  it("leaves out a count that does not apply or has run out", () => {
    expect(
      demoBannerMessages(trialing(null), { usage: [meter("shipments.total", 12, 2)] }),
    ).toEqual([{ kind: "shipments", value: 10 }]);
    expect(demoBannerMessages(trialing(0), { usage: [meter("shipments.total", 12, 2)] })).toEqual([
      { kind: "shipments", value: 10 },
    ]);
    expect(demoBannerMessages(trialing(3), { usage: [meter("shipments.total", 12, 12)] })).toEqual([
      { kind: "days", value: 3 },
    ]);
    expect(demoBannerMessages(trialing(3), undefined)).toEqual([{ kind: "days", value: 3 }]);
  });

  it("shows nothing once the trial is over, or off the free demo", () => {
    const usage = { usage: [meter("shipments.total", 12, 2)] };
    expect(
      demoBannerMessages({ kind: "read_only", readOnlyUntil: NOW, daysUntilPurge: 3 }, usage),
    ).toEqual([]);
    expect(demoBannerMessages({ kind: "expired" }, usage)).toEqual([]);
    expect(demoBannerMessages(null, usage)).toEqual([]);
  });
});
