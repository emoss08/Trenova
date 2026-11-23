import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  buildEmploymentCycles,
  CYCLE_STEPS,
  cycleStage,
  type CycleChecklist,
  type CycleEvent,
  type CycleStage,
  type EmploymentCycle,
} from "../checklists/checklist-cycles";

const DAY = 86_400;

export type LifecycleChecklist = CycleChecklist & { completedAt?: number | null };

export type LifecycleStop = {
  stage: CycleStage;
  state: "done" | "current" | "ahead";
  /** The current stop is held up: a suspension, or an offboarding under way. */
  danger: boolean;
  meta: string;
};

/** Whole months from one date to another, counting a month only once its day has come round. */
export function monthsBetween(from: number, to: number): number {
  const a = new Date(from * 1000);
  const b = new Date(to * 1000);
  let months = (b.getUTCFullYear() - a.getUTCFullYear()) * 12 + (b.getUTCMonth() - a.getUTCMonth());
  if (b.getUTCDate() < a.getUTCDate()) months -= 1;
  return Math.max(0, months);
}

/** A length of service in words: "4 years 2 months", "7 months", "Under a month". */
export function describeTenure(months: number, t: TranslateFn): string {
  const years = Math.floor(months / 12);
  const rest = months % 12;
  if (years === 0 && rest === 0) return t("Under a month");
  if (years === 0) return t("{0, plural, one {# month} other {# months}}", rest);
  if (rest === 0) return t("{0, plural, one {# year} other {# years}}", years);
  return t(
    "{0, plural, one {# year} other {# years}} {1, plural, one {# month} other {# months}}",
    years,
    rest,
  );
}

function span(from: number, to: number, t: TranslateFn): string {
  const days = Math.max(0, Math.round((to - from) / DAY));
  return t(
    "{0} to {1} · {2, plural, one {# day} other {# days}}",
    formatUnixDateMedium(from),
    formatUnixDateMedium(to),
    days,
  );
}

/** The last open suspension or leave in the cycle, with when it started. */
function openHold(
  events: readonly CycleEvent[],
): { kind: "Suspended" | "LeaveStarted"; at: number } | null {
  let suspension: number | null = null;
  let leave: number | null = null;
  for (const event of [...events].sort((a, b) => a.effectiveAt - b.effectiveAt)) {
    if (event.kind === "Suspended") suspension = event.effectiveAt;
    if (event.kind === "Reinstated") suspension = null;
    if (event.kind === "LeaveStarted") leave = event.effectiveAt;
    if (event.kind === "LeaveEnded") leave = null;
    if (event.kind === "Terminated" || event.kind === "Hired" || event.kind === "Rehired") {
      suspension = null;
      leave = null;
    }
  }
  if (suspension !== null) return { kind: "Suspended", at: suspension };
  if (leave !== null) return { kind: "LeaveStarted", at: leave };
  return null;
}

function newest<T extends LifecycleChecklist>(rows: readonly T[], kind: string): T | null {
  return (
    [...rows].filter((row) => row.kind === kind).sort((a, b) => b.startedAt - a.startedAt)[0] ??
    null
  );
}

export type LifecycleInput = {
  events: readonly CycleEvent[];
  checklists: readonly LifecycleChecklist[];
  /** The worker's hire date on file, for a history that starts without a Hired event. */
  hireDate: number | null;
  now: number;
};

/**
 * The employment drawn as four stops on a route: Onboarding, Active, Offboarding, Left, each
 * done, current or ahead, with the dates that put it there. Built from the current cycle the
 * checklists already derive (`buildEmploymentCycles`, `cycleStage`).
 */
export function buildLifecycle(
  { events, checklists, hireDate, now }: LifecycleInput,
  t: TranslateFn,
): LifecycleStop[] {
  const cycles = buildEmploymentCycles(events, checklists);
  const cycle: EmploymentCycle<LifecycleChecklist> = cycles[0] ?? {
    openedBy: null,
    closedBy: null,
    checklists: [],
  };
  const stage = cycleStage(cycle);
  const stageIndex = CYCLE_STEPS.indexOf(stage);
  const onboarding = newest(cycle.checklists, "Onboarding");
  const offboarding = newest(cycle.checklists, "Offboarding");
  const hiredAt = cycle.openedBy?.effectiveAt ?? hireDate;
  const leftAt = cycle.closedBy?.effectiveAt ?? null;
  const activeFrom = onboarding?.completedAt ?? hiredAt;
  const cycleEvents = events.filter(
    (event) =>
      (hiredAt === null || event.effectiveAt >= hiredAt) &&
      (leftAt === null || event.effectiveAt <= leftAt),
  );
  const hold = stage === "active" ? openHold(cycleEvents) : null;

  const stateOf = (index: number): LifecycleStop["state"] =>
    index < stageIndex ? "done" : index === stageIndex ? "current" : "ahead";

  const onboardingMeta = (): string => {
    if (stage === "onboarding") {
      return onboarding
        ? t("since {0}", formatUnixDateMedium(onboarding.startedAt))
        : t("Under way");
    }
    if (onboarding?.completedAt) return span(onboarding.startedAt, onboarding.completedAt, t);
    if (hiredAt) return t("Hired {0}", formatUnixDateMedium(hiredAt));
    return t("No onboarding on record");
  };

  const activeMeta = (): string => {
    if (stage === "onboarding") return t("After onboarding");
    if (stage === "active") {
      if (!activeFrom) return t("Employed");
      const since = t(
        "since {0} · {1}",
        formatUnixDateMedium(activeFrom),
        describeTenure(monthsBetween(activeFrom, now), t),
      );
      if (hold?.kind === "Suspended") {
        return t("{0} · suspended from {1}", since, formatUnixDateMedium(hold.at));
      }
      if (hold?.kind === "LeaveStarted") {
        return t("{0} · on leave from {1}", since, formatUnixDateMedium(hold.at));
      }
      return since;
    }
    if (activeFrom && leftAt) {
      return t("{0} to {1}", formatUnixDateMedium(activeFrom), formatUnixDateMedium(leftAt));
    }
    return leftAt ? t("Ended {0}", formatUnixDateMedium(leftAt)) : t("Ended");
  };

  const offboardingMeta = (): string => {
    if (stage === "offboarding") {
      return leftAt
        ? t("from {0} · checklist under way", formatUnixDateMedium(leftAt))
        : t("Checklist under way");
    }
    if (stage === "left") {
      if (offboarding?.completedAt) return span(offboarding.startedAt, offboarding.completedAt, t);
      return t("Nothing open");
    }
    return t("Not started");
  };

  const leftMeta = (): string => {
    if (stage === "left") return leftAt ? t("since {0}", formatUnixDateMedium(leftAt)) : t("Left");
    if (stage === "offboarding") return t("When offboarding settles");
    return t("Still employed");
  };

  return [
    { stage: "onboarding", state: stateOf(0), danger: false, meta: onboardingMeta() },
    {
      stage: "active",
      state: stateOf(1),
      danger: hold?.kind === "Suspended",
      meta: activeMeta(),
    },
    {
      stage: "offboarding",
      state: stateOf(2),
      danger: stage === "offboarding",
      meta: offboardingMeta(),
    },
    { stage: "left", state: stateOf(3), danger: false, meta: leftMeta() },
  ];
}
