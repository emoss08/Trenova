import type { AuditActivityBucket } from "@/lib/graphql/audit-overview";
import { dayBounds, dayKey } from "@/lib/day-grouping";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { AuditActorKind } from "./audit-entry-sentence";

/** The chart's series, bottom to top, each with the token its marks and swatches take. */
export const ACTIVITY_SERIES: readonly {
  key: AuditActorKind;
  fill: string;
  swatch: string;
}[] = [
  { key: "people", fill: "fill-series-1", swatch: "bg-series-1" },
  { key: "agents", fill: "fill-series-11", swatch: "bg-series-11" },
  { key: "integrations", fill: "fill-series-2", swatch: "bg-series-2" },
  { key: "jobs", fill: "fill-border-strong", swatch: "bg-border-strong" },
];

export const ACTOR_KIND_LABELS = defineLabels({
  people: "People",
  agents: "Agents",
  integrations: "Integrations",
  jobs: "Jobs",
} as const);

/** Off hours, in the organization's time zone: from 8 PM to 6 AM, as the server counts them. */
export const OFF_HOURS_FROM = 20;
export const OFF_HOURS_TO = 6;

const HOUR_SECONDS = 3600;
const DAY_SECONDS = 86_400;

export function bucketTotal(bucket: AuditActivityBucket): number {
  return bucket.people + bucket.agents + bucket.integrations + bucket.jobs;
}

/**
 * Entries per local day of `timezone`, keyed as the trail keys its day groups, for the days whose
 * edges fall on bucket edges: a day the buckets cut through, at the window's start or across
 * time zones whose hours or days do not line up, is left out rather than miscounted. The last
 * day may run past the last bucket, since it is still going.
 */
export function entriesByDay(
  buckets: readonly AuditActivityBucket[],
  timezone: string,
): Map<string, number> {
  const out = new Map<string, number>();
  if (buckets.length === 0) return out;
  const starts = new Set(buckets.map((bucket) => bucket.at));
  const lastAt = buckets[buckets.length - 1].at;
  const sums = new Map<string, number>();
  for (const bucket of buckets) {
    const key = dayKey(bucket.at, timezone);
    sums.set(key, (sums.get(key) ?? 0) + bucketTotal(bucket));
  }
  for (const [key, total] of sums) {
    const bounds = dayBounds(key, timezone);
    if (!bounds || !starts.has(bounds.start)) continue;
    if (!starts.has(bounds.end) && bounds.end <= lastAt) continue;
    out.set(key, total);
  }
  return out;
}

/** A step for the value axis that puts three to five gridlines under the tallest stack. */
export function niceStep(max: number): number {
  if (max <= 0) return 1;
  const rough = max / 4;
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  const normalized = rough / magnitude;
  const nice = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10;
  return nice * magnitude;
}

/**
 * The instants buckets `a` through `b` cover, `[from, to]`: a bucket ends where the next one
 * starts, so a day bucket keeps its 23 or 25 hours across a clock change.
 */
export function bucketRange(
  buckets: readonly AuditActivityBucket[],
  a: number,
  b: number,
  unit: "Hour" | "Day",
): { from: number; to: number } | null {
  const first = Math.max(0, Math.min(a, b));
  const last = Math.min(buckets.length - 1, Math.max(a, b));
  if (!buckets[first] || !buckets[last]) return null;
  const next = buckets[last + 1];
  const end = next ? next.at : buckets[last].at + (unit === "Hour" ? HOUR_SECONDS : DAY_SECONDS);
  return { from: buckets[first].at, to: end - 1 };
}

/** The index of the bucket an instant falls in, or -1 outside them. */
export function bucketIndexOf(buckets: readonly AuditActivityBucket[], at: number): number {
  for (let i = buckets.length - 1; i >= 0; i--) {
    if (at >= buckets[i].at) {
      return i;
    }
  }
  return -1;
}

const hourFormatters = new Map<string, Intl.DateTimeFormat>();

/** The hour of the day an instant falls on in the time zone, 0 to 23. */
export function localHour(at: number, timezone: string): number {
  let format = hourFormatters.get(timezone);
  if (!format) {
    format = new Intl.DateTimeFormat("en-US", {
      hour: "numeric",
      hourCycle: "h23",
      timeZone: timezone,
    });
    hourFormatters.set(timezone, format);
  }
  return Number(format.format(new Date(at * 1000))) % 24;
}

export function isOffHour(at: number, timezone: string): boolean {
  const hour = localHour(at, timezone);
  return hour >= OFF_HOURS_FROM || hour < OFF_HOURS_TO;
}

export function shareOf(part: number, whole: number): number {
  return whole > 0 ? part / whole : 0;
}

/** The change against the window before, as a whole percent, or null with nothing to compare. */
export function percentChange(now: number, before: number): number | null {
  return before > 0 ? Math.round(((now - before) / before) * 100) : null;
}
