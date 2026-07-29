import {
  DESK_URGENCY_RANK,
  type CapKind,
  type DeskEntry,
  type DeskUrgency,
  type DetentionNotificationStatus,
  type NoticeDeliveryStatus,
  type OccurrenceStatus,
  type ScoreBand,
} from "../types/detention";
import { formatCurrency } from "./utils";

/** Plain-language names for the ceiling that stopped a charge from growing. */
export const CAP_KIND_LABEL: Record<CapKind, string> = {
  None: "No cap",
  MaxBillableMinutes: "Minutes cap",
  MaxChargePerStop: "Stop cap",
  MaxChargePerDay: "Daily cap",
  MaxChargePerShipment: "Shipment cap",
  LayoverBoundary: "Layover boundary",
};

/**
 * Formats a money delta with an explicit sign so a gain can never be misread as
 * a total. Uses a true minus sign, which lines up in tabular figures.
 */
export function formatSignedCurrency(value: number, currency = "USD"): string {
  const formatted = formatCurrency(Math.abs(value), currency);

  if (value > 0) {
    return `+${formatted}`;
  }

  if (value < 0) {
    return `−${formatted}`;
  }

  return formatted;
}

/** Green when the carrier gains, red when it loses, neutral at zero. */
export function deltaToneClass(value: number): string {
  if (value > 0) {
    return "text-emerald-600 dark:text-emerald-400";
  }

  if (value < 0) {
    return "text-red-600 dark:text-red-400";
  }

  return "text-muted-foreground";
}

/** Formats a minute count the way a dispatcher reads a clock. */
export function formatDetentionMinutes(minutes: number): string {
  const abs = Math.abs(minutes);

  if (abs < 60) {
    return `${abs}m`;
  }

  const hours = Math.floor(abs / 60);
  const rest = abs % 60;

  return rest === 0 ? `${hours}h` : `${hours}h ${rest}m`;
}

/** Formats a countdown, making an elapsed deadline read as overdue. */
export function formatCountdown(minutes: number | null | undefined): string {
  if (minutes === null || minutes === undefined) {
    return "—";
  }

  if (minutes <= 0) {
    return `${formatDetentionMinutes(minutes)} overdue`;
  }

  return `in ${formatDetentionMinutes(minutes)}`;
}

export const URGENCY_LABEL: Record<DeskUrgency, string> = {
  NoticeOverdue: "Notice overdue",
  NoticeDueSoon: "Notice due soon",
  Accruing: "Accruing",
  FreeTimeEnding: "Free time ending",
  Lost: "Not collectable",
  Normal: "Within free time",
};

export type UrgencyStyle = {
  /** The accrued portion of the clock bar. */
  bar: string;
  dot: string;
  text: string;
};

/**
 * Tailwind classes per urgency. The desk stays deliberately monochrome: colour
 * is spent only on the notice window, because that is the one state where
 * acting in the next few minutes changes whether the money is collectable at
 * all. Everything else is stated in neutral ink and read from the numbers.
 */
export const URGENCY_STYLES: Record<DeskUrgency, UrgencyStyle> = {
  NoticeOverdue: {
    bar: "bg-amber-500",
    dot: "bg-amber-500",
    text: "text-amber-700 dark:text-amber-500",
  },
  NoticeDueSoon: {
    bar: "bg-amber-400",
    dot: "bg-amber-400",
    text: "text-amber-700 dark:text-amber-500",
  },
  Accruing: {
    bar: "bg-foreground/70",
    dot: "bg-foreground/70",
    text: "text-foreground",
  },
  FreeTimeEnding: {
    bar: "bg-foreground/40",
    dot: "bg-foreground/40",
    text: "text-muted-foreground",
  },
  Lost: {
    bar: "bg-muted-foreground/30",
    dot: "bg-muted-foreground/30",
    text: "text-muted-foreground",
  },
  Normal: {
    bar: "bg-foreground/25",
    dot: "bg-foreground/20",
    text: "text-muted-foreground",
  },
};

