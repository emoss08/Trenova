import type {
  DisciplinaryLadder,
  SafetyScorecard,
  WorkerDisciplinaryActionRow,
  WorkerSafetyEventRow,
} from "@/lib/graphql/worker-safety";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { describeSafetyEvent } from "@trenova/shared/lib/safety";
import {
  disciplinaryLevelSchema,
  type DisciplinaryLevel,
  type InspectionResult,
  type SafetyEventKind,
  type SafetyEventStatus,
  type SafetySeverity,
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
  /** The event's id; `earlier` for the points of events not read yet. */
  id: string;
  points: number;
  /** Share of the filled length, 0 to 1. */
  share: number;
  /** Null on the `earlier` segment. */
  occurredAt: number | null;
  rollsOffAt: number | null;
};

/** The id of the segment that holds the points of events past the pages read so far. */
export const EARLIER_SEGMENT = "earlier";

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
 * to the scorecard's active points. One segment per loaded event still carrying points, oldest
 * first; points the scorecard counts on events not read yet (an older page) are one `earlier`
 * segment ahead of them, so the shares always add up to the server's figure.
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
  const loaded = carrying.reduce((sum, event) => sum + event.activePoints, 0);
  const whole = Math.max(points, loaded);
  const earlier = whole - loaded;
  const segments: GaugeSegment[] = carrying.map((event) => ({
    id: event.id,
    points: event.activePoints,
    share: whole > 0 ? event.activePoints / whole : 0,
    occurredAt: event.occurredAt,
    rollsOffAt: event.pointsExpireAt ?? null,
  }));
  if (earlier > 0) {
    segments.unshift({
      id: EARLIER_SEGMENT,
      points: earlier,
      share: earlier / whole,
      occurredAt: null,
      rollsOffAt: null,
    });
  }
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

/** The event in a few words: "Level 2 inspection — failed", "Preventable accident". */
export function safetyEventHeadline(
  event: Pick<
    WorkerSafetyEventRow,
    "kind" | "severity" | "preventable" | "inspectionResult" | "inspectionLevel"
  >,
): string {
  return describeSafetyEvent({
    kind: event.kind as SafetyEventKind,
    severity: event.severity as SafetySeverity,
    preventable: event.preventable,
    inspectionResult: (event.inspectionResult ?? null) as InspectionResult | null,
    inspectionLevel: event.inspectionLevel ?? null,
  });
}

/**
 * A fine or cost the server sends as decimal text, in minor units; null when there is none.
 * Rounded to the cent once, here, so every figure after it is integer money.
 */
export function eventAmountMinor(value: string | null | undefined): number | null {
  if (value == null || value.trim() === "") return null;
  const amount = Number(value);
  if (!Number.isFinite(amount)) return null;
  const minor = Math.round(amount * 100);
  return minor !== 0 ? minor : null;
}

/** A fine or cost as money in the reader's locale; null when there is none. */
export function formatEventAmount(value: string | null | undefined): string | null {
  const minor = eventAmountMinor(value);
  return minor === null ? null : formatMinor(minor);
}

/**
 * What an action's line says under its level. The stair's tooltips read the ladder's active
 * actions, which carry fewer fields than a row of the list; what is missing is left out.
 */
export type ActionSummaryFields = Pick<
  WorkerDisciplinaryActionRow,
  "status" | "issuedAt" | "expiresAt"
> &
  Partial<
    Pick<
      WorkerDisciplinaryActionRow,
      "rescindReason" | "issuedBy" | "suspensionDays" | "acknowledgedAt"
    >
  >;

export function actionSummary(action: ActionSummaryFields, t: TranslateFn): string {
  if (action.status === "Rescinded") {
    return action.rescindReason ? t("Rescinded: {0}", action.rescindReason) : t("Rescinded");
  }
  const parts = [
    action.issuedBy?.name
      ? t("{0} by {1}", formatUnixDateMedium(action.issuedAt), action.issuedBy.name)
      : formatUnixDateMedium(action.issuedAt),
    action.suspensionDays
      ? t("{0, plural, one {# day off} other {# days off}}", action.suspensionDays)
      : null,
    action.expiresAt
      ? action.status === "Expired"
        ? t("rolled off {0}", formatUnixDateMedium(action.expiresAt))
        : t("rolls off {0}", formatUnixDateMedium(action.expiresAt))
      : null,
    action.status === "Active" && action.acknowledgedAt !== undefined
      ? action.acknowledgedAt
        ? t("Acknowledged")
        : t("waiting for the acknowledgement")
      : null,
  ].filter(Boolean);
  return parts.join(" · ");
}
