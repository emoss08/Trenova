/**
 * Where a chart's words may stand. Charts are laid out at the exhibit's real pixel width, so the
 * layout knows, near enough, how wide each text is: a mono figure is 0.6 of its size a
 * character, a name in the sans face about 0.57. Every label a chart draws is placed here
 * against everything already standing (marks, other labels, the plot's edges); a label that
 * fits nowhere is not drawn. Fewer words, never smaller ones, never one over another.
 */

export type TextClass = "tk" | "lb" | "nm";
export type Anchor = "start" | "middle" | "end";

/** A rectangle on the chart: left, right, top, bottom. */
export type Box = { x0: number; x1: number; y0: number; y1: number };

/** One place a label may stand: its anchor point and how it hangs from it. */
export type Spot = { x: number; y: number; anchor: Anchor };

/** A label to place: its text, its class, and the spots it may take, best first. */
export type LabelAsk = { text: string; cls: TextClass; spots: Spot[] };

/** The width of one character of each class, and its size, in pixels (desk board-beyond.css). */
const CHAR: Record<TextClass, number> = { tk: 6.3, lb: 6.7, nm: 6.9 };
const SIZE: Record<TextClass, number> = { tk: 10.5, lb: 11, nm: 12 };

/** How far apart two labels stand at the least. */
export const LABEL_GAP = 3;

export function textWidth(text: string, cls: TextClass): number {
  return text.length * CHAR[cls];
}

/** The box a text takes, set at `spot`: from its cap height above the baseline to its descent. */
export function textBox(text: string, cls: TextClass, spot: Spot): Box {
  const width = textWidth(text, cls);
  const x0 =
    spot.anchor === "start" ? spot.x : spot.anchor === "end" ? spot.x - width : spot.x - width / 2;
  return { x0, x1: x0 + width, y0: spot.y - SIZE[cls] * 0.8, y1: spot.y + SIZE[cls] * 0.25 };
}

export function circleBox(cx: number, cy: number, r: number): Box {
  return { x0: cx - r, x1: cx + r, y0: cy - r, y1: cy + r };
}

export function overlaps(a: Box, b: Box, gap = 0): boolean {
  return a.x0 < b.x1 + gap && b.x0 < a.x1 + gap && a.y0 < b.y1 + gap && b.y0 < a.y1 + gap;
}

export function within(box: Box, bounds: Box): boolean {
  return box.x0 >= bounds.x0 && box.x1 <= bounds.x1 && box.y0 >= bounds.y0 && box.y1 <= bounds.y1;
}

/**
 * Each label at its first spot that stays inside `bounds` and clear of every obstacle and of
 * every label placed before it; null where no spot is clear. Labels are placed in the order
 * given, so the order is the priority: the most important first.
 */
export function placeLabels(
  asks: readonly LabelAsk[],
  obstacles: readonly Box[],
  bounds: Box,
): (Spot | null)[] {
  const taken: Box[] = [];
  return asks.map((ask) => {
    for (const spot of ask.spots) {
      const box = textBox(ask.text, ask.cls, spot);
      if (!within(box, bounds)) continue;
      if (obstacles.some((obstacle) => overlaps(box, obstacle))) continue;
      if (taken.some((other) => overlaps(box, other, LABEL_GAP))) continue;
      taken.push(box);
      return spot;
    }
    return null;
  });
}

/**
 * The order to keep a run of `count` ticks in: the first, the last, then the middles of what
 * is left, so that whatever number fits is spread evenly along the axis.
 */
export function bisectionOrder(count: number): number[] {
  if (count <= 0) return [];
  const order = [0];
  if (count > 1) order.push(count - 1);
  let spans: [number, number][] = [[0, count - 1]];
  while (spans.length > 0) {
    const next: [number, number][] = [];
    for (const [from, to] of spans) {
      if (to - from < 2) continue;
      const middle = Math.floor((from + to) / 2);
      order.push(middle);
      next.push([from, middle], [middle, to]);
    }
    spans = next;
  }
  return order;
}

/** A tick along an axis: where it stands, what it says, and whether it is kept before others. */
export type AxisTick = { x: number; text: string; cls: TextClass; first?: boolean };

/**
 * The ticks an axis can carry with none over another: the preferred ticks first (a bound's own
 * day, a marked week), then the rest spread evenly; each centred on its point and moved in
 * from an edge rather than cut by it. Returns where each kept tick's text is set.
 */
export function thinTicks(
  ticks: readonly AxisTick[],
  bounds: { x0: number; x1: number },
  gap = 8,
): { index: number; spot: Spot }[] {
  const order = [
    ...ticks.flatMap((tick, index) => (tick.first ? [index] : [])),
    ...bisectionOrder(ticks.length).filter((index) => !ticks[index].first),
  ];
  const kept: { index: number; box: Box; spot: Spot }[] = [];
  for (const index of order) {
    const tick = ticks[index];
    if (tick.text === "") continue;
    const width = textWidth(tick.text, tick.cls);
    const spot: Spot =
      tick.x - width / 2 < bounds.x0
        ? { x: bounds.x0, y: 0, anchor: "start" }
        : tick.x + width / 2 > bounds.x1
          ? { x: bounds.x1, y: 0, anchor: "end" }
          : { x: tick.x, y: 0, anchor: "middle" };
    const box = textBox(tick.text, tick.cls, spot);
    if (box.x0 < bounds.x0 - 0.5 || box.x1 > bounds.x1 + 0.5) continue;
    if (
      kept.some((other) => overlaps({ ...box, y0: 0, y1: 1 }, { ...other.box, y0: 0, y1: 1 }, gap))
    ) {
      continue;
    }
    kept.push({ index, box, spot });
  }
  return kept.sort((a, b) => a.index - b.index).map(({ index, spot }) => ({ index, spot }));
}

/**
 * A drawn line as small boxes along it, every few pixels, so a label can be kept off the line
 * itself and not merely off its points.
 */
export function lineBoxes(points: readonly { x: number; y: number }[], half = 1.5): Box[] {
  const boxes: Box[] = [];
  for (let index = 0; index < points.length; index += 1) {
    const from = points[index];
    const to = points[index + 1] ?? from;
    const steps = Math.max(1, Math.ceil(Math.hypot(to.x - from.x, to.y - from.y) / 3));
    for (let step = 0; step <= steps; step += 1) {
      const x = from.x + ((to.x - from.x) * step) / steps;
      const y = from.y + ((to.y - from.y) * step) / steps;
      boxes.push({ x0: x - half, x1: x + half, y0: y - half, y1: y + half });
    }
  }
  return boxes;
}

/** The spots around a point a label may take, nearest first: above, below, right, left. */
export function spotsAround(x: number, y: number, clear: number): Spot[] {
  return [
    { x, y: y - clear - 3, anchor: "middle" },
    { x, y: y + clear + 11, anchor: "middle" },
    { x: x + clear + 4, y: y + 4, anchor: "start" },
    { x: x - clear - 4, y: y + 4, anchor: "end" },
  ];
}
