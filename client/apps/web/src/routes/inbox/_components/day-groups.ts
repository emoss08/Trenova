/**
 * Mail grouped the way a person remembers it arriving: today, yesterday, a
 * weekday this week, and a date before that.
 *
 * A message that came in at 23:40 was yesterday's to the person reading it,
 * whatever the server's zone says. The list arrives newest first and the
 * groups keep that order, so grouping never reorders.
 */
export type DayBucket = "today" | "yesterday" | "week" | "earlier";

export type DayGroup<T> = {
  bucket: DayBucket;
  /** Local midnight of the day, in Unix seconds. */
  day: number;
  items: T[];
};

const WEEK_DAYS = 7;
const DAY_SECONDS = 86_400;

/** The start of the day a moment falls in, in Unix seconds. */
export type StartOfDay = (seconds: number) => number;

function browserStartOfDay(seconds: number): number {
  const date = new Date(seconds * 1000);
  date.setHours(0, 0, 0, 0);

  return Math.floor(date.getTime() / 1000);
}

/*
 * Whole days between two day starts. A day across a clock change is 23 or 25
 * hours long, so the gap is rounded rather than divided exactly.
 */
function daysBetween(fromDay: number, toDay: number): number {
  return Math.round((toDay - fromDay) / DAY_SECONDS);
}

function bucketFor(ago: number): DayBucket {
  if (ago <= 0) {
    return "today";
  }
  if (ago === 1) {
    return "yesterday";
  }
  if (ago < WEEK_DAYS) {
    return "week";
  }

  return "earlier";
}

/**
 * Days are the reader's days: pass the start-of-day for the zone the rest of
 * the page formats times in, so a heading never disagrees with the times
 * under it. The browser's own zone is the fallback.
 */
export function groupByDay<T extends { receivedAt: number }>(
  items: readonly T[],
  now: number,
  startOfDay: StartOfDay = browserStartOfDay,
): DayGroup<T>[] {
  const today = startOfDay(now);
  const groups: DayGroup<T>[] = [];

  for (const item of items) {
    const day = Math.min(startOfDay(item.receivedAt), today);
    const last = groups.at(-1);
    if (last !== undefined && last.day === day) {
      last.items.push(item);
      continue;
    }

    groups.push({ bucket: bucketFor(daysBetween(day, today)), day, items: [item] });
  }

  return groups;
}
