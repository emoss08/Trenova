import { FREE_DEMO_PLAN_KEY } from "@/lib/plan-meters";
import type { BillingSummary } from "@/types/platform-billing";

const SECONDS_PER_DAY = 86_400;

export type CloudTrialStatus =
  | { kind: "trialing"; daysRemaining: number | null; trialEndsAt: number | null }
  | { kind: "read_only"; readOnlyUntil: number | null; daysUntilPurge: number | null }
  | { kind: "expired" };

function daysUntil(timestamp: number, now: number): number {
  return Math.max(0, Math.ceil((timestamp - now) / SECONDS_PER_DAY));
}

export const TRIAL_ENDING_METER = "shipments.total";

/**
 * The shipment allowance that ends the free demo trial early once it is used up, as the
 * server reports it. Null when the summary carries no such limit.
 */
export function trialEndingLimit(
  summary: Pick<BillingSummary, "usage"> | undefined,
): number | null {
  const meter = summary?.usage.find((entry) => entry.meterKey === TRIAL_ENDING_METER);
  return meter && meter.limit > 0 ? meter.limit : null;
}

export function isFreeDemoPlan(summary: Pick<BillingSummary, "plan" | "subscription"> | undefined) {
  const planKey = summary?.plan?.key || summary?.subscription?.planKey || "";
  return planKey === FREE_DEMO_PLAN_KEY;
}

/**
 * Where a free demo organization stands in its lifecycle, for the banner and the
 * plan page. Null for anything that is not the free demo — an internal organization
 * with no subscription, a paid plan, a self-hosted install.
 */
export function cloudTrialStatus(
  summary: Pick<BillingSummary, "plan" | "subscription"> | undefined,
  nowSeconds: number,
): CloudTrialStatus | null {
  if (!summary || !isFreeDemoPlan(summary) || !summary.subscription) {
    return null;
  }

  const { status, trialEndsAt, readOnlyUntil } = summary.subscription;

  if (status === "read_only") {
    return {
      kind: "read_only",
      readOnlyUntil: readOnlyUntil ?? null,
      daysUntilPurge: readOnlyUntil ? daysUntil(readOnlyUntil, nowSeconds) : null,
    };
  }

  if (status === "expired") {
    return { kind: "expired" };
  }

  if (status === "trialing") {
    if (trialEndsAt && trialEndsAt <= nowSeconds) {
      return {
        kind: "read_only",
        readOnlyUntil: readOnlyUntil ?? null,
        daysUntilPurge: readOnlyUntil ? daysUntil(readOnlyUntil, nowSeconds) : null,
      };
    }
    return {
      kind: "trialing",
      daysRemaining: trialEndsAt ? daysUntil(trialEndsAt, nowSeconds) : null,
      trialEndsAt: trialEndsAt ?? null,
    };
  }

  return null;
}