/** What the notice state means for the charge, said the way a clerk would say it. */
export const NOTIFICATION_STATUS_LABEL: Record<DetentionNotificationStatus, string> = {
  NotRequired: "No notice required",
  Pending: "Notice not sent",
  Sent: "Notice sent",
  Late: "Notice sent late",
  Missed: "Notice missed",
  Failed: "Notice failed",
};

export const SCORE_BAND_STYLES: Record<ScoreBand, string> = {
  Strong: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  Adequate: "bg-sky-500/15 text-sky-700 dark:text-sky-400",
  Weak: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
  AtRisk: "bg-red-500/15 text-red-700 dark:text-red-400",
};

/** Maps a 0-100 collectability score onto its qualitative band. */
export function scoreBand(score: number): ScoreBand {
  if (score >= 85) return "Strong";
  if (score >= 65) return "Adequate";
  if (score >= 40) return "Weak";
  return "AtRisk";
}

export const OCCURRENCE_STATUS_LABEL: Record<OccurrenceStatus, string> = {
  Accruing: "Accruing",
  Pending: "Pending review",
  Approved: "Approved",
  Billed: "Billed",
  Waived: "Waived",
  Disputed: "Disputed",
  NotBillable: "Not billable",
};

/**
 * Status as a single dot. Amber marks the two states that are waiting on a
 * person — an unapproved charge and a live dispute; everything else is neutral
 * ink, weighted by how settled it is.
 */
export const OCCURRENCE_STATUS_DOT: Record<OccurrenceStatus, string> = {
  Accruing: "bg-foreground/70",
  Pending: "bg-amber-500",
  Approved: "bg-foreground/70",
  Billed: "bg-foreground/40",
  Waived: "bg-muted-foreground/30",
  Disputed: "bg-amber-500",
  NotBillable: "bg-muted-foreground/30",
};

/** Delivery state as a dot: neutral while in flight, red only on a real failure. */
export const NOTICE_DELIVERY_DOT: Record<NoticeDeliveryStatus, string> = {
  Queued: "bg-muted-foreground/30",
  Sent: "bg-foreground/40",
  Delivered: "bg-foreground/70",
  Opened: "bg-foreground/70",
  Bounced: "bg-red-500",
  Failed: "bg-red-500",
};

export const OCCURRENCE_STATUS_STYLES: Record<OccurrenceStatus, string> = {
  Accruing: "bg-red-500/15 text-red-700 dark:text-red-400",
  Pending: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
  Approved: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  Billed: "bg-sky-500/15 text-sky-700 dark:text-sky-400",
  Waived: "bg-muted text-muted-foreground",
  Disputed: "bg-orange-500/15 text-orange-700 dark:text-orange-400",
  NotBillable: "bg-muted text-muted-foreground",
};

/**
 * Sorts the desk so the thing a dispatcher can still save comes first, then the
 * largest exposure within the same urgency band.
 */
export function sortDeskEntries(entries: DeskEntry[]): DeskEntry[] {
  return [...entries].sort((a, b) => {
    const rank = DESK_URGENCY_RANK[a.urgency] - DESK_URGENCY_RANK[b.urgency];
    if (rank !== 0) {
      return rank;
    }

    if (a.amountAtRisk !== b.amountAtRisk) {
      return b.amountAtRisk - a.amountAtRisk;
    }

    return a.occurrence.freeTimeExpiresAt - b.occurrence.freeTimeExpiresAt;
  });
}

/** The lanes the desk can be narrowed to, in the order a clerk works them. */
export const DESK_FILTERS = [
  { id: "all", label: "All", hint: "Every stop a driver is sitting on right now" },
  {
    id: "notice",
    label: "Notice",
    hint: "The contractual notice has not gone out yet — acting here decides whether the money is collectable at all",
  },
  { id: "accruing", label: "Accruing", hint: "Past free time and billing by the minute" },
  { id: "free", label: "Free time", hint: "Still inside the granted free time" },
  { id: "lost", label: "Lost", hint: "Past the notice deadline — no longer defensible" },
] as const;

export type DeskFilterId = (typeof DESK_FILTERS)[number]["id"];

