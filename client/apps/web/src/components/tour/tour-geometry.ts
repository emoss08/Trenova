import type { TourSide } from "./tour-model";

export type Box = { x: number; y: number; width: number; height: number };
export type Extent = { width: number; height: number };

/** Where a tour card goes: the box it hangs off, the side of it, and the space between. */
export type CardPlacement = { anchor: Box; side: TourSide; gap: number };

/** The space between a lit part and the card beside it. */
const CARD_GAP = 14;
/** The least space kept between the card and the window's edge. */
const WINDOW_EDGE = 12;

const OPPOSITE: Record<TourSide, TourSide> = {
  top: "bottom",
  bottom: "top",
  left: "right",
  right: "left",
};

const ACROSS: Record<TourSide, readonly [TourSide, TourSide]> = {
  top: ["right", "left"],
  bottom: ["right", "left"],
  left: ["bottom", "top"],
  right: ["bottom", "top"],
};

/** The part two boxes share, or null when they do not touch. */
export function overlap(a: Box, b: Box): Box | null {
  const x = Math.max(a.x, b.x);
  const y = Math.max(a.y, b.y);
  const right = Math.min(a.x + a.width, b.x + b.width);
  const bottom = Math.min(a.y + a.height, b.y + b.height);
  if (right <= x || bottom <= y) return null;
  return { x, y, width: right - x, height: bottom - y };
}

/** A box grown by the same amount on every side. */
export function grow(box: Box, by: number): Box {
  return { x: box.x - by, y: box.y - by, width: box.width + by * 2, height: box.height + by * 2 };
}

export function sameBox(a: Box | null, b: Box | null): boolean {
  if (a === null || b === null) return a === b;
  return a.x === b.x && a.y === b.y && a.width === b.width && a.height === b.height;
}

/**
 * Where the card goes for a lit part, so that the whole card is always inside the window.
 * It takes the side the step asked for when the card fits there, then the opposite side, then
 * whichever of the other two has more room. A part that leaves no side with room (a list that
 * fills the window) gets the card against the window's edge, top or bottom, whichever covers
 * less of the part.
 */
export function placeCard({
  lit,
  viewport,
  card,
  preferred,
}: {
  /** The part being shown, as much of it as is on screen. */
  lit: Box;
  viewport: Extent;
  card: Extent;
  preferred: TourSide;
}): CardPlacement {
  const room: Record<TourSide, number> = {
    top: lit.y,
    bottom: viewport.height - (lit.y + lit.height),
    left: lit.x,
    right: viewport.width - (lit.x + lit.width),
  };
  const fits = (side: TourSide) => {
    const needed = side === "top" || side === "bottom" ? card.height : card.width;
    return room[side] >= needed + CARD_GAP + WINDOW_EDGE;
  };
  const [one, other] = ACROSS[preferred];
  const across = room[one] >= room[other] ? [one, other] : [other, one];
  const side = [preferred, OPPOSITE[preferred], ...across].find(fits);
  if (side) return { anchor: lit, side, gap: CARD_GAP };

  const x = Math.min(Math.max(lit.x + lit.width / 2, 0), viewport.width);
  return room.bottom >= room.top
    ? { anchor: { x, y: viewport.height - WINDOW_EDGE, width: 0, height: 0 }, side: "top", gap: 0 }
    : { anchor: { x, y: WINDOW_EDGE, width: 0, height: 0 }, side: "bottom", gap: 0 };
}
