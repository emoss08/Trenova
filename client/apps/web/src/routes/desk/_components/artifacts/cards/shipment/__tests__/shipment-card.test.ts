import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatPercent } from "@trenova/shared/lib/utils";
import type {
  Shipment,
  ShipmentBillingReadiness,
  ShipmentBillingRequirement,
  ShipmentCommodity,
  ShipmentMove,
  Stop,
} from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { cardDay, cardMoment, money } from "../../../record-card/record-card-values";
import {
  actualsLabel,
  assignmentStatusLabel,
  billingPhase,
  carrierStatusLabel,
  coverageSummary,
  freightClassLabel,
  marginGauge,
  stopOffsets,
  tenderStatusLabel,
  truckOffset,
  chargesOf,
  coverageOf,
  durationLabel,
  etaReading,
  freightOf,
  initialsOf,
  journeyProgress,
  journeyStops,
  laneOf,
  marginOf,
  nowReading,
  punctualityLabel,
  readinessChecks,
  shipmentStage,
  STAGE_TONE,
  stageLabel,
  stopTypeLabel,
  temperatureLabel,
  totalMiles,
  windowLabel,
  type JourneyStop,
} from "../shipment-card";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

/**
 * A wall-clock moment in October 2026 in the zone the tests run in, as unix
 * seconds. Building times from the local clock keeps "the same day" and "the
 * day after" true wherever the suite runs.
 */
function at(day: number, hour: number, minute = 0): number {
  return new Date(2026, 9, day, hour, minute).getTime() / 1000;
}

function clock(seconds: number): string {
  return new Date(seconds * 1000).toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
}

function range(start: number, end: number): string {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).formatRange(new Date(start * 1000), new Date(end * 1000));
}

function location(name: string, city: string, state?: string): Stop["location"] {
  return {
    name,
    city,
    state: state ? { abbreviation: state, name: state } : null,
  } as Stop["location"];
}

function stop(fields: Partial<Stop> & { id: string }): Stop {
  return {
    locationId: `loc-${fields.id}`,
    status: "New",
    type: "Pickup",
    scheduleType: "Open",
    sequence: 0,
    scheduledWindowStart: 0,
    scheduledWindowEnd: null,
    actualArrival: null,
    actualDeparture: null,
    pieces: null,
    weight: null,
    ...fields,
  } as Stop;
}

function move(fields: Partial<ShipmentMove> & { id: string }): ShipmentMove {
  return {
    status: "New",
    loaded: true,
    sequence: 0,
    distance: null,
    stops: [],
    assignment: null,
    carrierAssignment: null,
    ...fields,
  } as ShipmentMove;
}

function shipment(fields: Partial<Shipment> = {}): Shipment {
  return {
    id: "shp_1",
    status: "New",
    moves: [],
    additionalCharges: [],
    commodities: [],
    ...fields,
  } as Shipment;
}

function journeyStop(fields: Partial<JourneyStop> = {}): JourneyStop {
  return {
    id: "stp",
    moveId: "m1",
    moveIndex: 0,
    index: 0,
    type: "Pickup",
    place: { name: "", city: "", state: "", label: "" },
    appointment: false,
    windowStart: null,
    windowEnd: null,
    arrivedAt: null,
    departedAt: null,
    phase: "next",
    punctuality: null,
    dwellMinutes: null,
    pieces: null,
    weight: null,
    ...fields,
  };
}

/**
 * Two moves handed over out of order, their stops out of order too: Dallas
 * done early, Shreveport done late, the truck at the Shreveport split pickup,
 * one delivery called off, Atlanta still to come.
 */
function journeyMoves(m2Distance: number | null = 300): ShipmentMove[] {
  return [
    move({
      id: "m2",
      sequence: 1,
      status: "InTransit",
      distance: m2Distance,
      stops: [
        stop({
          id: "s5",
          type: "Delivery",
          sequence: 2,
          scheduledWindowStart: at(13, 8),
          scheduledWindowEnd: at(13, 10),
          location: location("Peach DC", "Atlanta", "GA"),
          pieces: 12,
          weight: 18000,
        }),
        stop({
          id: "s3",
          type: "SplitPickup",
          status: "InTransit",
          sequence: 0,
          scheduledWindowStart: at(12, 16),
          scheduledWindowEnd: at(12, 17),
          actualArrival: at(12, 16),
          location: location("Red River Yard", "Shreveport", "LA"),
        }),
        stop({
          id: "s4",
          type: "Delivery",
          status: "Canceled",
          sequence: 1,
          scheduledWindowStart: at(12, 18),
          scheduledWindowEnd: at(12, 19),
          location: location("Jackson Hub", "Jackson", "MS"),
        }),
      ],
    }),
    move({
      id: "m1",
      sequence: 0,
      status: "Completed",
      distance: 100,
      stops: [
        stop({
          id: "s2",
          type: "Delivery",
          status: "Completed",
          sequence: 1,
          scheduledWindowStart: at(12, 12),
          scheduledWindowEnd: at(12, 14),
          actualArrival: at(12, 14, 5),
          actualDeparture: at(12, 15),
          location: location("Red River Yard", "Shreveport", "LA"),
        }),
        stop({
          id: "s1",
          type: "Pickup",
          status: "Completed",
          scheduleType: "Appointment",
          sequence: 0,
          scheduledWindowStart: at(12, 8),
          scheduledWindowEnd: at(12, 9),
          actualArrival: at(12, 7, 40),
          actualDeparture: at(12, 8, 28),
          location: location("Dallas DC", "Dallas", "TX"),
          pieces: 24,
          weight: 40000,
        }),
      ],
    }),
  ];
}

const JOURNEY_NOW = at(12, 18, 30);

