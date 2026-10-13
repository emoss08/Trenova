import { describe, expect, it } from "vitest";
import type { CheckCallRow } from "@/lib/graphql/check-calls";
import { checkCallPlace, loggableMoves, sortCheckCalls } from "../check-calls";

function call(overrides: Partial<CheckCallRow>): CheckCallRow {
  return {
    id: "chkc_1",
    shipmentId: "shp_1",
    shipmentMoveId: "sm_1",
    stopId: null,
    source: "Dispatcher",
    kind: "Update",
    reportedAt: 1_800_000_000,
    receivedAt: 1_800_000_000,
    latitude: null,
    longitude: null,
    city: null,
    stateCode: null,
    delayed: false,
    delayReason: null,
    estimatedArrival: null,
    note: null,
    contactName: null,
    reportedById: null,
    trackingEventId: null,
    ...overrides,
  } as CheckCallRow;
}

describe("checkCallPlace", () => {
  it("joins the city and state that were given", () => {
    expect(checkCallPlace({ city: "Tulsa", stateCode: "OK" })).toBe("Tulsa, OK");
    expect(checkCallPlace({ city: "Tulsa", stateCode: null })).toBe("Tulsa");
    expect(checkCallPlace({ city: " ", stateCode: "OK" })).toBe("OK");
    expect(checkCallPlace({ city: null, stateCode: null })).toBe("");
  });
});

describe("loggableMoves", () => {
  it("offers every saved move except a canceled one", () => {
    const result = loggableMoves(
      [
        { id: "sm_1", status: "InTransit" },
        { id: undefined, status: "New" },
        { id: "sm_3", status: "Canceled" },
        { id: "sm_4", status: "Completed" },
      ],
      (_move, index) => `Move ${index + 1}`,
    );

    expect(result).toEqual([
      { id: "sm_1", label: "Move 1" },
      { id: "sm_4", label: "Move 4" },
    ]);
  });
});

describe("sortCheckCalls", () => {
  it("puts the latest report first and breaks ties by when it arrived", () => {
    const early = call({ id: "chkc_early", reportedAt: 100, receivedAt: 500 });
    const lateFirst = call({ id: "chkc_late_a", reportedAt: 300, receivedAt: 310 });
    const lateSecond = call({ id: "chkc_late_b", reportedAt: 300, receivedAt: 320 });

    expect(sortCheckCalls([early, lateFirst, lateSecond]).map((row) => row.id)).toEqual([
      "chkc_late_b",
      "chkc_late_a",
      "chkc_early",
    ]);
  });
});
