import { defineLabels, type LabelMap } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import type { BadgeVariant } from "@trenova/shared/types/badge";

/**
 * The console's shared vocabulary. Urgency, availability, verdict, and severity each read
 * the same way everywhere they appear, so a dispatcher learns the colors once.
 */

export const URGENCY_ORDER = ["Late", "Now", "Today", "Tomorrow", "Planned"] as const;

export type UrgencyBucket = (typeof URGENCY_ORDER)[number];

/**
 * The groups follow the server's rule (`dispatchconsoleservice/decorate.go`): Late once the
 * window has closed, Due now while it is open or opens within four hours, then rolling
 * 24- and 48-hour spans. The windows roll with the clock, so there is no "today".
 */
export const URGENCY_META: Record<
  UrgencyBucket,
  {
    label: string;
    description: string;
    variant: BadgeVariant;
    dotClass: string;
    /** The colour token a move in this group is tinted in on the timeline. */
    colorVar: string;
  }
> = {
  Late: {
    label: "Late",
    description: "pickup window closed, no arrival",
    variant: "danger",
    dotClass: "bg-destructive",
    colorVar: "--danger",
  },
  Now: {
    label: "Due now",
    description: "window open or opens within 4 hours",
    variant: "warning",
    dotClass: "bg-warning",
    colorVar: "--warning",
  },
  Today: {
    label: "Within 24 hours",
    description: "window opens in 4 to 24 hours",
    variant: "info",
    dotClass: "bg-info",
    colorVar: "--info",
  },
  Tomorrow: {
    label: "Within 48 hours",
    description: "window opens in 24 to 48 hours",
    variant: "info",
    dotClass: "bg-accent-violet",
    colorVar: "--accent-violet",
  },
  Planned: {
    label: "Later",
    description: "further out, or no appointment yet",
    variant: "neutral",
    dotClass: "bg-muted-foreground/50",
    colorVar: "--muted-foreground",
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
    variant: "success",
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
    dotClass: "bg-info",
    labelClass: "text-info-foreground",
  },
  Blocked: {
    label: "Blocked",
    variant: "danger",
    dotClass: "bg-destructive",
    labelClass: "text-destructive",
  },
  TimeOff: {
    label: "Time off",
    variant: "info",
    dotClass: "bg-accent-violet",
    labelClass: "text-accent-violet-on-subtle",
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
  feasible: { label: "Feasible", variant: "success" },
  tight: { label: "Tight", variant: "warning" },
  infeasible: { label: "Infeasible", variant: "danger" },
  unknown: { label: "Unknown", variant: "neutral" },
};

export function verdictMeta(value: string) {
  return VERDICT_META[value] ?? VERDICT_META.unknown;
}

export const SEVERITY_META: Record<string, { label: string; variant: BadgeVariant }> = {
  Block: { label: "Blocking", variant: "danger" },
  Warn: { label: "Warning", variant: "warning" },
  Info: { label: "Info", variant: "neutral" },
};

export function severityMeta(value: string) {
  return SEVERITY_META[value] ?? SEVERITY_META.Info;
}

export const DUTY_STATUS_META: Record<string, { label: string; variant: BadgeVariant }> = {
  driving: { label: "Driving", variant: "info" },
  onDuty: { label: "On duty", variant: "warning" },
  offDuty: { label: "Off duty", variant: "neutral" },
  sleeperBed: { label: "Sleeper", variant: "info" },
  yardMove: { label: "Yard move", variant: "info" },
  personalConveyance: { label: "Personal", variant: "info" },
};

export function dutyStatusMeta(value: string) {
  return DUTY_STATUS_META[value];
}

export function scoreTone(score: number): string {
  if (score >= 75) return "text-success-foreground";
  if (score >= 50) return "text-warning-foreground";
  return "text-muted-foreground";
}

export function formatMiles(value: number | null | undefined): string {
  if (value === null || value === undefined) return "—";
  return translate("{0} mi", Math.round(value).toLocaleString());
}

export type PickupTone = "danger" | "warning" | "neutral";

export type PickupState = {
  tone: PickupTone;
  /** What the window is doing now: "Closed 25m ago", "Open · closes in 1h 48m". */
  headline: string;
  /** The window itself, so the headline can be checked at a glance. */
  window: string;
};

type PickupMove = {
  urgency: string;
  originWindowStart: number;
  originWindowEnd?: number | null;
  originActualArrival?: number | null;
  destinationWindowStart: number;
  destinationWindowEnd?: number | null;
};

const FOUR_HOURS_SECONDS = 4 * 60 * 60;

function windowLabel(
  start: number,
  end: number | null | undefined,
  formatTime: (seconds: number) => string,
  formatClock: (seconds: number) => string,
): string {
  if (start <= 0) return translate("no appointment");
  if (!end || end <= start) return translate("window {0}", formatTime(start));
  // A window inside one day names the day once: "Nov 1, 4:31–6:31 PM".
  const sameDay = new Date(start * 1000).toDateString() === new Date(end * 1000).toDateString();
  return translate(
    "window {0}–{1}",
    formatTime(start),
    sameDay ? formatClock(end) : formatTime(end),
  );
}

/**
 * Where a move's next window stands, phrased from the server's urgency so the queue and the
 * groups never disagree. Once the truck has arrived at pickup the delivery window is the
 * one that matters, as it is on the server.
 */
export function pickupState(
  move: PickupMove,
  nowSeconds: number,
  formatTime: (seconds: number) => string,
  formatClock: (seconds: number) => string = formatTime,
): PickupState {
  const delivering = (move.originActualArrival ?? 0) > 0;
  const start = delivering ? move.destinationWindowStart : move.originWindowStart;
  const rawEnd = delivering ? move.destinationWindowEnd : move.originWindowEnd;
  const end = rawEnd && rawEnd > start ? rawEnd : start;
  const window = windowLabel(start, rawEnd, formatTime, formatClock);

  if (start <= 0) {
    return { tone: "neutral", headline: translate("No appointment"), window };
  }

  if (move.urgency === "Late" || end < nowSeconds) {
    const span = formatMinutesSpan(Math.max(1, Math.round((nowSeconds - end) / 60)));
    return {
      tone: "danger",
      headline: delivering
        ? translate("Delivery closed {0} ago", span)
        : translate("Closed {0} ago", span),
      window,
    };
  }

  if (start <= nowSeconds) {
    const span = formatMinutesSpan(Math.max(1, Math.round((end - nowSeconds) / 60)));
    return {
      tone: "warning",
      headline: delivering
        ? translate("Delivery open · closes in {0}", span)
        : translate("Open · closes in {0}", span),
      window,
    };
  }

  const untilOpen = start - nowSeconds;
  if (untilOpen <= FOUR_HOURS_SECONDS) {
    const span = formatMinutesSpan(Math.max(1, Math.round(untilOpen / 60)));
    return {
      tone: "neutral",
      headline: delivering
        ? translate("Delivery opens in {0}", span)
        : translate("Opens in {0}", span),
      window,
    };
  }

  return {
    tone: "neutral",
    headline: delivering
      ? translate("Delivery opens {0}", formatTime(start))
      : translate("Opens {0}", formatTime(start)),
    window,
  };
}

/**
 * The rest plan the projection engine says would make a trip legal, phrased for the
 * dispatcher who has to communicate it to the driver.
 */
export const HOS_STRATEGY_META: Record<string, LabelMap<{ label: string; phrase: string }>> = {
  currentClocks: defineLabels({ label: "Current clocks", phrase: "Legal on current clocks" }),
  splitSleeper: defineLabels({
    label: "Split sleeper",
    phrase: "Legal with a split sleeper berth pairing",
  }),
  tenHourReset: defineLabels({ label: "10-hour reset", phrase: "Legal after a 10-hour reset" }),
  restart34: defineLabels({
    label: "34-hour restart",
    phrase: "Legal after a 34-hour cycle restart",
  }),
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
