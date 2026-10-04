import { SIGNUPS_PAUSED_REASON, type PlanLimitNotice } from "@trenova/shared/lib/plan-limit";
import { create } from "zustand";

/**
 * How long a refusal the person has just dismissed stays quiet. A screen that polls,
 * or a bulk action that trips the same limit on every row, would otherwise reopen
 * the dialog the moment it closed.
 */
export const PLAN_LIMIT_QUIET_MS = 8_000;

export function planLimitKey(notice: PlanLimitNotice): string {
  return notice.kind === "quota"
    ? `quota:${notice.meter}`
    : `restricted:${notice.capability}:${notice.reason}`;
}

export type PlanLimitDecision = "open" | "absorb" | "ignore";

/**
 * What to do with a refusal the transport just reported:
 * - "open" shows the dialog;
 * - "absorb" leaves the dialog as it is — it is already open, or was just dismissed
 *   for this same limit — but still counts as explained, so no toast repeats it;
 * - "ignore" leaves it to the caller entirely: nobody is signed in (a signup page
 *   explains its own refusals), or the refusal is the signup wait list, which is not
 *   a limit on an organization.
 */
export function decidePlanLimit({
  notice,
  isAuthenticated,
  openNotice,
  lastDismissed,
  now,
}: {
  notice: PlanLimitNotice;
  isAuthenticated: boolean;
  openNotice: PlanLimitNotice | null;
  lastDismissed: { key: string; at: number } | null;
  now: number;
}): PlanLimitDecision {
  if (!isAuthenticated) {
    return "ignore";
  }
  if (notice.kind === "restricted" && notice.reason === SIGNUPS_PAUSED_REASON) {
    return "ignore";
  }
  if (openNotice) {
    return "absorb";
  }
  if (
    lastDismissed &&
    lastDismissed.key === planLimitKey(notice) &&
    now - lastDismissed.at < PLAN_LIMIT_QUIET_MS
  ) {
    return "absorb";
  }
  return "open";
}

type PlanLimitState = {
  notice: PlanLimitNotice | null;
  lastDismissed: { key: string; at: number } | null;
  show: (notice: PlanLimitNotice) => void;
  dismiss: () => void;
};

export const usePlanLimitStore = create<PlanLimitState>()((set, get) => ({
  notice: null,
  lastDismissed: null,
  show: (notice) => set({ notice }),
  dismiss: () => {
    const { notice } = get();
    set({
      notice: null,
      lastDismissed: notice ? { key: planLimitKey(notice), at: Date.now() } : get().lastDismissed,
    });
  },
}));
