import type { BadgeVariant } from "@trenova/shared/types/badge";

/**
 * The console's shared vocabulary. Urgency, availability, verdict, and severity each read
 * the same way everywhere they appear, so a dispatcher learns the colors once.
 */

export const URGENCY_ORDER = ["Late", "Now", "Today", "Tomorrow", "Planned"] as const;

export type UrgencyBucket = (typeof URGENCY_ORDER)[number];

export const URGENCY_META: Record<
  UrgencyBucket,
  { label: string; description: string; variant: BadgeVariant; dotClass: string }
> = {
  Late: {
    label: "Late",
    description: "Pickup window already open",
    variant: "inactive",
    dotClass: "bg-destructive",
  },
  Now: {
    label: "Next 4 Hours",
    description: "Needs a driver now",
    variant: "warning",
    dotClass: "bg-warning",
  },
  Today: {
    label: "Today",
    description: "Picks up later today",
    variant: "info",
    dotClass: "bg-blue-500",
  },
  Tomorrow: {
    label: "Tomorrow",
    description: "Picks up tomorrow",
    variant: "purple",
    dotClass: "bg-purple-500",
  },
  Planned: {
    label: "Planned",
    description: "Further out",
    variant: "outline",
    dotClass: "bg-muted-foreground/50",
  },
};

export function urgencyMeta(value: string) {
  return URGENCY_META[value as UrgencyBucket] ?? URGENCY_META.Planned;
}

export function groupMovesByUrgency<T extends { urgency: string }>(
  moves: readonly T[],
): Map<UrgencyBucket, T[]> {
  const buckets = new Map<UrgencyBucket, T[]>();
  for (const bucket of URGENCY_ORDER) {
    buckets.set(bucket, []);
  }
  for (const move of moves) {
    const bucket = URGENCY_META[move.urgency as UrgencyBucket]
      ? (move.urgency as UrgencyBucket)
      : "Planned";
    buckets.get(bucket)?.push(move);
  }
  return buckets;
}

export const AVAILABILITY_META: Record<
  string,
  { label: string; variant: BadgeVariant; dotClass: string; labelClass: string }
> = {
  Open: {
    label: "Open",
    variant: "active",
    dotClass: "bg-success",
    labelClass: "text-success",
  },
  Finishing: {
    label: "Finishing",
    variant: "warning",
    dotClass: "bg-warning",
    labelClass: "text-warning",
  },
  Working: {
    label: "Working",
    variant: "info",
    dotClass: "bg-blue-500",
    labelClass: "text-blue-600 dark:text-blue-400",
  },
  Blocked: {
    label: "Blocked",
    variant: "inactive",
    dotClass: "bg-destructive",
    labelClass: "text-destructive",
  },
  TimeOff: {
    label: "Time off",
    variant: "purple",
    dotClass: "bg-purple-500",
    labelClass: "text-purple-600 dark:text-purple-400",
  },
};

export function availabilityMeta(value: string) {
  return (
    AVAILABILITY_META[value] ?? {
      label: value,
      variant: "outline" as BadgeVariant,
      dotClass: "bg-muted-foreground/50",
      labelClass: "text-muted-foreground",
    }
  );
}

export type CapacityFilter = "all" | "open" | "finishing" | "working" | "blocked" | "timeOff";

/**
 * The capacity rail's lenses. They live here rather than on the rail because the URL
 * state and the summary strip both address them without rendering the rail.
 */
export const CAPACITY_FILTERS: readonly {
  id: CapacityFilter;
  label: string;
  availability?: string;
}[] = [
  { id: "all", label: "All" },
  { id: "open", label: "Open", availability: "Open" },
  { id: "finishing", label: "Finishing", availability: "Finishing" },
  { id: "working", label: "Working", availability: "Working" },
  { id: "blocked", label: "Blocked", availability: "Blocked" },
  { id: "timeOff", label: "Time off", availability: "TimeOff" },
];