describe("journeyStops", () => {
  it("orders moves and their stops by sequence and flattens them", () => {
    const stops = journeyStops(journeyMoves(), JOURNEY_NOW);
    expect(stops.map((entry) => entry.id)).toEqual(["s1", "s2", "s3", "s4", "s5"]);
    expect(stops.map((entry) => entry.moveId)).toEqual(["m1", "m1", "m2", "m2", "m2"]);
    expect(stops.map((entry) => entry.moveIndex)).toEqual([0, 0, 1, 1, 1]);
    expect(stops.map((entry) => entry.index)).toEqual([0, 1, 2, 3, 4]);
    expect(stops.map((entry) => entry.type)).toEqual([
      "Pickup",
      "Delivery",
      "SplitPickup",
      "Delivery",
      "Delivery",
    ]);
  });

  it("marks done, current, canceled and later stops", () => {
    const stops = journeyStops(journeyMoves(), JOURNEY_NOW);
    expect(stops.map((entry) => entry.phase)).toEqual([
      "done",
      "done",
      "current",
      "canceled",
      "later",
    ]);
  });

  it("reads each arrival against its window and how long the truck stayed", () => {
    const stops = journeyStops(journeyMoves(), JOURNEY_NOW);
    expect(stops.map((entry) => entry.punctuality)).toEqual([
      { state: "early", minutes: 20 },
      { state: "late", minutes: 5 },
      { state: "onTime", minutes: 0 },
      null,
      null,
    ]);
    expect(stops.map((entry) => entry.dwellMinutes)).toEqual([48, 55, null, null, null]);
  });

  it("carries the window, the actuals, the place and the freight", () => {
    const [first, , current, , last] = journeyStops(journeyMoves(), JOURNEY_NOW);
    expect(first).toMatchObject({
      appointment: true,
      windowStart: at(12, 8),
      windowEnd: at(12, 9),
      arrivedAt: at(12, 7, 40),
      departedAt: at(12, 8, 28),
      place: { name: "Dallas DC", city: "Dallas", state: "TX", label: "Dallas, TX" },
      pieces: 24,
      weight: 40000,
    });
    expect(current).toMatchObject({ appointment: false, arrivedAt: at(12, 16), departedAt: null });
    expect(last).toMatchObject({
      arrivedAt: null,
      departedAt: null,
      pieces: 12,
      weight: 18000,
      place: { label: "Atlanta, GA" },
    });
  });

  it("makes the first open stop next until the truck is at it, and holds a missed window", () => {
    const moves = [
      move({
        id: "m1",
        stops: [
          stop({
            id: "p",
            status: "Completed",
            scheduledWindowStart: at(12, 8),
            scheduledWindowEnd: at(12, 9),
            actualArrival: at(12, 8, 30),
            actualDeparture: at(12, 9),
          }),
          stop({
            id: "d1",
            type: "Delivery",
            sequence: 1,
            scheduledWindowStart: at(12, 12),
            scheduledWindowEnd: at(12, 14),
          }),
          stop({
            id: "d2",
            type: "Delivery",
            sequence: 2,
            scheduledWindowStart: at(12, 15),
          }),
        ],
      }),
    ];

    const before = journeyStops(moves, at(12, 13));
    expect(before.map((entry) => entry.phase)).toEqual(["done", "next", "later"]);
    expect(before.map((entry) => entry.punctuality)).toEqual([
      { state: "onTime", minutes: 0 },
      null,
      null,
    ]);

    const missed = journeyStops(moves, at(12, 15));
    expect(missed[1].punctuality).toEqual({ state: "late", minutes: 60 });
    expect(missed[2].punctuality).toBeNull();

    const both = journeyStops(moves, at(12, 16, 30));
    expect(both[1].punctuality).toEqual({ state: "late", minutes: 150 });
    expect(both[2].punctuality).toEqual({ state: "late", minutes: 90 });
  });

  it("reads an arrival against the start when the window has no end", () => {
    const moves = [
      move({
        id: "m1",
        stops: [
          stop({
            id: "late",
            status: "Completed",
            scheduledWindowStart: at(12, 10),
            actualArrival: at(12, 10, 30),
          }),
          stop({
            id: "early",
            status: "Completed",
            sequence: 1,
            scheduledWindowStart: at(12, 10),
            actualArrival: at(12, 9, 50),
          }),
          stop({
            id: "inside",
            status: "Completed",
            sequence: 2,
            scheduledWindowStart: at(12, 10),
            scheduledWindowEnd: at(12, 12),
            actualArrival: at(12, 11),
          }),
          stop({
            id: "unscheduled",
            status: "Completed",
            sequence: 3,
            actualArrival: at(12, 11),
          }),
        ],
      }),
    ];
    expect(journeyStops(moves, at(12, 20)).map((entry) => entry.punctuality)).toEqual([
      { state: "late", minutes: 30 },
      { state: "early", minutes: 10 },
      { state: "onTime", minutes: 0 },
      null,
    ]);
  });

  it("calls a stop on a canceled move canceled, and skips it when choosing the next one", () => {
    const moves = [
      move({
        id: "m1",
        status: "Canceled",
        stops: [stop({ id: "gone", scheduledWindowStart: at(12, 8) })],
      }),
      move({ id: "m2", sequence: 1, stops: [stop({ id: "ahead" })] }),
    ];
    const stops = journeyStops(moves, at(12, 20));
    expect(stops.map((entry) => [entry.id, entry.phase, entry.punctuality])).toEqual([
      ["gone", "canceled", null],
      ["ahead", "next", null],
    ]);
  });

  it("names a place by city and state, the city, the name or the address", () => {
    const moves = [
      move({
        id: "m1",
        stops: [
          stop({ id: "a", location: location("Acme Plant", "") }),
          stop({ id: "b", sequence: 1, location: location("Bluff Yard", "Memphis") }),
          stop({ id: "c", sequence: 2, location: null, addressLine: " 12 Main St " }),
          stop({ id: "d", sequence: 3, location: undefined }),
        ],
      }),
    ];
    expect(journeyStops(moves, at(12, 8)).map((entry) => entry.place)).toEqual([
      { name: "Acme Plant", city: "", state: "", label: "Acme Plant" },
      { name: "Bluff Yard", city: "Memphis", state: "", label: "Memphis" },
      { name: "", city: "", state: "", label: "12 Main St" },
      { name: "", city: "", state: "", label: "" },
    ]);
  });

  it("gives nothing for no moves and survives a move with no stops", () => {
    expect(journeyStops([], at(12, 8))).toEqual([]);
    expect(
      journeyStops([move({ id: "m1", stops: undefined as unknown as Stop[] })], at(12, 8)),
    ).toEqual([]);
  });
});

describe("journeyProgress", () => {
  it("weighs moves by their distance when every running move has one", () => {
    const moves = journeyMoves(300);
    expect(journeyProgress(moves, journeyStops(moves, JOURNEY_NOW))).toBeCloseTo(175 / 400);
  });

  it("leaves a canceled move without a distance out of the weighing", () => {
    const moves = [
      ...journeyMoves(300),
      move({
        id: "m3",
        sequence: 2,
        status: "Canceled",
        distance: null,
        stops: [stop({ id: "s6", type: "Delivery" })],
      }),
    ];
    expect(journeyProgress(moves, journeyStops(moves, JOURNEY_NOW))).toBeCloseTo(175 / 400);
  });

  it("counts stops when a move has no distance, a current stop as half", () => {
    for (const distance of [null, 0]) {
      const moves = journeyMoves(distance);
      expect(journeyProgress(moves, journeyStops(moves, JOURNEY_NOW))).toBeCloseTo(2.5 / 4);
    }
  });

  it("is whole when every stop is done and nothing with no stops", () => {
    const done = [
      move({
        id: "m1",
        distance: 50,
        stops: [
          stop({ id: "a", status: "Completed" }),
          stop({ id: "b", sequence: 1, status: "Completed" }),
          stop({ id: "c", sequence: 2, status: "Canceled" }),
        ],
      }),
    ];
    expect(journeyProgress(done, journeyStops(done, at(12, 8)))).toBe(1);
    expect(journeyProgress([], [])).toBe(0);

    const allCanceled = [move({ id: "m1", stops: [stop({ id: "a", status: "Canceled" })] })];
    expect(journeyProgress(allCanceled, journeyStops(allCanceled, at(12, 8)))).toBe(0);
  });

  it("is nothing before the first stop is reached", () => {
    const moves = [move({ id: "m1", stops: [stop({ id: "a" }), stop({ id: "b", sequence: 1 })] })];
    expect(journeyProgress(moves, journeyStops(moves, at(12, 8)))).toBe(0);
  });
});

