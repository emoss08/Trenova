import { arrayMove } from "@dnd-kit/sortable";

export type Lane = "packet" | "out";
export type Lanes = Record<Lane, string[]>;

const LANE_PREFIX = "lane:";

/** The drop target id that stands for a whole lane, not a card in it. */
export const laneDropId = (lane: Lane) => `${LANE_PREFIX}${lane}`;

export const isLaneDropId = (id: string) => id.startsWith(LANE_PREFIX);

/** The lane a drop target belongs to: a lane itself, or the lane holding a card. */
export function laneOf(lanes: Lanes, id: string): Lane | null {
  if (isLaneDropId(id)) return id.slice(LANE_PREFIX.length) as Lane;
  if (lanes.packet.includes(id)) return "packet";
  if (lanes.out.includes(id)) return "out";
  return null;
}

/**
 * The lanes with a dragged card moved across into the lane of `overId`: beside the
 * card it is over, after it when `past` its middle, or last when it is over the lane
 * itself. A card already in that lane is left to the sortable to move.
 */
export function lanesAfterCross(lanes: Lanes, id: string, overId: string, past: boolean): Lanes {
  const from = laneOf(lanes, id);
  const to = laneOf(lanes, overId);
  if (!from || !to || from === to) return lanes;
  const into = lanes[to].filter((entry) => entry !== id);
  const overIndex = into.indexOf(overId);
  const at = overIndex >= 0 ? overIndex + (past ? 1 : 0) : into.length;
  into.splice(at, 0, id);
  return { ...lanes, [from]: lanes[from].filter((entry) => entry !== id), [to]: into };
}

/** The lanes once a card is dropped over `overId` in its own lane: moved to that card's place. */
export function lanesAfterDrop(lanes: Lanes, id: string, overId: string | null): Lanes {
  const lane = laneOf(lanes, id);
  if (!overId || !lane || laneOf(lanes, overId) !== lane) return lanes;
  const from = lanes[lane].indexOf(id);
  const to = lanes[lane].indexOf(overId);
  if (from < 0 || to < 0 || from === to) return lanes;
  return { ...lanes, [lane]: arrayMove(lanes[lane], from, to) };
}

export function sameOrder(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((entry, index) => entry === b[index]);
}

/**
 * The lanes with the rest of a dragged group taken out, so they fold into the card
 * in hand for the length of the drag. The card in hand stays where it was.
 */
export function lanesWithoutGroup(lanes: Lanes, id: string, group: readonly string[]): Lanes {
  const folded = new Set(group.filter((entry) => entry !== id));
  if (folded.size === 0) return lanes;
  return {
    packet: lanes.packet.filter((entry) => !folded.has(entry)),
    out: lanes.out.filter((entry) => !folded.has(entry)),
  };
}

/**
 * The lanes once a dragged group lands: the rest of the group, in its own order,
 * right after the card in hand, in whichever lane that card was dropped.
 */
export function lanesWithGroup(lanes: Lanes, id: string, group: readonly string[]): Lanes {
  const lane = laneOf(lanes, id);
  const rest = group.filter((entry) => entry !== id);
  if (!lane || rest.length === 0) return lanes;
  const placed = new Set(rest);
  const next = {
    packet: lanes.packet.filter((entry) => !placed.has(entry)),
    out: lanes.out.filter((entry) => !placed.has(entry)),
  };
  next[lane].splice(next[lane].indexOf(id) + 1, 0, ...rest);
  return next;
}