export const DESK_FILTER_IDS: readonly DeskFilterId[] = DESK_FILTERS.map((filter) => filter.id);

/** Every urgency belongs to exactly one lane, so the counts always sum to the total. */
const URGENCY_FILTER: Record<DeskUrgency, Exclude<DeskFilterId, "all">> = {
  NoticeOverdue: "notice",
  NoticeDueSoon: "notice",
  Accruing: "accruing",
  FreeTimeEnding: "free",
  Normal: "free",
  Lost: "lost",
};

export function matchesDeskFilter(entry: DeskEntry, filter: DeskFilterId): boolean {
  return filter === "all" || URGENCY_FILTER[entry.urgency] === filter;
}

export type DeskFilterCounts = Record<DeskFilterId, number>;

export function countDeskFilters(entries: DeskEntry[]): DeskFilterCounts {
  const counts: DeskFilterCounts = {
    all: entries.length,
    notice: 0,
    accruing: 0,
    free: 0,
    lost: 0,
  };

  for (const entry of entries) {
    counts[URGENCY_FILTER[entry.urgency]] += 1;
  }

  return counts;
}

/**
 * Matches the four things a clerk actually knows when they go looking for a
 * stop: where it is, who it is for, the PRO on the paperwork, and whether it is
 * a pickup or a delivery.
 */
export function matchesDeskSearch(entry: DeskEntry, term: string): boolean {
  const needle = term.trim().toLowerCase();

  if (needle.length === 0) {
    return true;
  }

  const { occurrence } = entry;

  return (
    occurrence.locationName.toLowerCase().includes(needle) ||
    occurrence.customerName.toLowerCase().includes(needle) ||
    occurrence.shipmentProNumber.toLowerCase().includes(needle) ||
    occurrence.stopType.toLowerCase().includes(needle)
  );
}

export const DESK_SORTS = [
  { id: "urgency", label: "Urgency", hint: "What can still be saved, first" },
  { id: "exposure", label: "Exposure", hint: "Largest collectable amount first" },
  { id: "dwell", label: "Time on site", hint: "Longest sitting driver first" },
  { id: "deadline", label: "Next deadline", hint: "Soonest clock to run out first" },
] as const;

export type DeskSortId = (typeof DESK_SORTS)[number]["id"];

export const DESK_SORT_IDS: readonly DeskSortId[] = DESK_SORTS.map((sort) => sort.id);

/**
 * The next moment this stop changes state — whichever of the notice deadline or
 * the free-time expiry lands first. Used both for sorting and for the single
 * countdown a card leads with.
 */
export function nextDeskDeadline(entry: DeskEntry): number {
  const { occurrence } = entry;

  if (occurrence.noticeDeadlineAt && occurrence.noticeDeadlineAt < occurrence.freeTimeExpiresAt) {
    return occurrence.noticeDeadlineAt;
  }

  return occurrence.freeTimeExpiresAt;
}

/** Sorts by the requested key, always falling back to urgency so ties stay stable. */
export function sortDeskEntriesBy(entries: DeskEntry[], sort: DeskSortId): DeskEntry[] {
  if (sort === "urgency") {
    return sortDeskEntries(entries);
  }

  const byUrgency = (a: DeskEntry, b: DeskEntry) =>
    DESK_URGENCY_RANK[a.urgency] - DESK_URGENCY_RANK[b.urgency];

  const comparators: Record<
    Exclude<DeskSortId, "urgency">,
    (a: DeskEntry, b: DeskEntry) => number
  > = {
    exposure: (a, b) => b.amountAtRisk - a.amountAtRisk || byUrgency(a, b),
    dwell: (a, b) => a.occurrence.clockStartAt - b.occurrence.clockStartAt || byUrgency(a, b),
    deadline: (a, b) => nextDeskDeadline(a) - nextDeskDeadline(b) || byUrgency(a, b),
  };

  return [...entries].sort(comparators[sort]);
}

/**
 * Geometry for the clock track: one bar spanning arrival to the last deadline
 * that matters, so free time, accrual, and the notice cutoff can be read
 * against each other instead of as three unrelated numbers.
 */