describe("laneOf", () => {
  it("runs from the first pickup to the last delivery, leaving canceled stops aside", () => {
    const stops = journeyStops(
      [
        move({
          id: "m1",
          stops: [
            stop({ id: "d0", type: "Delivery", location: location("Pre", "Waco", "TX") }),
            stop({ id: "p", sequence: 1, location: location("Origin", "Dallas", "TX") }),
            stop({
              id: "d",
              type: "Delivery",
              sequence: 2,
              location: location("Dest", "Atlanta", "GA"),
            }),
            stop({
              id: "x",
              type: "SplitDelivery",
              status: "Canceled",
              sequence: 3,
              location: location("Gone", "Miami", "FL"),
            }),
          ],
        }),
      ],
      at(12, 8),
    );
    const lane = laneOf(stops);
    expect(lane.from?.label).toBe("Dallas, TX");
    expect(lane.to?.label).toBe("Atlanta, GA");
  });

  it("falls back to the first and last stop", () => {
    const deliveries = journeyStops(
      [
        move({
          id: "m1",
          stops: [
            stop({ id: "a", type: "Delivery", location: location("A", "Austin", "TX") }),
            stop({
              id: "b",
              type: "Delivery",
              sequence: 1,
              location: location("B", "Boise", "ID"),
            }),
          ],
        }),
      ],
      at(12, 8),
    );
    expect(laneOf(deliveries)).toMatchObject({
      from: { label: "Austin, TX" },
      to: { label: "Boise, ID" },
    });

    const pickups = journeyStops(
      [
        move({
          id: "m1",
          stops: [
            stop({ id: "a", type: "SplitPickup", location: location("A", "Austin", "TX") }),
            stop({ id: "b", sequence: 1, location: location("B", "Boise", "ID") }),
          ],
        }),
      ],
      at(12, 8),
    );
    expect(laneOf(pickups)).toMatchObject({
      from: { label: "Austin, TX" },
      to: { label: "Boise, ID" },
    });
  });

  it("uses every stop when all of them were canceled, and nothing with none", () => {
    const canceled = journeyStops(
      [
        move({
          id: "m1",
          status: "Canceled",
          stops: [
            stop({ id: "a", location: location("A", "Austin", "TX") }),
            stop({
              id: "b",
              type: "Delivery",
              sequence: 1,
              location: location("B", "Boise", "ID"),
            }),
          ],
        }),
      ],
      at(12, 8),
    );
    expect(laneOf(canceled)).toMatchObject({
      from: { label: "Austin, TX" },
      to: { label: "Boise, ID" },
    });
    expect(laneOf([])).toEqual({ from: null, to: null });
  });
});

describe("shipmentStage", () => {
  it("takes the server's stage when it sent one", () => {
    expect(shipmentStage({ status: "InTransit", stage: "Late" })).toBe("Late");
  });

  it("derives one from every status the way the server does", () => {
    expect(shipmentStage({ status: "New" })).toBe("NeedsCoverage");
    expect(shipmentStage({ status: "PartiallyAssigned" })).toBe("NeedsCoverage");
    expect(shipmentStage({ status: "Assigned" })).toBe("Scheduled");
    expect(shipmentStage({ status: "InTransit" })).toBe("Moving");
    expect(shipmentStage({ status: "PartiallyCompleted" })).toBe("Moving");
    expect(shipmentStage({ status: "Delayed" })).toBe("Late");
    expect(shipmentStage({ status: "Completed" })).toBe("Delivered");
    expect(shipmentStage({ status: "ReadyToInvoice" })).toBe("Delivered");
    expect(shipmentStage({ status: "Invoiced" })).toBe("Delivered");
    expect(shipmentStage({ status: "Canceled" })).toBe("Canceled");
  });
});

describe("stageLabel and STAGE_TONE", () => {
  it("names and colours every stage", () => {
    expect(stageLabel("Late", t)).toBe("Late");
    expect(stageLabel("NeedsCoverage", t)).toBe("Needs coverage");
    expect(stageLabel("Moving", t)).toBe("Moving");
    expect(stageLabel("Scheduled", t)).toBe("Scheduled");
    expect(stageLabel("Delivered", t)).toBe("Delivered");
    expect(stageLabel("Canceled", t)).toBe("Canceled");
    expect(STAGE_TONE).toEqual({
      Late: "warn",
      NeedsCoverage: "draft",
      Moving: "progress",
      Scheduled: "ink",
      Delivered: "success",
      Canceled: "muted",
    });
  });
});

describe("durationLabel", () => {
  it("says a span the way a dispatcher does, sign aside", () => {
    expect(durationLabel(45, t)).toBe("45m");
    expect(durationLabel(44.6, t)).toBe("45m");
    expect(durationLabel(80, t)).toBe("1h 20m");
    expect(durationLabel(60, t)).toBe("1h 0m");
    expect(durationLabel(3060, t)).toBe("2d 3h");
    expect(durationLabel(-80, t)).toBe("1h 20m");
    expect(durationLabel(0, t)).toBe("0m");
    expect(durationLabel(Number.NaN, t)).toBe("0m");
  });
});

describe("etaReading", () => {
  it("is nothing without an eta", () => {
    expect(etaReading(null, t)).toBeNull();
    expect(etaReading(undefined, t)).toBeNull();
  });

  it("reads every verdict", () => {
    const arrival = at(13, 9, 30);
    expect(
      etaReading({ verdict: "OnTime", slackMinutes: 80, estimatedArrival: arrival }, t),
    ).toEqual({ state: "ok", label: "On time", detail: "1h 20m to spare", at: arrival });
    expect(
      etaReading({ verdict: "AtRisk", slackMinutes: 15, estimatedArrival: arrival }, t),
    ).toEqual({ state: "warn", label: "At risk", detail: "only 15m to spare", at: arrival });
    expect(
      etaReading(
        { verdict: "Late", slackMinutes: -40, estimatedArrival: arrival, reason: "Weather" },
        t,
      ),
    ).toEqual({
      state: "bad",
      label: "Late",
      detail: "40m past the window · Weather",
      at: arrival,
    });
    expect(etaReading({ verdict: "Unknown", reason: "No position yet" }, t)).toEqual({
      state: "unknown",
      label: "No ETA yet",
      detail: "No position yet",
      at: null,
    });
  });

  it("says what it can when the slack or the arrival is missing", () => {
    expect(etaReading({ verdict: "Late", slackMinutes: null, reason: "" }, t)).toEqual({
      state: "bad",
      label: "Late",
      detail: "",
      at: null,
    });
    expect(etaReading({ verdict: "OnTime", estimatedArrival: 0 }, t)).toEqual({
      state: "ok",
      label: "On time",
      detail: "",
      at: null,
    });
    expect(etaReading({ verdict: "AtRisk", slackMinutes: -5, reason: "Low hours" }, t)).toEqual({
      state: "warn",
      label: "At risk",
      detail: "5m past the window · Low hours",
      at: null,
    });
    expect(etaReading({ verdict: "Unknown" }, t)?.detail).toBe("");
  });
});

describe("stopTypeLabel", () => {
  it("names every stop type", () => {
    expect(stopTypeLabel("Pickup", t)).toBe("Pickup");
    expect(stopTypeLabel("Delivery", t)).toBe("Delivery");
    expect(stopTypeLabel("SplitDelivery", t)).toBe("Split delivery");
    expect(stopTypeLabel("SplitPickup", t)).toBe("Split pickup");
  });
});

