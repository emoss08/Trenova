import {
  DISCIPLINARY_LEVEL_LABELS,
  INSPECTION_RESULT_LABELS,
  SAFETY_EVENT_KIND_LABELS,
  SAFETY_SEVERITY_LABELS,
  type DisciplinaryLevel,
  type InspectionResult,
  type SafetyEventKind,
  type SafetyRating,
  type SafetySeverity,
} from "../types/worker-safety";

export type SafetyRatingMeta = {
  label: string;
  badgeVariant: "active" | "warning" | "inactive" | "outline";
  textClass: string;
  ringTone: "success" | "warning" | "critical";
  /** Best first, so a list of workers can be sorted worst-last. */
  rank: number;
};

const RATING_META: Record<SafetyRating, SafetyRatingMeta> = {
  Excellent: {
    label: "Excellent",
    badgeVariant: "active",
    textClass: "text-green-600 dark:text-green-400",
    ringTone: "success",
    rank: 0,
  },
  Good: {
    label: "Good",
    badgeVariant: "outline",
    textClass: "text-sky-600 dark:text-sky-400",
    ringTone: "success",
    rank: 1,
  },
  Watch: {
    label: "Watch",
    badgeVariant: "warning",
    textClass: "text-amber-600 dark:text-amber-400",
    ringTone: "warning",
    rank: 2,
  },
  AtRisk: {
    label: "At risk",
    badgeVariant: "inactive",
    textClass: "text-red-600 dark:text-red-400",
    ringTone: "critical",
    rank: 3,
  },
};

export function safetyRatingMeta(rating: SafetyRating): SafetyRatingMeta {
  return RATING_META[rating] ?? RATING_META.Watch;
}

/** Maps the 0–100 score onto a 0–1 gauge, clamped. */
export function scoreRingValue(score: number): number {
  if (!Number.isFinite(score)) return 0;
  return Math.min(1, Math.max(0, score / 100));
}

export type DisciplinaryLevelMeta = {
  label: string;
  /** 1 for Coaching through 6 for Termination, mirroring the server's ladder. */
  rank: number;
  endsEmployment: boolean;
  textClass: string;
};

const LADDER: DisciplinaryLevel[] = [
  "Coaching",
  "VerbalWarning",
  "WrittenWarning",
  "FinalWarning",
  "Suspension",
  "Termination",
];

const LEVEL_TONE: Record<DisciplinaryLevel, string> = {
  Coaching: "text-sky-600 dark:text-sky-400",
  VerbalWarning: "text-sky-600 dark:text-sky-400",
  WrittenWarning: "text-amber-600 dark:text-amber-400",
  FinalWarning: "text-amber-600 dark:text-amber-400",
  Suspension: "text-red-600 dark:text-red-400",
  Termination: "text-red-600 dark:text-red-400",
};

export function disciplinaryLevelMeta(level: DisciplinaryLevel): DisciplinaryLevelMeta {
  const rank = LADDER.indexOf(level) + 1;
  return {
    label: DISCIPLINARY_LEVEL_LABELS[level] ?? level,
    rank,
    endsEmployment: level === "Termination",
    textClass: LEVEL_TONE[level] ?? "text-muted-foreground",
  };
}

/**
 * The rung above the one given — what the office would normally issue next.
 * Mirrors the server so the form and the ladder agree.
 */
export function nextDisciplinaryLevel(
  level: DisciplinaryLevel | null | undefined,
): DisciplinaryLevel {
  if (!level) return "Coaching";
  const index = LADDER.indexOf(level);
  if (index < 0) return "Coaching";
  return LADDER[Math.min(index + 1, LADDER.length - 1)];
}

export function describeSeverity(severity: SafetySeverity): string {
  return SAFETY_SEVERITY_LABELS[severity] ?? severity;
}

export type SafetyEventDescriptor = {
  kind: SafetyEventKind;
  severity: SafetySeverity;
  preventable?: boolean | null;
  inspectionResult?: InspectionResult | null;
  inspectionLevel?: number | null;
};

/**
 * One sentence for what an event was. Accidents say whether they were
 * preventable, inspections say the level and the outcome; near misses need
 * no severity because they did not happen.
 */
export function describeSafetyEvent(event: SafetyEventDescriptor): string {
  const kind = SAFETY_EVENT_KIND_LABELS[event.kind] ?? event.kind;
  if (event.kind === "NearMiss") return kind;
  if (event.kind === "Inspection") {
    const level = event.inspectionLevel ? `Level ${event.inspectionLevel} ` : "";
    const outcome = event.inspectionResult
      ? ` — ${(INSPECTION_RESULT_LABELS[event.inspectionResult] ?? "").toLowerCase()}`
      : "";
    // The level reads as the start of the phrase when there is one, so the
    // kind only drops its capital behind it.
    return `${level}${level ? kind.toLowerCase() : kind}${outcome}`;
  }
  const severity = describeSeverity(event.severity);
  const base = `${severity} ${kind.toLowerCase()}`;
  if (event.kind !== "Accident") return base;
  return `${base}, ${event.preventable ? "preventable" : "non-preventable"}`;
}

/** Clean-inspection record for the last twelve months. */
export function summariseInspections(counts: {
  inspections: number;
  inspectionsPassed: number;
}): string {
  if (counts.inspections <= 0) return "No inspections in the last year";
  const percent = Math.round((counts.inspectionsPassed / counts.inspections) * 100);
  return `${counts.inspectionsPassed} of ${counts.inspections} clean (${percent}%)`;
}
