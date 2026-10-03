import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** Where an item sits in the tower, by how long is left to head it off. */
export type Lane = "now" | "today" | "later" | "fyi";

export const LANES: Lane[] = ["now", "today", "later", "fyi"];

/** Inside this many minutes an item has to be acted on now. */
const NOW_MINUTES = 240;
const DAY_MINUTES = 1440;

/** Minutes until the item's deadline, negative once it has passed; null when it has none. */
export function minutesLeft(item: Pick<WatchtowerItem, "dueAt">, now: number): number | null {
  if (item.dueAt === null || item.dueAt === undefined) {
    return null;
  }
  return Math.round((item.dueAt - now) / 60);
}

/**
 * The lane an item belongs in. With a deadline it is how long is left; without
 * one, the severity its source gave it decides, so a critical item with no
 * clock on it is not filed under "for your information".
 */
export function laneOf(item: Pick<WatchtowerItem, "dueAt" | "severity">, now: number): Lane {
  const left = minutesLeft(item, now);
  if (left === null) {
    if (item.severity === "Critical") return "now";
    if (item.severity === "Warning") return "today";
    return "fyi";
  }
  if (left <= NOW_MINUTES) return "now";
  if (left <= DAY_MINUTES) return "today";
  return "later";
}

const SEVERITY_RANK = { Critical: 0, Warning: 1, Info: 2 } as const;

/** The tower in the order to work it: lane by lane, soonest deadline first, then most severe, then newest. */
export function inFocusOrder<T extends Pick<WatchtowerItem, "dueAt" | "severity" | "occurredAt">>(
  items: readonly T[],
  now: number,
): T[] {
  return [...items].sort((a, b) => {
    const lane = LANES.indexOf(laneOf(a, now)) - LANES.indexOf(laneOf(b, now));
    if (lane !== 0) return lane;
    const due = (a.dueAt ?? Number.MAX_SAFE_INTEGER) - (b.dueAt ?? Number.MAX_SAFE_INTEGER);
    if (due !== 0) return due;
    const severity =
      (SEVERITY_RANK[a.severity as keyof typeof SEVERITY_RANK] ?? 3) -
      (SEVERITY_RANK[b.severity as keyof typeof SEVERITY_RANK] ?? 3);
    if (severity !== 0) return severity;
    return b.occurredAt - a.occurredAt;
  });
}

/** "45m", "3h 10m", "9 days": a stretch of time as the tower says it. */
export function formatSpan(minutes: number, t: TranslateFn): string {
  const span = Math.abs(minutes);
  if (span < 60) return t("{0}m", span);
  if (span < DAY_MINUTES) return t("{0}h {1}m", Math.floor(span / 60), span % 60);
  return t("{0, plural, one {# day} other {# days}}", Math.round(span / DAY_MINUTES));
}

/**
 * The clock on a card: the time left and what happens then, or, once the
 * deadline has passed, how long ago, or how long a clock has been running.
 */
export function clockText(
  item: Pick<WatchtowerItem, "dueAt" | "dueLabel">,
  now: number,
  t: TranslateFn,
): { time: string; label: string } | null {
  const left = minutesLeft(item, now);
  if (left === null) return null;
  const label = item.dueLabel || t("Deadline");
  if (left > 0) return { time: formatSpan(left, t), label: t("{0} in", label) };
  if (left === 0) return { time: t("Now"), label };
  const running = /accruing|running/iu.test(label);
  return {
    time: running ? t("{0} running", formatSpan(left, t)) : t("{0} ago", formatSpan(left, t)),
    label,
  };
}

/** How far along an agent run is, by the stage it has reached. */
export type RunStage = "working" | "needs" | "done" | "failed";

const STAGES = ["Pending", "GatheringContext", "Diagnosing", "AwaitingDecision"] as const;

export function runStage(status: string): RunStage {
  if (status === "AwaitingDecision") return "needs";
  if (status === "Completed" || status === "ShadowCompleted") return "done";
  if (status === "Failed") return "failed";
  return "working";
}

/** The stages a run goes through and how many it has passed, for the progress bar. */
export function runSteps(status: string, t: TranslateFn): { steps: string[]; reached: number } {
  const steps = [
    t("Starting"),
    t("Gathering context"),
    t("Working out what to do"),
    t("Asking you to decide"),
  ];
  const index = STAGES.indexOf(status as (typeof STAGES)[number]);
  if (index >= 0) return { steps, reached: index };
  return { steps, reached: steps.length };
}

/** A run is still working while it has not reached an answer or an error. */
export function runOpen(status: string): boolean {
  return runStage(status) === "working" || runStage(status) === "needs";
}
