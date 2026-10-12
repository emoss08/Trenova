import type { InboundMessageCounts } from "@/lib/graphql/inbox";
import { describe, expect, it } from "vitest";
import { LANE_ENUM, LANE_ORDER, isLaneKey, laneCount, visibleLanes } from "../lanes";

const empty: InboundMessageCounts = {
  waiting: 4,
  waitingOnThem: 0,
  snoozed: 0,
  assignedToMe: 0,
  overdue: 0,
  processing: 0,
  handled: 3,
  ignored: 0,
  quarantined: 0,
  total: 7,
  byClassification: [],
  byMailbox: [],
};

describe("triage lanes", () => {
  it("maps every lane the address knows to the server's lane, in the rail's order", () => {
    expect(LANE_ORDER.map((lane) => LANE_ENUM[lane])).toEqual([
      "Waiting",
      "WaitingOnThem",
      "AssignedToMe",
      "Snoozed",
      "Handled",
      "Ignored",
      "Everything",
    ]);
  });

  it("shows Waiting on them, Assigned to me and Snoozed only while they hold something", () => {
    expect(visibleLanes(empty)).toEqual(["waiting", "handled", "ignored", "all"]);
    expect(visibleLanes({ ...empty, snoozed: 2, assignedToMe: 1 })).toEqual([
      "waiting",
      "mine",
      "snoozed",
      "handled",
      "ignored",
      "all",
    ]);
  });

  it("counts each lane from the one aggregate", () => {
    expect(laneCount({ ...empty, waitingOnThem: 5 }, "them")).toBe(5);
    expect(laneCount(empty, "all")).toBe(7);
    expect(laneCount(undefined, "waiting")).toBe(0);
  });

  it("refuses a lane the address made up", () => {
    expect(isLaneKey("snoozed")).toBe(true);
    expect(isLaneKey("everything")).toBe(false);
    expect(isLaneKey("toString")).toBe(false);
    expect(isLaneKey(null)).toBe(false);
  });
});
