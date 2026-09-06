import type { WorkerTrainingHealth } from "../types/worker-training";

export type TrainingHealthMeta = {
  label: string;
  badgeVariant: "active" | "warning" | "inactive" | "secondary";
  textClass: string;
  ringClass: string;
  dotClass: string;
  /** Weight used to sort the worst slot first. */
  rank: number;
  /** Whether a required course in this state leaves the worker unqualified. */
  blocks: boolean;
};

const HEALTH_META: Record<WorkerTrainingHealth, TrainingHealthMeta> = {
  Missing: {
    label: "Missing",
    badgeVariant: "secondary",
    textClass: "text-muted-foreground",
    ringClass: "border-dashed border-muted-foreground/40",
    dotClass: "bg-muted-foreground/60",
    rank: 0,
    blocks: true,
  },
  Failed: {
    label: "Failed",
    badgeVariant: "inactive",
    textClass: "text-red-600 dark:text-red-400",
    ringClass: "border-red-500/40 bg-red-500/5",
    dotClass: "bg-red-500",
    rank: 1,
    blocks: true,
  },
  Expired: {
    label: "Expired",
    badgeVariant: "inactive",
    textClass: "text-red-600 dark:text-red-400",
    ringClass: "border-red-500/40 bg-red-500/5",
    dotClass: "bg-red-500",
    rank: 2,
    blocks: true,
  },
  Overdue: {
    label: "Overdue",
    badgeVariant: "inactive",
    textClass: "text-red-600 dark:text-red-400",
    ringClass: "border-red-500/40 bg-red-500/5",
    dotClass: "bg-red-500",
    rank: 3,
    blocks: true,
  },
  DueSoon: {
    label: "Due soon",
    badgeVariant: "warning",
    textClass: "text-amber-600 dark:text-amber-400",
    ringClass: "border-amber-500/40 bg-amber-500/5",
    dotClass: "bg-amber-500",
    rank: 4,
    blocks: false,
  },
  ExpiringSoon: {
    label: "Expiring soon",
    badgeVariant: "warning",
    textClass: "text-amber-600 dark:text-amber-400",
    ringClass: "border-amber-500/40 bg-amber-500/5",
    dotClass: "bg-amber-500",
    rank: 5,
    blocks: false,
  },
  Scheduled: {
    label: "Scheduled",
    badgeVariant: "secondary",
    textClass: "text-sky-600 dark:text-sky-400",
    ringClass: "border-sky-500/30",
    dotClass: "bg-sky-500",
    rank: 6,
    blocks: false,
  },
  Current: {
    label: "Current",
    badgeVariant: "active",
    textClass: "text-green-600 dark:text-green-400",
    ringClass: "border-border",
    dotClass: "bg-green-500",
    rank: 7,
    blocks: false,
  },
};

export function trainingHealthMeta(health: WorkerTrainingHealth): TrainingHealthMeta {
  return HEALTH_META[health] ?? HEALTH_META.Missing;
}

export type TrainingTiming = {
  health: WorkerTrainingHealth;
  daysUntilDue?: number | null;
  daysUntilExpiry?: number | null;
};

/**
 * One phrase for where a course stands in time. Open work talks about the due
 * date, completions about expiry; both mirror the server's whole-day maths so
 * 0 is today and negatives are already past.
 */
export function describeTrainingTiming(timing: TrainingTiming): string {
  switch (timing.health) {
    case "Missing":
      return "Not assigned";
    case "Failed":
      return "Failed — retake needed";
    case "Scheduled":
    case "DueSoon":
    case "Overdue":
      return describeDue(timing.daysUntilDue);
    case "Current":
    case "ExpiringSoon":
    case "Expired":
      return describeExpiry(timing.daysUntilExpiry);
    default:
      return "";
  }
}

function describeDue(days: number | null | undefined): string {
  if (days == null) return "No due date";
  if (days === 0) return "Due today";
  if (days === 1) return "Due tomorrow";
  if (days < 0) return `Overdue by ${-days} day${days === -1 ? "" : "s"}`;
  return `Due in ${days} days`;
}

function describeExpiry(days: number | null | undefined): string {
  if (days == null) return "Does not expire";
  if (days === 0) return "Expires today";
  if (days === 1) return "Expires tomorrow";
  if (days === -1) return "Expired yesterday";
  if (days < 0) return `Expired ${-days} days ago`;
  return `Expires in ${days} days`;
}

export type TrainingSortable = {
  name: string;
  health: WorkerTrainingHealth;
  required: boolean;
};

/** Worst first, required before optional within the same state, then by name. */
export function sortTrainingWorstFirst<T extends TrainingSortable>(items: readonly T[]): T[] {
  return [...items].sort((a, b) => {
    const rank = trainingHealthMeta(a.health).rank - trainingHealthMeta(b.health).rank;
    if (rank !== 0) return rank;
    if (a.required !== b.required) return a.required ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
}

export type TrainingProgress = { satisfied: number; required: number; ratio: number };

/** How much of the required matrix is satisfied, for gauges and chips. */
export function trainingProgress(
  items: readonly { required: boolean; health: WorkerTrainingHealth }[],
): TrainingProgress {
  let required = 0;
  let satisfied = 0;
  for (const item of items) {
    if (!item.required) continue;
    required += 1;
    if (item.health === "Current" || item.health === "ExpiringSoon") satisfied += 1;
  }
  return { satisfied, required, ratio: required === 0 ? 1 : satisfied / required };
}