export type DeskClockTrack = {
  /** Where free time ends, 0-100 across the track. */
  freePercent: number;
  /** Where the clock stands now (or where it stopped, for a closed stop). */
  elapsedPercent: number;
  /** Width of the billing portion beyond free time. */
  accruedPercent: number;
  /** Where the notice deadline falls, or null when no notice is required. */
  noticePercent: number | null;
  noticePassed: boolean;
  isAccruing: boolean;
};

/** Headroom past the last marker so a deadline never renders flush against the end. */
const CLOCK_HORIZON_PADDING = 1.08;
const MIN_CLOCK_SPAN_SECONDS = 60;

export function deskClockTrack(entry: DeskEntry, nowSeconds: number): DeskClockTrack {
  const { occurrence } = entry;
  const start = occurrence.clockStartAt;
  // A departed stop freezes at its stop time; an open one tracks the live clock.
  const cursor = Math.max(start, occurrence.clockStopAt ?? nowSeconds);
  const noticeAt = occurrence.noticeDeadlineAt ?? null;
  const horizon = Math.max(occurrence.freeTimeExpiresAt, cursor, noticeAt ?? start);
  const span = Math.max((horizon - start) * CLOCK_HORIZON_PADDING, MIN_CLOCK_SPAN_SECONDS);

  const percentAt = (at: number) => Math.min(100, Math.max(0, ((at - start) / span) * 100));

  const freePercent = percentAt(occurrence.freeTimeExpiresAt);
  const elapsedPercent = percentAt(cursor);

  return {
    freePercent,
    elapsedPercent,
    accruedPercent: Math.max(0, elapsedPercent - freePercent),
    noticePercent: noticeAt === null ? null : percentAt(noticeAt),
    noticePassed: noticeAt !== null && cursor >= noticeAt,
    isAccruing: cursor > occurrence.freeTimeExpiresAt,
  };
}

export type DeskFloorStats = {
  /** Mean minutes on site across every open stop, 0 when the floor is empty. */
  averageOnSiteMinutes: number;
  longestOnSiteMinutes: number;
  longestLocationName: string;
};

/**
 * How long drivers are actually sitting, right now. The averages are read off
 * the live clock rather than the server's snapshot, so they move with the rest
 * of the desk between refreshes.
 */
export function deskFloorStats(entries: DeskEntry[], nowSeconds: number): DeskFloorStats {
  let total = 0;
  let counted = 0;
  let longest = 0;
  let longestLocationName = "";

  for (const { occurrence } of entries) {
    const minutes = Math.max(
      0,
      Math.round(((occurrence.clockStopAt ?? nowSeconds) - occurrence.clockStartAt) / 60),
    );

    total += minutes;
    counted += 1;

    if (minutes > longest) {
      longest = minutes;
      longestLocationName = occurrence.locationName;
    }
  }

  return {
    averageOnSiteMinutes: counted === 0 ? 0 : Math.round(total / counted),
    longestOnSiteMinutes: longest,
    longestLocationName,
  };
}

export type DeskSummary = {
  total: number;
  accruing: number;
  noticesDue: number;
  lost: number;
  amountAtRisk: number;
  amountLost: number;
};

export function summarizeDesk(entries: DeskEntry[]): DeskSummary {
  return entries.reduce<DeskSummary>(
    (acc, entry) => {
      acc.total += 1;

      if (entry.urgency === "Lost") {
        acc.lost += 1;
        acc.amountLost += entry.occurrence.grossAmount;
        return acc;
      }

      acc.amountAtRisk += entry.amountAtRisk;

      if (entry.occurrence.roundedMinutes > 0) {
        acc.accruing += 1;
      }

      if (entry.urgency === "NoticeDueSoon" || entry.urgency === "NoticeOverdue") {
        acc.noticesDue += 1;
      }

      return acc;
    },
    {
      total: 0,
      accruing: 0,
      noticesDue: 0,
      lost: 0,
      amountAtRisk: 0,
      amountLost: 0,
    },
  );
}