describe("windowLabel", () => {
  it("writes a window within a day as one range after its kind", () => {
    const label = windowLabel(
      journeyStop({ appointment: true, windowStart: at(12, 8), windowEnd: at(12, 10) }),
      t,
    );
    expect(label).toBe(`Appt ${range(at(12, 8), at(12, 10))}`);
    expect(windowLabel(journeyStop({ windowStart: at(12, 8), windowEnd: at(12, 16) }), t)).toBe(
      `Open ${range(at(12, 8), at(12, 16))}`,
    );
  });

  it("names both days when the window spans them", () => {
    const label = windowLabel(journeyStop({ windowStart: at(12, 8), windowEnd: at(13, 9) }), t);
    expect(label).toBe(`Open ${range(at(12, 8), at(13, 9))}`);
    expect(label).toContain(cardDay(at(13, 9)));
  });

  it("writes one moment when there is no usable end", () => {
    expect(windowLabel(journeyStop({ appointment: true, windowStart: at(12, 8) }), t)).toBe(
      `Appt ${cardMoment(at(12, 8))}`,
    );
    expect(
      windowLabel(
        journeyStop({ appointment: true, windowStart: at(12, 8), windowEnd: at(12, 7) }),
        t,
      ),
    ).toBe(`Appt ${cardMoment(at(12, 8))}`);
    expect(windowLabel(journeyStop({ windowEnd: at(12, 16) }), t)).toBe(
      `Open ${cardMoment(at(12, 16))}`,
    );
  });

  it("is empty with no window", () => {
    expect(windowLabel(journeyStop(), t)).toBe("");
  });

  it("reads a window as it comes from a stop", () => {
    const [first] = journeyStops(journeyMoves(), JOURNEY_NOW);
    expect(windowLabel(first, t)).toBe(`Appt ${range(at(12, 8), at(12, 9))}`);
  });
});

describe("punctualityLabel", () => {
  it("says on time, early or late by how much", () => {
    expect(punctualityLabel(null, t)).toBe("");
    expect(punctualityLabel({ state: "onTime", minutes: 0 }, t)).toBe("On time");
    expect(punctualityLabel({ state: "early", minutes: 20 }, t)).toBe("Early 20m");
    expect(punctualityLabel({ state: "late", minutes: 65 }, t)).toBe("Late 1h 5m");
  });
});

describe("actualsLabel", () => {
  const slot = { windowStart: at(12, 8), windowEnd: at(12, 10) };

  it("says when the truck came and went and how long it stayed", () => {
    expect(
      actualsLabel(
        journeyStop({
          ...slot,
          arrivedAt: at(12, 8, 12),
          departedAt: at(12, 9, 40),
          dwellMinutes: 88,
        }),
        t,
      ),
    ).toBe(`Arrived ${clock(at(12, 8, 12))} · left ${clock(at(12, 9, 40))} · 1h 28m on site`);
    expect(actualsLabel(journeyStop({ ...slot, arrivedAt: at(12, 8, 12) }), t)).toBe(
      `Arrived ${clock(at(12, 8, 12))}`,
    );
  });

  it("adds the day when it is not the window's", () => {
    expect(
      actualsLabel(
        journeyStop({
          ...slot,
          arrivedAt: at(12, 9),
          departedAt: at(13, 7),
          dwellMinutes: 1320,
        }),
        t,
      ),
    ).toBe(`Arrived ${clock(at(12, 9))} · left ${cardMoment(at(13, 7))} · 22h 0m on site`);
    expect(actualsLabel(journeyStop({ arrivedAt: at(12, 9) }), t)).toBe(
      `Arrived ${cardMoment(at(12, 9))}`,
    );
  });

  it("says a departure alone, and nothing with no actuals", () => {
    expect(actualsLabel(journeyStop({ ...slot, departedAt: at(12, 9) }), t)).toBe(
      `Left ${clock(at(12, 9))}`,
    );
    expect(actualsLabel(journeyStop(slot), t)).toBe("");
  });

  it("reads the actuals as they come from a stop", () => {
    const [first] = journeyStops(journeyMoves(), JOURNEY_NOW);
    expect(actualsLabel(first, t)).toBe(
      `Arrived ${clock(at(12, 7, 40))} · left ${clock(at(12, 8, 28))} · 48m on site`,
    );
  });
});

describe("coverageOf", () => {
  const moves = [
    move({
      id: "mB",
      sequence: 1,
      coverageType: "carrier",
      carrierAssignment: {
        id: "ca1",
        status: "Confirmed",
        carrier: { name: "Swift Transport", code: "SWFT" },
        totalCost: "1450.50" as unknown as number,
        currencyCode: "CAD",
        proNumber: "P-77",
        externalDriverName: "Lee Park",
      } as ShipmentMove["carrierAssignment"],
    }),
    move({
      id: "mA",
      sequence: 0,
      coverageType: "driver",
      assignment: {
        id: "as1",
        status: "InProgress",
        primaryWorkerId: "wrk_1",
        primaryWorker: { wholeName: "Ana Ruiz" },
        secondaryWorker: { firstName: "Ben", lastName: "Cole", wholeName: "" },
        tractor: { code: "T-12" },
        trailer: null,
      } as ShipmentMove["assignment"],
    }),
    move({
      id: "mC",
      sequence: 2,
      coverageType: "carrier",
      carrierAssignment: {
        id: "ca2",
        status: "Canceled",
        carrier: { name: "Old Carrier" },
      } as ShipmentMove["carrierAssignment"],
    }),
    move({ id: "mD", sequence: 3, coverageType: "unassigned" }),
    move({
      id: "mE",
      sequence: 4,
      assignment: {
        id: "as2",
        status: "Canceled",
        primaryWorkerId: "wrk_2",
        primaryWorker: { wholeName: "Gone Driver" },
      } as ShipmentMove["assignment"],
    }),
    move({
      id: "mF",
      sequence: 5,
      coverageType: "carrier",
      carrierAssignment: {
        id: "ca3",
        status: "Pending",
        carrier: null,
        totalCost: null,
        proNumber: null,
        externalDriverName: null,
      } as ShipmentMove["carrierAssignment"],
    }),
  ];

  it("reads one coverage per move, in move order", () => {
    expect(coverageOf(moves)).toEqual([
      {
        kind: "driver",
        moveId: "mA",
        moveIndex: 0,
        drivers: ["Ana Ruiz", "Ben Cole"],
        tractor: "T-12",
        trailer: "",
        status: "InProgress",
      },
      {
        kind: "carrier",
        moveId: "mB",
        moveIndex: 1,
        carrier: "Swift Transport",
        status: "Confirmed",
        proNumber: "P-77",
        cost: 1450.5,
        currency: "CAD",
        driver: "Lee Park",
      },
      { kind: "none", moveId: "mC", moveIndex: 2 },
      { kind: "none", moveId: "mD", moveIndex: 3 },
      { kind: "none", moveId: "mE", moveIndex: 4 },
      {
        kind: "carrier",
        moveId: "mF",
        moveIndex: 5,
        carrier: "",
        status: "Pending",
        proNumber: "",
        cost: null,
        currency: "USD",
        driver: "",
      },
    ]);
  });

  it("takes the driver when the move says a driver covers it and both are on file", () => {
    const both = move({
      id: "m1",
      coverageType: "driver",
      assignment: {
        id: "as1",
        status: "New",
        tractorId: "trc_1",
        tractor: { code: "T-1" },
      } as ShipmentMove["assignment"],
      carrierAssignment: {
        id: "ca1",
        status: "Pending",
        carrier: { name: "Swift Transport" },
      } as ShipmentMove["carrierAssignment"],
    });
    expect(coverageOf([both])).toEqual([
      {
        kind: "driver",
        moveId: "m1",
        moveIndex: 0,
        drivers: [],
        tractor: "T-1",
        trailer: "",
        status: "New",
      },
    ]);
  });

  it("is empty with no moves", () => {
    expect(coverageOf([])).toEqual([]);
  });
});

