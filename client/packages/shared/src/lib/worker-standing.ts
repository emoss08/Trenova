import {
  WORKER_STANDING_LABELS,
  type ConcernSeverity,
  type WorkerStanding,
} from "../types/worker-overview";

export type StandingMeta = {
  label: string;
  badgeVariant: "active" | "warning" | "inactive" | "outline";
  textClass: string;
  ringTone: "success" | "warning" | "critical";
  /** A one-line reading of the verdict, for the summary line under the badge. */
  blurb: string;
  /** Best first, so a roster can be sorted worst-last. */
  rank: number;
};

const STANDING_META: Record<WorkerStanding, StandingMeta> = {
  Good: {
    label: WORKER_STANDING_LABELS.Good,
    badgeVariant: "active",
    textClass: "text-green-600 dark:text-green-400",
    ringTone: "success",
    blurb: "Nothing on the record needs attention.",
    rank: 0,
  },
  Watch: {
    label: WORKER_STANDING_LABELS.Watch,
    badgeVariant: "warning",
    textClass: "text-amber-600 dark:text-amber-400",
    ringTone: "warning",
    blurb: "Something is coming due soon.",
    rank: 1,
  },
  AtRisk: {
    label: WORKER_STANDING_LABELS.AtRisk,
    badgeVariant: "inactive",
    textClass: "text-red-600 dark:text-red-400",
    ringTone: "critical",
    blurb: "Something on the record has already lapsed.",
    rank: 2,
  },
  Blocked: {
    label: WORKER_STANDING_LABELS.Blocked,
    badgeVariant: "inactive",
    textClass: "text-red-600 dark:text-red-400",
    ringTone: "critical",
    blurb: "The worker cannot be put on a load today.",
    rank: 3,
  },
};

const FALLBACK: StandingMeta = {
  label: "Unknown",
  badgeVariant: "outline",
  textClass: "text-muted-foreground",
  ringTone: "warning",
  blurb: "The standing could not be worked out.",
  rank: 4,
};

export function workerStandingMeta(standing: WorkerStanding | string): StandingMeta {
  return STANDING_META[standing as WorkerStanding] ?? FALLBACK;
}

export type ConcernSeverityMeta = {
  label: string;
  textClass: string;
  borderClass: string;
  /** A marker dot: the one place a concern row spends colour. */
  dotClass: string;
  rank: number;
};

const SEVERITY_META: Record<ConcernSeverity, ConcernSeverityMeta> = {
  Critical: {
    label: "Needs action",
    textClass: "text-red-600 dark:text-red-400",
    borderClass: "border-red-500/40 bg-red-500/5",
    dotClass: "bg-destructive",
    rank: 0,
  },
  Warning: {
    label: "Coming due",
    textClass: "text-amber-600 dark:text-amber-400",
    borderClass: "border-amber-500/40 bg-amber-500/5",
    dotClass: "bg-warning",
    rank: 1,
  },
  Info: {
    label: "For information",
    textClass: "text-muted-foreground",
    borderClass: "border-border",
    dotClass: "bg-muted-foreground/50",
    rank: 2,
  },
};

const SEVERITY_FALLBACK: ConcernSeverityMeta = {
  label: "Note",
  textClass: "text-muted-foreground",
  borderClass: "border-border",
  dotClass: "bg-muted-foreground/50",
  rank: 3,
};

export function concernSeverityMeta(severity: ConcernSeverity | string): ConcernSeverityMeta {
  return SEVERITY_META[severity as ConcernSeverity] ?? SEVERITY_FALLBACK;
}

/**
 * Groups concerns by severity while preserving the server's worst-first order
 * inside each group, so the panel can render one block per severity without
 * re-sorting and disagreeing with the standing.
 */
export function groupConcernsBySeverity<T extends { severity: string }>(
  concerns: readonly T[],
): { severity: ConcernSeverity; items: T[] }[] {
  const order: ConcernSeverity[] = ["Critical", "Warning", "Info"];
  return order
    .map((severity) => ({
      severity,
      items: concerns.filter((concern) => concern.severity === severity),
    }))
    .filter((group) => group.items.length > 0);
}
