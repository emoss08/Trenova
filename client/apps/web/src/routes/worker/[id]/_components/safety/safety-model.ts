import type {
  DisciplinaryLadder,
  SafetyScorecard,
  WorkerDisciplinaryActionRow,
  WorkerSafetyEventRow,
} from "@/lib/graphql/worker-safety";
import {
  disciplinaryLevelSchema,
  type DisciplinaryLevel,
  type SafetyEventStatus,
} from "@trenova/shared/types/worker-safety";

export const DISCIPLINE_STAIR: readonly DisciplinaryLevel[] = disciplinaryLevelSchema.options;

export type EventFilter = "open" | "all";

/** Open events first, then the newest; closed ones keep their date order. */
export function sortEvents(events: readonly WorkerSafetyEventRow[]): WorkerSafetyEventRow[] {
  return [...events].sort((a, b) => {
    const aOpen = a.status !== "Closed";
    const bOpen = b.status !== "Closed";
    if (aOpen !== bOpen) return aOpen ? -1 : 1;
    return b.occurredAt - a.occurredAt;
  });
}

export function filterEvents(
  events: readonly WorkerSafetyEventRow[],
  filter: EventFilter,
): WorkerSafetyEventRow[] {
  return filter === "open" ? events.filter((event) => event.status !== "Closed") : [...events];
}

export function eventStatusTone(status: string): "danger" | "warning" | "neutral" {
  switch (status as SafetyEventStatus) {
    case "Open":
      return "danger";
    case "UnderReview":
      return "warning";
    default:
      return "neutral";
  }
}

/** Only the kinds that carry roadside violation codes list them. */
export function eventCarriesViolations(event: Pick<WorkerSafetyEventRow, "kind">): boolean {
  return event.kind === "Inspection" || event.kind === "Citation";
}

export type GaugeSegment = {
  id: string;
  points: number;
  /** Share of the filled length, 0 to 1. */
  share: number;
  occurredAt: number;
  rollsOffAt: number | null;
};

export type PointsGauge = {
  points: number;
  /** The last number on the track. */
  max: number;
  watch: number;
  atRisk: number;
  tone: "calm" | "watch" | "risk";
  /** The filled part of the track, 0 to 1. */
  fill: number;
  segments: GaugeSegment[];
  /** Points that roll off together, latest date first. */
  rollOffs: { date: number; points: number }[];
};

/**
 * The points track: zero to two past the at-risk line (further when the points pass it), filled
 * to the active points with one segment per event still carrying points, oldest first.
 */
export function pointsGauge(
  scorecard: Pick<
    SafetyScorecard,
    "activePoints" | "pointsWatchThreshold" | "pointsAtRiskThreshold"
  >,
  events: readonly WorkerSafetyEventRow[],
): PointsGauge {
  const points = Math.max(0, scorecard.activePoints);
  const watch = scorecard.pointsWatchThreshold;
  const atRisk = scorecard.pointsAtRiskThreshold;
  const max = Math.max(atRisk + 2, points, 1);
  const carrying = events
    .filter((event) => event.activePoints > 0)
    .sort((a, b) => a.occurredAt - b.occurredAt);
  const total = carrying.reduce((sum, event) => sum + event.activePoints, 0);
  const segments = carrying.map((event) => ({
    id: event.id,
    points: event.activePoints,
    share: total > 0 ? event.activePoints / total : 0,
    occurredAt: event.occurredAt,
    rollsOffAt: event.pointsExpireAt ?? null,
  }));
  const byDate = new Map<number, number>();
  for (const segment of segments) {
    if (segment.rollsOffAt == null) continue;
    byDate.set(segment.rollsOffAt, (byDate.get(segment.rollsOffAt) ?? 0) + segment.points);
  }
  const rollOffs = [...byDate.entries()]
    .map(([date, sum]) => ({ date, points: sum }))
    .sort((a, b) => b.date - a.date);
  return {
    points,
    max,
    watch,
    atRisk,
    tone: points >= atRisk ? "risk" : points >= watch ? "watch" : "calm",
    fill: Math.min(points, max) / max,
    segments,
    rollOffs,
  };
}

export type StairStep = {
  level: DisciplinaryLevel;
  reached: boolean;
  active: boolean;
  next: boolean;
  preview: boolean;
};

/**
 * The six steps from coaching to termination: every step up to the highest action still active
 * is reached, that one is the active step, and the server's suggested step is next. A level being
 * issued in the pane is previewed on its step.
 */
export function disciplineStair(
  ladder: Pick<DisciplinaryLadder, "highestLevel" | "suggestedLevel">,
  preview: DisciplinaryLevel | null,
): StairStep[] {
  const top = ladder.highestLevel
    ? DISCIPLINE_STAIR.indexOf(ladder.highestLevel as DisciplinaryLevel)
    : -1;
  const next = DISCIPLINE_STAIR.indexOf(ladder.suggestedLevel as DisciplinaryLevel);
  return DISCIPLINE_STAIR.map((level, index) => ({
    level,
    reached: index <= top,
    active: index === top,
    next: index === next && index > top,
    preview: preview === level,
  }));
}

/** Active actions first, then the newest. */
export function sortActions(
  actions: readonly WorkerDisciplinaryActionRow[],
): WorkerDisciplinaryActionRow[] {
  return [...actions].sort((a, b) => {
    const aActive = a.status === "Active";
    const bActive = b.status === "Active";
    if (aActive !== bActive) return aActive ? -1 : 1;
    return b.issuedAt - a.issuedAt;
  });
}

/** A decimal amount the server sends as text, as money; empty when there is none. */
export function decimalAmount(value: string | null | undefined): number | null {
  if (value == null || value.trim() === "") return null;
  const amount = Number(value);
  return Number.isFinite(amount) && amount !== 0 ? amount : null;
}