describe("initialsOf", () => {
  it("takes a name's first two initials, and a mark for no name", () => {
    expect(initialsOf("Ana Ruiz")).toBe("AR");
    expect(initialsOf("mary jane watson")).toBe("MJ");
    expect(initialsOf("Cher")).toBe("C");
    expect(initialsOf("")).toBe("?");
    expect(initialsOf("   ")).toBe("?");
  });
});

function commodityLine(
  fields: Partial<ShipmentCommodity>,
  commodity: Record<string, unknown> | null,
): ShipmentCommodity {
  return {
    commodityId: "com",
    pieces: 1,
    weight: 0,
    ...fields,
    commodity: commodity as ShipmentCommodity["commodity"],
  } as ShipmentCommodity;
}

describe("freightOf", () => {
  const lines = [
    commodityLine(
      { id: "sc1", pieces: 10, weight: 12000 },
      {
        name: "Frozen poultry",
        hazardousMaterialId: null,
        minTemperature: -10,
        maxTemperature: 10,
        freightClass: "Class70",
        fragile: false,
        stackable: true,
      },
    ),
    commodityLine(
      { id: "sc2", pieces: 4, weight: 3000 },
      {
        name: "Lithium batteries",
        hazardousMaterialId: "hm_1",
        minTemperature: -5,
        maxTemperature: null,
        freightClass: null,
        fragile: true,
        stackable: false,
      },
    ),
    commodityLine({ pieces: 1, weight: 0 }, null),
  ];

  it("adds up the commodities when the shipment has no figures of its own", () => {
    expect(
      freightOf({
        pieces: null,
        weight: null,
        temperatureMin: null,
        temperatureMax: null,
        commodities: lines,
      }),
    ).toEqual({
      pieces: 15,
      weight: 15000,
      temperature: { min: -5, max: 10 },
      hazmat: true,
      lines: [
        {
          id: "sc1",
          name: "Frozen poultry",
          pieces: 10,
          weight: 12000,
          hazmat: false,
          fragile: false,
          stackable: true,
          freightClass: "Class70",
        },
        {
          id: "sc2",
          name: "Lithium batteries",
          pieces: 4,
          weight: 3000,
          hazmat: true,
          fragile: true,
          stackable: false,
          freightClass: "",
        },
        {
          id: "commodity-2",
          name: "",
          pieces: 1,
          weight: 0,
          hazmat: false,
          fragile: false,
          stackable: null,
          freightClass: "",
        },
      ],
    });
  });

  it("keeps the shipment's own figures and range", () => {
    const summary = freightOf({
      pieces: 24,
      weight: 40000,
      temperatureMin: 34,
      temperatureMax: 38,
      commodities: [lines[0]],
    });
    expect(summary).toMatchObject({
      pieces: 24,
      weight: 40000,
      temperature: { min: 34, max: 38 },
      hazmat: false,
    });
    expect(
      freightOf({ temperatureMin: 34, temperatureMax: null, commodities: lines }).temperature,
    ).toEqual({ min: 34, max: null });
  });

  it("reads a zero on the shipment as no figure", () => {
    expect(freightOf({ pieces: 0, weight: 0, commodities: lines })).toMatchObject({
      pieces: 15,
      weight: 15000,
    });
    expect(freightOf({ pieces: 0, weight: 0, commodities: [] })).toMatchObject({
      pieces: 0,
      weight: 0,
    });
  });

  it("is empty for a shipment with no freight on file", () => {
    expect(freightOf({ commodities: [] })).toEqual({
      pieces: null,
      weight: null,
      temperature: null,
      hazmat: false,
      lines: [],
    });
    expect(freightOf({ commodities: undefined as unknown as ShipmentCommodity[] }).lines).toEqual(
      [],
    );
  });
});

describe("temperatureLabel", () => {
  it("writes a range, a floor or a ceiling", () => {
    expect(temperatureLabel({ min: 34, max: 38 }, t)).toBe("34–38°F");
    expect(temperatureLabel({ min: 34, max: null }, t)).toBe("≥ 34°F");
    expect(temperatureLabel({ min: null, max: 38 }, t)).toBe("≤ 38°F");
    expect(temperatureLabel({ min: 35, max: 35 }, t)).toBe("35°F");
    expect(temperatureLabel({ min: -10, max: 0 }, t)).toBe("-10–0°F");
    expect(temperatureLabel({ min: null, max: null }, t)).toBe("");
  });
});

describe("chargesOf", () => {
  const charged = shipment({
    freightChargeAmount: "2400.00" as unknown as number,
    totalChargeAmount: "2565.00" as unknown as number,
    ratingDetail: {
      formulaTemplateName: "Per mile",
      agreementName: "Acme 2026",
      ruleLabel: "Dry van TX-GA",
    } as Shipment["ratingDetail"],
    additionalCharges: [
      {
        id: "c1",
        method: "PerUnit",
        amount: 45,
        unit: 2,
        accessorialCharge: { code: "DET", description: "Detention" },
        isDetention: true,
      },
      {
        id: "c2",
        method: "Percentage",
        amount: 12,
        unit: 1,
        accessorialCharge: { code: "FSC", description: "" },
      },
      {
        id: "c3",
        method: "Flat",
        amount: 75,
        unit: 1,
        accessorialCharge: { code: "LUMP", description: "Lumper" },
      },
      { method: "Flat", amount: "35.00", unit: 1, accessorialCharge: null },
    ] as unknown as Shipment["additionalCharges"],
  });

  it("lists the linehaul first, then each charge, with the total", () => {
    expect(chargesOf(charged, t)).toEqual({
      lines: [
        { key: "linehaul", label: "Linehaul", detail: "Acme 2026 · Dry van TX-GA", amount: 2400 },
        { key: "c1", label: "Detention", detail: `2 × ${money(45)}`, amount: 90 },
        {
          key: "c2",
          label: "FSC",
          detail: `${formatPercent(12, 0)} of linehaul`,
          amount: null,
        },
        { key: "c3", label: "Lumper", detail: "", amount: 75 },
        { key: "charge-3", label: "Charge", detail: "", amount: 35 },
      ],
      total: 2565,
      ratedBy: "Rated by Acme 2026",
      overridden: false,
      overrideReason: "",
      locked: false,
    });
  });

  it("names the formula when no contract priced it, and says when the rate was overridden", () => {
    const charges = chargesOf(
      shipment({
        freightChargeAmount: 1800,
        totalChargeAmount: 1800,
        ratingDetail: { formulaTemplateName: "Per mile" } as Shipment["ratingDetail"],
        autoRated: true,
        rateOverrideAmount: 1800,
        rateOverrideReason: " Customer request ",
        rateLocked: true,
      }),
      t,
    );
    expect(charges.lines).toEqual([
      { key: "linehaul", label: "Linehaul", detail: "Per mile", amount: 1800 },
    ]);
    expect(charges).toMatchObject({
      total: 1800,
      ratedBy: "Auto-rated by Per mile",
      overridden: true,
      overrideReason: "Customer request",
      locked: true,
    });
  });

  it("falls back to the shipment's formula, and to nothing", () => {
    expect(
      chargesOf(
        shipment({
          ratingDetail: null,
          formulaTemplate: { name: "Flat rate" } as Shipment["formulaTemplate"],
        }),
        t,
      ).lines[0].detail,
    ).toBe("Flat rate");

    const bare = chargesOf(
      shipment({ freightChargeAmount: null, totalChargeAmount: null, additionalCharges: [] }),
      t,
    );
    expect(bare).toEqual({
      lines: [{ key: "linehaul", label: "Linehaul", detail: "", amount: null }],
      total: null,
      ratedBy: "",
      overridden: false,
      overrideReason: "",
      locked: false,
    });
  });

  it("writes a fractional percentage to two places", () => {
    const charges = chargesOf(
      shipment({
        additionalCharges: [
          { id: "f", method: "Percentage", amount: 12.5, unit: 3, accessorialCharge: null },
        ] as unknown as Shipment["additionalCharges"],
      }),
      t,
    );
    expect(charges.lines[1]).toEqual({
      key: "f",
      label: "Charge",
      detail: `${formatPercent(12.5, 2)} of linehaul`,
      amount: null,
    });
  });
});