/** Open capacity first, blocked last — the order a dispatcher reads the rail in. */
export const AVAILABILITY_SORT_RANK: Record<string, number> = {
  Open: 0,
  Finishing: 1,
  Working: 2,
  TimeOff: 3,
  Blocked: 4,
};

/**
 * Verdicts intentionally reuse the vocabulary the shipment assignment dialog already
 * shows, so the console and that dialog never disagree about what "tight" means.
 */
export const VERDICT_META: Record<string, { label: string; variant: BadgeVariant }> = {
  feasible: { label: "Feasible", variant: "active" },
  tight: { label: "Tight", variant: "warning" },
  infeasible: { label: "Infeasible", variant: "inactive" },
  unknown: { label: "Unknown", variant: "outline" },
};

export function verdictMeta(value: string) {
  return VERDICT_META[value] ?? VERDICT_META.unknown;
}

export const SEVERITY_META: Record<string, { label: string; variant: BadgeVariant }> = {
  Block: { label: "Blocking", variant: "inactive" },
  Warn: { label: "Warning", variant: "warning" },
  Info: { label: "Info", variant: "outline" },
};

export function severityMeta(value: string) {
  return SEVERITY_META[value] ?? SEVERITY_META.Info;
}

export const DUTY_STATUS_META: Record<string, { label: string; variant: BadgeVariant }> = {
  driving: { label: "Driving", variant: "info" },
  onDuty: { label: "On duty", variant: "warning" },
  offDuty: { label: "Off duty", variant: "outline" },
  sleeperBed: { label: "Sleeper", variant: "purple" },
  yardMove: { label: "Yard move", variant: "teal" },
  personalConveyance: { label: "Personal", variant: "teal" },
};

export function dutyStatusMeta(value: string) {
  return DUTY_STATUS_META[value];
}

export function scoreTone(score: number): string {
  if (score >= 75) return "text-green-600 dark:text-green-400";
  if (score >= 50) return "text-amber-600 dark:text-amber-400";
  return "text-muted-foreground";
}

export function formatMiles(value: number | null | undefined): string {
  if (value === null || value === undefined) return "—";
  return `${Math.round(value).toLocaleString()} mi`;
}

/**
 * Renders a span of minutes the way a dispatcher says it out loud, rolling hours into
 * days once they stop being scannable: "40m", "3h 10m", "11d 12h".
 */
export function formatMinutesSpan(minutes: number): string {
  const total = Math.abs(Math.round(minutes));
  const days = Math.floor(total / 1440);
  if (days > 0) {
    return `${days}d ${Math.floor((total % 1440) / 60)}h`;
  }
  const hours = Math.floor(total / 60);
  return hours > 0 ? `${hours}h ${total % 60}m` : `${total}m`;
}

/**
 * Renders a countdown the way a dispatcher says it out loud: "40m late", "in 3h 10m".
 */
export function formatMinutesToPickup(minutes: number): string {
  if (minutes === 0) return "due now";
  const span = formatMinutesSpan(minutes);
  return minutes < 0 ? `${span} late` : `in ${span}`;
}

/**
 * The rest plan the projection engine says would make a trip legal, phrased for the
 * dispatcher who has to communicate it to the driver.
 */
export const HOS_STRATEGY_META: Record<string, { label: string; phrase: string }> = {
  currentClocks: { label: "Current clocks", phrase: "Legal on current clocks" },
  splitSleeper: {
    label: "Split sleeper",
    phrase: "Legal with a split sleeper berth pairing",
  },
  tenHourReset: { label: "10-hour reset", phrase: "Legal after a 10-hour reset" },
  restart34: { label: "34-hour restart", phrase: "Legal after a 34-hour cycle restart" },
};

export function hosStrategyMeta(value: string | null | undefined) {
  if (!value) return null;
  return HOS_STRATEGY_META[value] ?? null;
}

export function workerInitials(firstName: string, lastName: string): string {
  const first = firstName.trim().charAt(0);
  const last = lastName.trim().charAt(0);
  const initials = `${first}${last}`.toUpperCase();
  return initials || "?";
}
