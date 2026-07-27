import type { BadgeVariant } from "@trenova/shared/types/badge";

/**
 * The console's shared vocabulary. Urgency, availability, verdict, and severity each read
 * the same way everywhere they appear, so a dispatcher learns the colors once.
 */

export const URGENCY_ORDER = ["Late", "Now", "Today", "Tomorrow", "Planned"] as const;

export type UrgencyBucket = (typeof URGENCY_ORDER)[number];

export const URGENCY_META: Record<
  UrgencyBucket,
  { label: string; description: string; variant: BadgeVariant }
> = {
  Late: {
    label: "Late",
    description: "Pickup window already open, still uncovered",
    variant: "inactive",
  },
  Now: {
    label: "Next 4 hours",
    description: "Needs a driver now",
    variant: "warning",
  },
  Today: { label: "Today", description: "Picks up later today", variant: "info" },
  Tomorrow: { label: "Tomorrow", description: "Picks up tomorrow", variant: "purple" },
  Planned: { label: "Planned", description: "Further out", variant: "outline" },
};

export function urgencyMeta(value: string) {
  return URGENCY_META[value as UrgencyBucket] ?? URGENCY_META.Planned;
}

export const AVAILABILITY_META: Record<string, { label: string; variant: BadgeVariant }> = {
  Open: { label: "Open", variant: "active" },
  Finishing: { label: "Finishing", variant: "warning" },
  Working: { label: "Working", variant: "info" },
  Blocked: { label: "Blocked", variant: "inactive" },
  TimeOff: { label: "Time off", variant: "purple" },
};

export function availabilityMeta(value: string) {
  return AVAILABILITY_META[value] ?? { label: value, variant: "outline" as BadgeVariant };
}

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

/**
 * Regulatory HOS ceilings, used only to size the clock bars. The remaining values
 * themselves always come from the telematics feed.
 */
export const HOS_LIMITS_MS = {
  drive: 11 * 3600 * 1000,
  shift: 14 * 3600 * 1000,
  cycle: 70 * 3600 * 1000,
} as const;

export function scoreTone(score: number): string {
  if (score >= 75) return "text-green-600 dark:text-green-400";
  if (score >= 50) return "text-amber-600 dark:text-amber-400";
  return "text-muted-foreground";
}

export function formatMiles(value: number | null | undefined): string {
  if (value === null || value === undefined) return "—";
  return `${Math.round(value).toLocaleString()} mi`;
}

export function formatMoney(value: number | null | undefined): string {
  if (value === null || value === undefined) return "—";
  return value.toLocaleString(undefined, {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 0,
  });
}

/**
 * Renders a countdown the way a dispatcher says it out loud: "40m late", "in 3h 10m".
 */
export function formatMinutesToPickup(minutes: number): string {
  if (minutes === 0) return "due now";

  const late = minutes < 0;
  const total = Math.abs(minutes);
  const hours = Math.floor(total / 60);
  const remainder = total % 60;

  const span = hours > 0 ? `${hours}h ${remainder}m` : `${remainder}m`;
  return late ? `${span} late` : `in ${span}`;
}