describe("marginOf", () => {
  const estimate = {
    shipmentId: "shp_1",
    loadedMiles: 800,
    deadheadMiles: 50,
    totalMiles: 850,
    costPerMile: "1.80",
    estimatedCost: "1530.00",
    profit: "1035.00",
    marginPercent: "40.35",
    breakEvenRpm: "1.91",
    targetMarginPercent: "15",
    missingDistance: false,
  };

  it("is nothing without an estimate", () => {
    expect(marginOf({ profitabilityEstimate: null, totalChargeAmount: 2565 })).toBeNull();
    expect(marginOf({ totalChargeAmount: 2565 })).toBeNull();
  });

  it("reads the estimate and the rate per loaded mile", () => {
    expect(
      marginOf({
        profitabilityEstimate: estimate,
        totalChargeAmount: "2565.00" as unknown as number,
      }),
    ).toEqual({
      profit: 1035,
      percent: 40.35,
      target: 15,
      cost: 1530,
      loadedMiles: 800,
      ratePerMile: 2565 / 800,
      missingDistance: false,
    });
  });

  it("gives no rate per mile without loaded miles", () => {
    expect(
      marginOf({
        profitabilityEstimate: {
          ...estimate,
          loadedMiles: 0,
          marginPercent: null,
          targetMarginPercent: undefined,
          missingDistance: true,
        },
        totalChargeAmount: 2565,
      }),
    ).toMatchObject({ ratePerMile: null, percent: null, target: null, missingDistance: true });
    expect(
      marginOf({ profitabilityEstimate: estimate, totalChargeAmount: null })?.ratePerMile,
    ).toBeNull();
  });
});

describe("totalMiles", () => {
  it("adds up the distances of the moves still running", () => {
    expect(
      totalMiles([
        move({ id: "a", distance: 100 }),
        move({ id: "b", distance: null }),
        move({ id: "c", distance: 250, status: "Canceled" }),
        move({ id: "d", distance: 300 }),
      ]),
    ).toBe(400);
  });

  it("is nothing when no move has a distance", () => {
    expect(totalMiles([move({ id: "a" }), move({ id: "b", distance: undefined })])).toBeNull();
    expect(totalMiles([move({ id: "a", distance: 0 })])).toBeNull();
    expect(totalMiles([])).toBeNull();
  });
});

describe("billingPhase", () => {
  it("reads where billing stands", () => {
    expect(billingPhase({ status: "Invoiced" })).toBe("invoiced");
    expect(billingPhase({ status: "Completed", billedAt: at(14, 9) })).toBe("invoiced");
    expect(billingPhase({ status: "Completed" })).toBe("readiness");
    expect(billingPhase({ status: "ReadyToInvoice", billedAt: null })).toBe("readiness");
    expect(billingPhase({ status: "PartiallyCompleted", stage: "Delivered" })).toBe("readiness");
    expect(billingPhase({ status: "InTransit", stage: "Moving" })).toBe("notYet");
    expect(billingPhase({ status: "New" })).toBe("notYet");
  });
});

function requirement(
  fields: Partial<ShipmentBillingRequirement> & { documentTypeId: string },
): ShipmentBillingRequirement {
  return {
    documentTypeCode: "",
    documentTypeName: "",
    satisfied: false,
    documentCount: 0,
    documentIds: [],
    ineligibleDocuments: [],
    ...fields,
  };
}

function readiness(fields: Partial<ShipmentBillingReadiness>): ShipmentBillingReadiness {
  return {
    shipmentId: "shp_1",
    shipmentStatus: "Completed",
    policy: {} as ShipmentBillingReadiness["policy"],
    requirements: [],
    missingRequirements: [],
    validationFailures: [],
    warnings: [],
    serviceFailureContext: { hasUnresolved: false, unresolvedCount: 0, serviceFailureIds: [] },
    canMarkReadyToInvoice: false,
    shouldAutoMarkReadyToInvoice: false,
    shouldAutoTransferToBilling: false,
    shouldAutoApproveBilling: false,
    payers: [],
    ...fields,
  };
}

describe("readinessChecks", () => {
  it("checks each requirement, failure, service failure and warning", () => {
    const bol = requirement({
      documentTypeId: "dt-bol",
      documentTypeCode: "BOL",
      documentTypeName: "Bill of lading",
      documentCount: 3,
      ineligibleDocuments: [
        { documentId: "d1", standing: "rejected" },
        { documentId: "d2", standing: "pending_review" },
        { documentId: "d3", standing: "rejected" },
      ],
    });
    const checks = readinessChecks(
      readiness({
        requirements: [
          requirement({
            documentTypeId: "dt-pod",
            documentTypeCode: "POD",
            documentTypeName: "Proof of delivery",
            satisfied: true,
            documentCount: 1,
          }),
          bol,
          requirement({
            documentTypeId: "dt-inv",
            documentTypeCode: "INV",
            satisfied: true,
            documentCount: 2,
            ineligibleDocuments: [{ documentId: "d9", standing: "expired" }],
          }),
        ],
        missingRequirements: [
          bol,
          requirement({
            documentTypeId: "dt-rc",
            documentTypeCode: "RC",
            documentTypeName: "Rate confirmation",
          }),
        ],
        validationFailures: [
          { field: "billToCustomerId", code: "required", message: "Bill-to customer is required" },
        ],
        serviceFailureContext: {
          hasUnresolved: true,
          unresolvedCount: 2,
          serviceFailureIds: ["sf1", "sf2"],
        },
        warnings: [{ code: "rate_variance", message: "Rate differs from the contract" }],
      }),
      t,
    );

    expect(checks).toEqual([
      {
        key: "requirement:dt-pod",
        state: "ok",
        title: "Proof of delivery",
        detail: "1 document on file",
      },
      {
        key: "requirement:dt-bol",
        state: "bad",
        title: "Bill of lading",
        detail: "2 rejected, 1 pending review",
      },
      {
        key: "requirement:dt-inv",
        state: "ok",
        title: "INV",
        detail: "2 documents on file · 1 expired",
      },
      {
        key: "requirement:dt-rc",
        state: "bad",
        title: "Rate confirmation",
        detail: "Not on file",
      },
      {
        key: "validation:billToCustomerId:required:0",
        state: "bad",
        title: "Bill-to customer is required",
        detail: "",
      },
      {
        key: "service-failures",
        state: "warn",
        title: "2 unresolved service failures",
        detail: "Review before invoicing",
      },
      {
        key: "warning:rate_variance:0",
        state: "warn",
        title: "Rate differs from the contract",
        detail: "",
      },
    ]);
  });

  it("holds a requirement the server lists as missing against it", () => {
    const pod = requirement({
      documentTypeId: "dt-pod",
      documentTypeName: "Proof of delivery",
      satisfied: true,
      documentCount: 1,
    });
    expect(
      readinessChecks(readiness({ requirements: [pod], missingRequirements: [pod] }), t),
    ).toEqual([
      {
        key: "requirement:dt-pod",
        state: "bad",
        title: "Proof of delivery",
        detail: "Not on file",
      },
    ]);
  });

  it("counts service failures by their ids when the count is missing", () => {
    expect(
      readinessChecks(
        readiness({
          serviceFailureContext: {
            hasUnresolved: true,
            unresolvedCount: 0,
            serviceFailureIds: ["sf1"],
          },
        }),
        t,
      ),
    ).toEqual([
      {
        key: "service-failures",
        state: "warn",
        title: "1 unresolved service failure",
        detail: "Review before invoicing",
      },
    ]);
  });

  it("survives a payload without warnings or service failure context", () => {
    const raw = {
      ...readiness({}),
      warnings: undefined,
      serviceFailureContext: undefined,
    } as unknown as ShipmentBillingReadiness;
    expect(readinessChecks(raw, t)).toEqual([]);
  });
});

