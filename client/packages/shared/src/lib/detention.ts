import {
  DESK_URGENCY_RANK,
  type DeskEntry,
  type DeskUrgency,
  type OccurrenceStatus,
  type ScoreBand,
} from "../types/detention";

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

/**
 * Tailwind classes per urgency. Amber is reserved for the notice window because
 * that is the only state where acting in the next few minutes changes whether
 * the money is collectable at all.
 */
export const URGENCY_STYLES: Record<
  DeskUrgency,
  { badge: string; bar: string; dot: string }
> = {
  NoticeOverdue: {
    badge: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
    bar: "bg-amber-500",
    dot: "bg-amber-500",
  },
  NoticeDueSoon: {
    badge: "bg-amber-400/15 text-amber-700 dark:text-amber-400",
    bar: "bg-amber-400",
    dot: "bg-amber-400",
  },
  Accruing: {
    badge: "bg-red-500/15 text-red-700 dark:text-red-400",
    bar: "bg-red-500",
    dot: "bg-red-500",
  },
  FreeTimeEnding: {
    badge: "bg-orange-400/15 text-orange-700 dark:text-orange-400",
    bar: "bg-orange-400",
    dot: "bg-orange-400",
  },
  Lost: {
    badge: "bg-muted text-muted-foreground",
    bar: "bg-muted-foreground/50",
    dot: "bg-muted-foreground/50",
  },
  Normal: {
    badge: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
    bar: "bg-emerald-500",
    dot: "bg-emerald-500",
  },
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
 * How much of the free-time budget has been consumed, 0-100. Values above 100
 * are clamped; the accrued portion is shown separately.
 */
export function freeTimeConsumedPercent(entry: DeskEntry, nowSeconds: number): number {
  const { occurrence } = entry;
  const granted = occurrence.freeMinutesGranted;

  if (granted <= 0) {
    return 100;
  }

  const elapsedMinutes = Math.max(0, (nowSeconds - occurrence.clockStartAt) / 60);

  return Math.min(100, Math.round((elapsedMinutes / granted) * 100));
}

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
