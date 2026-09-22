import type { InboundMessageStatus } from "@/lib/graphql/inbox";

/**
 * The lanes the inbox splits on.
 *
 * A lane is a question, not a filter chip: "what is waiting on me" and "what
 * did the desk handle" are read by different people at different times. So
 * each lane names its statuses rather than the page assembling them, and the
 * waiting lane deliberately holds both InReview and Quarantined — a message
 * nobody could make sense of is still a message waiting on somebody.
 */
export type LaneKey = "waiting" | "handled" | "ignored" | "all";

export const LANE_STATUSES: Record<LaneKey, InboundMessageStatus[]> = {
  waiting: ["InReview", "Quarantined"],
  handled: ["Actioned"],
  ignored: ["Ignored"],
  all: [],
};

export const LANE_ORDER: LaneKey[] = ["waiting", "handled", "ignored", "all"];

export function isLaneKey(value: string | null): value is LaneKey {
  return value !== null && value in LANE_STATUSES;
}