describe("nowReading", () => {
  it("says a canceled shipment is canceled, why and when", () => {
    expect(
      nowReading(
        shipment({
          status: "Canceled",
          cancelReason: "Customer canceled",
          canceledAt: at(11, 9),
        }),
        [],
        t,
        at(12, 8),
      ),
    ).toEqual({
      tone: "muted",
      title: "Canceled",
      detail: `Customer canceled · ${cardDay(at(11, 9))}`,
    });
    expect(
      nowReading(shipment({ status: "InTransit", stage: "Canceled" }), [], t, at(12, 8)),
    ).toEqual({ tone: "muted", title: "Canceled", detail: "" });
  });

  describe("delivered", () => {
    function delivered(arrival: number, extra: Partial<Shipment> = {}) {
      const moves = [
        move({
          id: "m1",
          status: "Completed",
          stops: [
            stop({
              id: "p",
              status: "Completed",
              scheduledWindowStart: at(12, 8),
              scheduledWindowEnd: at(12, 9),
              actualArrival: at(12, 8, 10),
              actualDeparture: at(12, 8, 50),
            }),
            stop({
              id: "d",
              type: "Delivery",
              status: "Completed",
              sequence: 1,
              scheduledWindowStart: at(12, 12),
              scheduledWindowEnd: at(12, 14),
              actualArrival: arrival,
              actualDeparture: at(12, 15, 10),
            }),
            stop({ id: "x", type: "Delivery", status: "Canceled", sequence: 2 }),
          ],
        }),
      ];
      const record = shipment({ status: "Completed", moves, ...extra });
      return nowReading(record, journeyStops(moves, at(12, 18)), t, at(12, 18));
    }

    it("says when it was delivered and how the last stop went", () => {
      expect(delivered(at(12, 14, 30), { actualDeliveryDate: at(12, 15, 30) })).toEqual({
        tone: "success",
        title: `Delivered ${cardMoment(at(12, 15, 30))}`,
        detail: "30m late at the last stop",
      });
      expect(delivered(at(12, 13)).detail).toBe("On time at the last stop");
      expect(delivered(at(12, 11, 30)).detail).toBe("30m early at the last stop");
    });

    it("takes the last stop's departure when the shipment has no delivery date", () => {
      expect(delivered(at(12, 13)).title).toBe(`Delivered ${cardMoment(at(12, 15, 10))}`);
    });

    it("says delivered alone with nothing on record", () => {
      expect(
        nowReading(shipment({ status: "Invoiced", stage: "Delivered" }), [], t, at(12, 8)),
      ).toEqual({ tone: "success", title: "Delivered", detail: "" });
    });
  });

  describe("needs coverage", () => {
    const driver = {
      id: "as1",
      status: "New",
      primaryWorkerId: "wrk_1",
      primaryWorker: { wholeName: "Ana Ruiz" },
    } as ShipmentMove["assignment"];

    it("counts the running moves nobody covers", () => {
      expect(
        nowReading(
          shipment({
            status: "PartiallyAssigned",
            moves: [
              move({ id: "m1" }),
              move({ id: "m2", sequence: 1, assignment: driver }),
              move({ id: "m3", sequence: 2 }),
              move({ id: "m4", sequence: 3, status: "Canceled" }),
            ],
          }),
          [],
          t,
          at(12, 8),
        ),
      ).toEqual({
        tone: "draft",
        title: "Needs coverage",
        detail: "2 of 3 moves have no driver or carrier",
      });
      expect(
        nowReading(
          shipment({
            moves: [move({ id: "m1" }), move({ id: "m2", sequence: 1, assignment: driver })],
          }),
          [],
          t,
          at(12, 8),
        ).detail,
      ).toBe("1 of 2 moves has no driver or carrier");
    });

    it("says when there is nothing to cover yet, and nothing when all is covered", () => {
      expect(nowReading(shipment({ status: "New" }), [], t, at(12, 8)).detail).toBe(
        "No moves planned yet",
      );
      expect(
        nowReading(
          shipment({ stage: "NeedsCoverage", moves: [move({ id: "m1", assignment: driver })] }),
          [],
          t,
          at(12, 8),
        ).detail,
      ).toBe("");
    });
  });

  describe("on the road", () => {
    it("says where the truck is, how it arrived and what the ETA says", () => {
      const moves = journeyMoves();
      const stops = journeyStops(moves, JOURNEY_NOW);
      const current = stops[2];
      const reading = nowReading(
        shipment({
          status: "InTransit",
          stage: "Moving",
          moves,
          eta: { verdict: "AtRisk", slackMinutes: 15, estimatedArrival: at(13, 9, 45) },
        }),
        stops,
        t,
        JOURNEY_NOW,
      );
      expect(reading).toEqual({
        tone: "warn",
        title: "At Shreveport, LA",
        detail: [windowLabel(current, t), "On time", "At risk, only 15m to spare"].join(" · "),
      });
    });

    it("keeps the stage's tone when the ETA is fine or unknown", () => {
      const moves = journeyMoves();
      const stops = journeyStops(moves, JOURNEY_NOW);
      const windowText = windowLabel(stops[2], t);

      expect(nowReading(shipment({ status: "InTransit", moves }), stops, t, JOURNEY_NOW)).toEqual({
        tone: "progress",
        title: "At Shreveport, LA",
        detail: `${windowText} · On time`,
      });
      expect(
        nowReading(
          shipment({ status: "InTransit", moves, eta: { verdict: "Unknown", reason: "No ping" } }),
          stops,
          t,
          JOURNEY_NOW,
        ),
      ).toEqual({
        tone: "progress",
        title: "At Shreveport, LA",
        detail: `${windowText} · On time`,
      });
    });

    const scheduled = [
      move({
        id: "m1",
        stops: [
          stop({
            id: "p",
            scheduleType: "Appointment",
            scheduledWindowStart: at(13, 8),
            scheduledWindowEnd: at(13, 10),
            location: location("Peach DC", "Atlanta", "GA"),
          }),
          stop({ id: "d", type: "Delivery", sequence: 1 }),
        ],
      }),
    ];

    it("names the next stop and its window", () => {
      const stops = journeyStops(scheduled, at(12, 12));
      expect(
        nowReading(shipment({ status: "Assigned", moves: scheduled }), stops, t, at(12, 12)),
      ).toEqual({
        tone: "ink",
        title: "Next: Pickup at Atlanta, GA",
        detail: `Appt ${range(at(13, 8), at(13, 10))}`,
      });
    });

    it("says how late the next stop is running by the time it is read", () => {
      const stops = journeyStops(scheduled, at(12, 12));
      expect(
        nowReading(shipment({ status: "Assigned", moves: scheduled }), stops, t, at(13, 11)),
      ).toEqual({
        tone: "warn",
        title: "Next: Pickup at Atlanta, GA",
        detail: `Appt ${range(at(13, 8), at(13, 10))} · Late 1h 0m`,
      });
    });

    it("names the stop type alone when the stop has no place", () => {
      const moves = [
        move({
          id: "m1",
          stops: [
            stop({ id: "p", status: "Completed" }),
            stop({ id: "d", type: "Delivery", sequence: 1 }),
          ],
        }),
      ];
      expect(
        nowReading(
          shipment({ status: "InTransit", moves }),
          journeyStops(moves, at(12, 8)),
          t,
          at(12, 8),
        ),
      ).toEqual({ tone: "progress", title: "Next: Delivery", detail: "" });
    });

    it("falls back to the stage when no stop is open", () => {
      const moves = [move({ id: "m1", stops: [stop({ id: "p", status: "Completed" })] })];
      const stops = journeyStops(moves, at(12, 8));
      expect(nowReading(shipment({ status: "InTransit", moves }), stops, t, at(12, 8))).toEqual({
        tone: "progress",
        title: "Moving",
        detail: "",
      });
      expect(
        nowReading(
          shipment({
            status: "InTransit",
            moves,
            eta: { verdict: "OnTime", slackMinutes: 80, estimatedArrival: at(12, 12) },
          }),
          stops,
          t,
          at(12, 8),
        ),
      ).toEqual({ tone: "progress", title: "Moving", detail: "On time, 1h 20m to spare" });
      expect(
        nowReading(
          shipment({
            status: "Delayed",
            eta: { verdict: "Late", slackMinutes: -40, reason: "Weather" },
          }),
          [],
          t,
          at(12, 8),
        ),
      ).toEqual({
        tone: "warn",
        title: "Late",
        detail: "Late, 40m past the window · Weather",
      });
    });
  });
});

