import type { InboundMessageCounts, TriageLane } from "@/lib/graphql/inbox";

/**
 * The lanes Triage splits on, as the address spells them.
 *
 * A lane is a question, not a filter chip: "what is waiting on me" and "what
 * did Nova handle" are read by different people at different times. Which
 * messages answer it is the server's to decide (`TriageLane`), from one
 * definition the list, the counts and the briefing share; the address keeps
 * short words so a link stays readable, and the old inbox's words still open.
 */
export type LaneKey = "waiting" | "them" | "mine" | "snoozed" | "handled" | "ignored" | "all";

export const LANE_ORDER: readonly LaneKey[] = [
  "waiting",
  "them",
  "mine",
  "snoozed",
  "handled",
  "ignored",
  "all",
];

export const LANE_ENUM: Record<LaneKey, TriageLane> = {
  waiting: "Waiting",
  them: "WaitingOnThem",
  mine: "AssignedToMe",
  snoozed: "Snoozed",
  handled: "Handled",
  ignored: "Ignored",
  all: "Everything",
};

/** Lanes that only earn a place in the rail while something is in them. */
const SHOWN_WHEN_NON_EMPTY: ReadonlySet<LaneKey> = new Set(["them", "mine", "snoozed"]);

/** Lanes whose messages all wait on somebody, so a due-first order means something. */
const WAITING_LANES: ReadonlySet<LaneKey> = new Set(["waiting", "them", "mine", "snoozed"]);

export function isLaneKey(value: string | null): value is LaneKey {
  return value !== null && Object.hasOwn(LANE_ENUM, value);
}

export function laneCount(counts: InboundMessageCounts | undefined, lane: LaneKey): number {
  if (counts === undefined) {
    return 0;
  }
  switch (lane) {
    case "waiting":
      return counts.waiting;
    case "them":
      return counts.waitingOnThem;
    case "mine":
      return counts.assignedToMe;
    case "snoozed":
      return counts.snoozed;
    case "handled":
      return counts.handled;
    case "ignored":
      return counts.ignored;
    case "all":
      return counts.total;
  }
}

/** The lanes the rail shows: the conditional ones only while they hold something. */
export function visibleLanes(counts: InboundMessageCounts | undefined): LaneKey[] {
  return LANE_ORDER.filter(
    (lane) => !SHOWN_WHEN_NON_EMPTY.has(lane) || laneCount(counts, lane) > 0,
  );
}

export function isWaitingLane(lane: LaneKey): boolean {
  return WAITING_LANES.has(lane);
}
