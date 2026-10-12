import { describe, expect, it } from "vitest";
import {
  groupTrackingEventsByStop,
  trackingOutcomePhase,
  type TrackingEventRow,
} from "../tracking-events";

function report(id: string, stopId: string, eventAt: number): TrackingEventRow {
  return {
    id,
    shipmentMoveId: "sm_1",
    stopId,
    source: "Telematics",
    kind: "Arrival",
    matchMethod: "Location",
    eventAt,
    receivedAt: eventAt + 5,
    sourceStatus: null,
    outcome: "Applied",
    outcomeReason: null,
  };
}

describe("groupTrackingEventsByStop", () => {
  it("orders stops along the route and each stop's reports oldest first", () => {
    const groups = groupTrackingEventsByStop(
      [report("a", "stp_2", 300), report("b", "stp_1", 200), report("c", "stp_2", 100)],
      [
        { id: "stp_2", moveSequence: 0, sequence: 1 },
        { id: "stp_1", moveSequence: 0, sequence: 0 },
      ],
    );

    expect(groups.map((group) => group.stopId)).toEqual(["stp_1", "stp_2"]);
    expect(groups[1]?.events.map((event) => event.id)).toEqual(["c", "a"]);
  });

  it("keeps reports for a stop no longer on the shipment after the route", () => {
    const groups = groupTrackingEventsByStop(
      [report("a", "stp_gone", 100), report("b", "stp_1", 200)],
      [{ id: "stp_1", moveSequence: 1, sequence: 0 }],
    );

    expect(groups.map((group) => group.stopId)).toEqual(["stp_1", "stp_gone"]);
    expect(groups[1]?.stop).toBeNull();
  });

  it("orders moves before stops", () => {
    const groups = groupTrackingEventsByStop(
      [report("a", "stp_b", 100), report("b", "stp_a", 200)],
      [
        { id: "stp_a", moveSequence: 1, sequence: 0 },
        { id: "stp_b", moveSequence: 0, sequence: 3 },
      ],
    );

    expect(groups.map((group) => group.stopId)).toEqual(["stp_b", "stp_a"]);
  });
});

describe("trackingOutcomePhase", () => {
  it("maps each outcome to a lifecycle phase", () => {
    expect(trackingOutcomePhase("Applied")).toBe("complete");
    expect(trackingOutcomePhase("Pending")).toBe("awaiting");
    expect(trackingOutcomePhase("Duplicate")).toBe("closed");
    expect(trackingOutcomePhase("Superseded")).toBe("closed");
    expect(trackingOutcomePhase("Refused")).toBe("failed");
  });
});
