import { describe, expect, it } from "vitest";
import type { CheckCallRow } from "@/lib/graphql/check-calls";
import {
  checkCallPlace,
  defaultCheckCallMoveId,
  groupCheckCallsByDay,
  loggableMoves,
  relativeDay,
  sortCheckCalls,
  stopTimesFreeze,
} from "../check-calls";

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

const unix = (iso: string) => Date.parse(iso) / 1000;

describe("defaultCheckCallMoveId", () => {
  const loggable = [
    { id: "sm_1", label: "Move 1" },
    { id: "sm_2", label: "Move 2" },
  ];

  it("picks the move the truck is driving", () => {
    expect(
      defaultCheckCallMoveId(loggable, [
        { id: "sm_1", status: "Completed" },
        { id: "sm_2", status: "InTransit" },
      ]),
    ).toBe("sm_2");
  });

  it("falls back to the first move that takes calls, then to nothing", () => {
    expect(
      defaultCheckCallMoveId(loggable, [
        { id: "sm_1", status: "Assigned" },
        { id: "sm_2", status: "New" },
      ]),
    ).toBe("sm_1");
    expect(defaultCheckCallMoveId([], [{ id: "sm_1", status: "Canceled" }])).toBe("");
  });
});

describe("groupCheckCallsByDay", () => {
  it("files each call under the day it was reported in the person's timezone", () => {
    const morning = call({ id: "chkc_morning", reportedAt: unix("2026-10-27T14:00:00Z") });
    // 03:30 UTC on the 27th is still 22:30 on the 26th in Chicago.
    const lateEvening = call({ id: "chkc_evening", reportedAt: unix("2026-10-27T03:30:00Z") });
    const afternoon = call({ id: "chkc_afternoon", reportedAt: unix("2026-10-26T20:00:00Z") });
    const older = call({ id: "chkc_older", reportedAt: unix("2026-10-24T16:00:00Z") });

    const days = groupCheckCallsByDay([morning, lateEvening, afternoon, older], "America/Chicago");

    expect(days.map(({ day, calls }) => [day, calls.map((row) => row.id)])).toEqual([
      [unix("2026-10-27T05:00:00Z"), ["chkc_morning"]],
      [unix("2026-10-26T05:00:00Z"), ["chkc_evening", "chkc_afternoon"]],
      [unix("2026-10-24T05:00:00Z"), ["chkc_older"]],
    ]);
  });

  it("returns no days for no calls", () => {
    expect(groupCheckCallsByDay([], "America/Chicago")).toEqual([]);
  });
});

describe("relativeDay", () => {
  const now = new Date("2026-10-27T15:00:00Z");

  it("names today and yesterday and nothing earlier", () => {
    expect(relativeDay(unix("2026-10-27T05:00:00Z"), now, "America/Chicago")).toBe("today");
    expect(relativeDay(unix("2026-10-26T05:00:00Z"), now, "America/Chicago")).toBe("yesterday");
    expect(relativeDay(unix("2026-10-25T05:00:00Z"), now, "America/Chicago")).toBe("earlier");
  });

  it("finds yesterday across the night the clocks go back", () => {
    // Sunday 1 November 2026 is 25 hours long in Chicago.
    const monday = new Date("2026-11-02T18:00:00Z");

    expect(relativeDay(unix("2026-11-01T05:00:00Z"), monday, "America/Chicago")).toBe("yesterday");
    expect(relativeDay(unix("2026-11-02T06:00:00Z"), monday, "America/Chicago")).toBe("today");
  });
});

describe("stopTimesFreeze", () => {
  it("freezes an invoiced load's stop times whatever billing says", () => {
    expect(stopTimesFreeze({ status: "Invoiced", billingTransferStatus: "Posted" })).toBe(
      "invoiced",
    );
    expect(stopTimesFreeze({ status: "Invoiced", billingTransferStatus: null })).toBe("invoiced");
  });

  it("freezes them for every billing status that still holds the load", () => {
    const held = [
      "ReadyForReview",
      "InReview",
      "OnHold",
      "Exception",
      "Approved",
      "Posted",
      "Canceled",
    ] as const;
    for (const billingTransferStatus of held) {
      expect(stopTimesFreeze({ status: "ReadyToInvoice", billingTransferStatus })).toBe(
        "billing-queue",
      );
    }
  });

  it("lets them change before billing has the load or after it hands it back", () => {
    expect(stopTimesFreeze({ status: "InTransit", billingTransferStatus: null })).toBeNull();
    expect(stopTimesFreeze({ status: "InTransit", billingTransferStatus: undefined })).toBeNull();
    expect(
      stopTimesFreeze({ status: "ReadyToInvoice", billingTransferStatus: "SentBackToOps" }),
    ).toBeNull();
  });
});