describe("stopOffsets", () => {
  it("gives each move the share of the strip its miles do, and spreads its stops across it", () => {
    const moves = [
      move({ id: "m2", sequence: 1, distance: 300 }),
      move({ id: "m1", sequence: 0, distance: 100 }),
    ];
    const stops = [
      journeyStop({ id: "a", moveIndex: 0 }),
      journeyStop({ id: "b", moveIndex: 0 }),
      journeyStop({ id: "c", moveIndex: 1 }),
      journeyStop({ id: "d", moveIndex: 1 }),
    ];

    expect(stopOffsets(moves, stops)).toEqual([0, 0.25, 0.25, 1]);
  });

  it("spaces the stops evenly when any move has no distance", () => {
    const moves = [move({ id: "m1", distance: 120 }), move({ id: "m2", sequence: 1 })];
    const stops = ["a", "b", "c"].map((id, index) =>
      journeyStop({ id, moveIndex: index < 2 ? 0 : 1 }),
    );

    expect(stopOffsets(moves, stops)).toEqual([0, 0.5, 1]);
  });

  it("puts a lone stop, or none, at the start", () => {
    expect(stopOffsets([], [journeyStop()])).toEqual([0]);
    expect(stopOffsets([], [])).toEqual([]);
  });
});

describe("truckOffset", () => {
  const offsets = [0, 0.4, 1];

  it("stands the truck at the stop it is at", () => {
    const stops = [
      journeyStop({ id: "a", phase: "done" }),
      journeyStop({ id: "b", phase: "current" }),
      journeyStop({ id: "c", phase: "later" }),
    ];

    expect(truckOffset(stops, offsets, 0.9)).toBe(0.4);
  });

  it("never draws the truck past a stop it has not reached, or behind one it has left", () => {
    const stops = [
      journeyStop({ id: "a", phase: "done" }),
      journeyStop({ id: "b", phase: "next" }),
      journeyStop({ id: "c", phase: "later" }),
    ];

    expect(truckOffset(stops, offsets, 0.9)).toBe(0.4);
    expect(truckOffset(stops, offsets, 0.2)).toBe(0.2);
    expect(truckOffset(stops, [0.3, 0.4, 1], 0.1)).toBe(0.3);
  });
});

describe("marginGauge", () => {
  const margin = {
    profit: 300,
    percent: 15,
    target: 20,
    cost: 1700,
    loadedMiles: 500,
    ratePerMile: 4,
    missingDistance: false,
  };

  it("fills the bar against twice the target and marks the target", () => {
    expect(marginGauge(margin)).toEqual({ fill: 15 / 40, tick: 0.5, tone: "warn" });
  });

  it("reads a margin at its target as fine, under water as bad, and none as unknown", () => {
    expect(marginGauge({ ...margin, percent: 25 }).tone).toBe("ok");
    expect(marginGauge({ ...margin, percent: -5 })).toMatchObject({ fill: 0, tone: "bad" });
    expect(marginGauge({ ...margin, percent: null })).toMatchObject({ fill: 0, tone: "unknown" });
    expect(marginGauge({ ...margin, target: null, percent: 60 })).toEqual({
      fill: 1,
      tick: null,
      tone: "ok",
    });
  });
});

describe("freightClassLabel", () => {
  it("reads the NMFC class as it is written", () => {
    expect(freightClassLabel("Class77_5", t)).toBe("Class 77.5");
    expect(freightClassLabel("Class100", t)).toBe("Class 100");
    expect(freightClassLabel("", t)).toBe("");
  });
});

describe("coverageSummary and the status labels", () => {
  it("names who covers a move in a few words", () => {
    expect(
      coverageSummary(
        {
          kind: "driver",
          moveId: "m1",
          moveIndex: 0,
          drivers: ["Ana Ruiz", "Ben Ode"],
          tractor: "T-12",
          trailer: "",
          status: "New",
        },
        t,
      ),
    ).toBe("Ana Ruiz and Ben Ode");
    expect(coverageSummary({ kind: "none", moveId: "m1", moveIndex: 0 }, t)).toBe(
      "No driver or carrier yet",
    );
  });

  it("says each assignment, carrier and tender status in words", () => {
    expect(assignmentStatusLabel("InProgress", t)).toBe("Under way");
    expect(carrierStatusLabel("Confirmed", t)).toBe("Confirmed");
    expect(tenderStatusLabel("Accepted", t)).toBe("Tender accepted");
  });
});
