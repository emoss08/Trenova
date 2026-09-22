import { describe, expect, it } from "vitest";
import { LANE_ORDER, LANE_STATUSES, isLaneKey, type LaneKey } from "../lanes";

describe("inbox lanes", () => {
  it("counts a quarantined message as waiting on somebody", () => {
    // A message nobody could make sense of is not a message that was dealt
    // with. If it only appeared under Everything, the one lane a person
    // actually watches would be quietly incomplete.
    expect(LANE_STATUSES.waiting).toContain("Quarantined");
    expect(LANE_STATUSES.waiting).toContain("InReview");
  });

  it("keeps the desk's own outcomes out of the waiting lane", () => {
    for (const status of LANE_STATUSES.waiting) {
      expect(status).not.toBe("Actioned");
      expect(status).not.toBe("Ignored");
    }
  });

  it("asks for every status in the Everything lane", () => {
    // Empty is the server's "no status filter". A hand-written list here would
    // silently drop any status added later — Received and Processing among
    // them, which is exactly where a stuck message sits.
    expect(LANE_STATUSES.all).toEqual([]);
  });

  it("names a lane for every key it orders, and orders every lane it names", () => {
    expect([...LANE_ORDER].sort()).toEqual(
      (Object.keys(LANE_STATUSES) as LaneKey[]).sort(),
    );
  });

  it("refuses a lane the URL made up", () => {
    expect(isLaneKey("waiting")).toBe(true);
    expect(isLaneKey("everything")).toBe(false);
    expect(isLaneKey(null)).toBe(false);
  });
});
