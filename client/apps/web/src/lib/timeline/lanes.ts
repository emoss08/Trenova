/**
 * Lane math every timeline shares: a lane's bars packed into rows, its overlaps, and its free
 * time. A bar is anything with an id and a span in Unix seconds, so a shipment, a move and a
 * driver's commitment all pack the same way.
 */

export type TimelineSpan = { id: string; start: number; end: number };

export type PlacedSpan<T extends TimelineSpan> = { item: T; row: number };

export type PackedLane<T extends TimelineSpan> = {
  placed: PlacedSpan<T>[];
  rows: number;
  conflicts: Set<string>;
};

/** Free time shorter than this between two bars is not worth a label. */
export const MIN_LABELED_GAP_SECONDS = 150 * 60;

/**
 * Lays one lane's bars into as few rows as keep them apart, earliest first, and marks every
 * bar whose time overlaps another bar in the lane: a driver booked twice at once. A carrier
 * runs many trucks and a pool holds work nobody has yet, so their lanes pack without
 * `flagOverlaps`.
 */
export function packLane<T extends TimelineSpan>(
  items: readonly T[],
  { flagOverlaps = true }: { flagOverlaps?: boolean } = {},
): PackedLane<T> {
  const sorted = [...items].sort((a, b) => a.start - b.start || a.end - b.end);
  const rowEnds: number[] = [];
  const placed: PlacedSpan<T>[] = [];
  for (const item of sorted) {
    let row = rowEnds.findIndex((end) => end <= item.start);
    if (row === -1) {
      row = rowEnds.length;
      rowEnds.push(item.end);
    } else {
      rowEnds[row] = item.end;
    }
    placed.push({ item, row });
  }
  const conflicts = new Set<string>();
  for (let i = 0; flagOverlaps && i < sorted.length; i++) {
    for (let j = i + 1; j < sorted.length && sorted[j].start < sorted[i].end; j++) {
      conflicts.add(sorted[i].id);
      conflicts.add(sorted[j].id);
    }
  }
  return { placed, rows: Math.max(1, rowEnds.length), conflicts };
}

/**
 * Where the next bar in the same row starts, by bar id, for every bar that has one: a
 * caption drawn beside a bar may use the room up to it. One pass per row, since a pool lane
 * can hold hundreds of bars.
 */
export function nextStartsInRow<T extends TimelineSpan>(
  packed: PackedLane<T>,
): Map<string, number> {
  const next = new Map<string, number>();
  const byRow = new Map<number, T[]>();
  for (const { item, row } of packed.placed) {
    const items = byRow.get(row);
    if (items) items.push(item);
    else byRow.set(row, [item]);
  }
  for (const items of byRow.values()) {
    items.sort((a, b) => a.start - b.start);
    // Walking back, `after` is the first start later than the bar in hand.
    let after: number | null = null;
    for (let i = items.length - 1; i >= 0; i--) {
      if (i < items.length - 1 && items[i + 1].start > items[i].start) after = items[i + 1].start;
      if (after !== null) next.set(items[i].id, after);
    }
  }
  return next;
}

export type LaneGap = { start: number; end: number; open: boolean };

/**
 * A lane's free time from `from` (no earlier than now) to `to`: each stretch of at least
 * MIN_LABELED_GAP_SECONDS between its bars, and the open time after its last one (`open`).
 * A lane with nothing on it is free the whole way.
 */
export function laneGaps(
  items: readonly Pick<TimelineSpan, "start" | "end">[],
  from: number,
  to: number,
): LaneGap[] {
  const spans = items
    .filter((item) => item.end > from && item.start < to)
    .map((item) => [item.start, item.end] as const)
    .sort((a, b) => a[0] - b[0]);
  const gaps: LaneGap[] = [];
  let cursor = from;
  for (const [start, end] of spans) {
    if (start - cursor >= MIN_LABELED_GAP_SECONDS)
      gaps.push({ start: cursor, end: start, open: false });
    cursor = Math.max(cursor, end);
  }
  if (to - cursor >= MIN_LABELED_GAP_SECONDS) gaps.push({ start: cursor, end: to, open: true });
  return gaps;
}
