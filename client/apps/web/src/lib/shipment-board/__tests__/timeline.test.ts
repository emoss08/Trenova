import type { CapacityUnit } from "@/lib/graphql/shipment-board";
import type { Shipment } from "@trenova/shared/types/shipment";
import { packLane } from "@/lib/timeline/lanes";
import { describe, expect, it } from "vitest";
import {
  buildCoverageLanes,
  buildTimelineLoads,
  checkAssignment,
  dayStartOfIso,
  groupLoadsByStage,
  isoDay,
  rangeFirstDay,
  type TimelineLoad,
} from "../timeline";

/* Fixtures follow the ShipmentFields fragment: moves carry stops and an
   assignment, the ETA carries every open stop's projection. Times are Unix
   seconds from a fixed Tuesday in UTC. */
const TUE = Date.UTC(2026, 10, 3) / 1000;
const HOUR = 3_600;
const at = (hours: number) => TUE + hours * HOUR;

type StopSpec = {
  id: string;
  city: string;
  from: number;
  to?: number | null;
  status?: "New" | "InTransit" | "Completed" | "Canceled";
  type?: "Pickup" | "Delivery";
  arrived?: number | null;
  departed?: number | null;
};

function stop(spec: StopSpec) {
  return {
    id: spec.id,
    type: spec.type ?? "Pickup",
    status: spec.status ?? "New",
    scheduledWindowStart: at(spec.from),
    scheduledWindowEnd: spec.to == null ? null : at(spec.to),
    actualArrival: spec.arrived == null ? null : at(spec.arrived),
    actualDeparture: spec.departed == null ? null : at(spec.departed),
    location: { city: spec.city, state: { abbreviation: "TX" } },
  };
}

function shipment(
  id: string | undefined,
  stops: StopSpec[],
  extra: Record<string, unknown> = {},
): Shipment {
  return {
    id,
    proNumber: id ? `PRO-${id}` : undefined,
    stage: "Moving",
    status: "InTransit",
    eta: null,
    tenderStatus: null,
    moves: [
      { id: `mov_${id}`, status: "InTransit", coverageType: "unassigned", stops: stops.map(stop) },
    ],
    ...extra,
  } as unknown as Shipment;
}

function assigned(workerId: string, name: string, tractorId: string | null, code: string | null) {
  return {
    coverageType: "driver",
    assignment: {
      tractorId,
      primaryWorker: { id: workerId, firstName: name.split(" ")[0], lastName: name.split(" ")[1] },
      tractor: code ? { id: tractorId, code } : null,
    },
  };
}

function withMove(s: Shipment, move: Record<string, unknown>): Shipment {
  return { ...s, moves: [{ ...s.moves[0], ...move }] } as Shipment;
}

const NOW = at(10);

describe("days", () => {
  it("reads an ISO date back to the start of that day, and refuses anything else", () => {
    const start = dayStartOfIso("2026-11-03", "UTC");

    expect(start).toBe(TUE);
    expect(isoDay(TUE + 5 * HOUR, "UTC")).toBe("2026-11-03");
    expect(dayStartOfIso("next tuesday", "UTC")).toBeNull();
  });

  it("opens a week on its Monday", () => {
    expect(isoDay(rangeFirstDay("week", TUE, "UTC"), "UTC")).toBe("2026-11-02");
    expect(rangeFirstDay("day", TUE, "UTC")).toBe(TUE);
  });
});

describe("buildTimelineLoads", () => {
  it("spans a load from its first window to its last, reaching out to a late ETA", () => {
    const [load] = buildTimelineLoads([
      shipment(
        "late",
        [
          { id: "p", city: "Laredo", from: -14, to: -13, status: "Completed", departed: -13.25 },
          { id: "d", city: "Dallas", from: 8, to: 9, type: "Delivery" },
        ],
        {
          stage: "Late",
          eta: {
            stopId: "d",
            estimatedArrival: at(10.9),
            stops: [{ stopId: "d", arrived: false, estimatedArrival: at(10.9), driveMinutes: 520 }],
          },
        },
      ),
    ]);

    expect(load.start).toBe(at(-14));
    expect(load.end).toBe(at(10.9));
    expect(load.eta).toEqual({ stopIndex: 1, at: at(10.9), lateMinutes: 114 });
    expect(load.driven).toBe("underway");
    expect(load.driveMinutes).toBe(520);
    expect(load.stops.map((s) => s.place)).toEqual(["Laredo, TX", "Dallas, TX"]);
  });

  it("takes the shipment's own ETA for the next stop when it has no per-stop projection", () => {
    const [load] = buildTimelineLoads([
      shipment(
        "a",
        [
          { id: "p", city: "Dallas", from: 6, to: 7, status: "Completed" },
          { id: "d", city: "Memphis", from: 15, to: 16, type: "Delivery" },
        ],
        { eta: { stopId: "d", estimatedArrival: at(14.5) } },
      ),
    ]);

    expect(load.eta).toEqual({ stopIndex: 1, at: at(14.5), lateMinutes: 0 });
    expect(load.driveMinutes).toBeNull();
  });

  it("takes lateness from the server's slack when the projection has it", () => {
    const [load] = buildTimelineLoads([
      shipment(
        "slack",
        [
          { id: "p", city: "Dallas", from: 6, to: 7, status: "Completed" },
          { id: "d", city: "Memphis", from: 15, to: 16, type: "Delivery" },
        ],
        {
          stage: "Late",
          eta: {
            stopId: "d",
            estimatedArrival: at(17),
            stops: [{ stopId: "d", arrived: false, estimatedArrival: at(17), slackMinutes: -75 }],
          },
        },
      ),
    ]);

    expect(load.eta?.lateMinutes).toBe(75);
  });

  it("leaves out canceled stops, and loads with no id, no stage it draws or no appointment", () => {
    const loads = buildTimelineLoads([
      shipment("kept", [
        { id: "p", city: "Dallas", from: 6, to: 7 },
        { id: "x", city: "Waco", from: 9, to: 10, status: "Canceled" },
        { id: "d", city: "Austin", from: 12, to: 13, type: "Delivery" },
      ]),
      shipment(undefined, [{ id: "p", city: "Dallas", from: 6 }]),
      shipment("canceled", [{ id: "p", city: "Dallas", from: 6 }], { stage: "Canceled" }),
      shipment("unscheduled", [{ id: "p", city: "Dallas", from: 6 }], {
        moves: [
          {
            id: "m",
            status: "New",
            stops: [{ id: "p", status: "New", scheduledWindowStart: 0 }],
          },
        ],
      }),
    ]);

    expect(loads.map((l) => l.id)).toEqual(["kept"]);
    expect(loads[0].stops.map((s) => s.city)).toEqual(["Dallas", "Austin"]);
  });

  it("draws a delivered load as driven end to end and one not started as not driven", () => {
    const [delivered, scheduled] = buildTimelineLoads([
      shipment(
        "done",
        [
          { id: "p", city: "Dallas", from: 1, to: 2, status: "Completed" },
          {
            id: "d",
            city: "Waco",
            from: 5,
            to: 6,
            status: "Completed",
            type: "Delivery",
            departed: 6.5,
          },
        ],
        { stage: "Delivered" },
      ),
      shipment("later", [{ id: "p", city: "Dallas", from: 14, to: 15 }], { stage: "Scheduled" }),
    ]);

    expect(delivered.driven).toBe("all");
    expect(delivered.end).toBe(at(6.5));
    expect(scheduled.driven).toBe("none");
  });
});

describe("groupLoadsByStage", () => {
  it("leads with the latest load, then orders each stage by start", () => {
    const loads = buildTimelineLoads([
      shipment("moving-late-start", [{ id: "a", city: "A", from: 9 }]),
      shipment("late-little", [{ id: "b", city: "B", from: 1, to: 2 }], {
        stage: "Late",
        eta: { stopId: "b", estimatedArrival: at(2.5) },
      }),
      shipment("moving-early", [{ id: "c", city: "C", from: 3 }]),
      shipment("late-lot", [{ id: "d", city: "D", from: 4, to: 5 }], {
        stage: "Late",
        eta: { stopId: "d", estimatedArrival: at(8) },
      }),
    ]);

    expect(groupLoadsByStage(loads).map((g) => [g.stage, g.loads.map((l) => l.id)])).toEqual([
      ["Late", ["late-lot", "late-little"]],
      ["Moving", ["moving-early", "moving-late-start"]],
    ]);
  });
});

function load(
  id: string,
  from: number,
  to: number,
  extra: Record<string, unknown> = {},
): TimelineLoad {
  return buildTimelineLoads([
    shipment(
      id,
      [
        { id: `${id}-p`, city: "Dallas", from, to: from },
        { id: `${id}-d`, city: "Waco", from: to, to, type: "Delivery" },
      ],
      extra,
    ),
  ])[0];
}

/** A load on a driver's assignment: the worker, their tractor, its code. */
function driven(
  id: string,
  from: number,
  to: number,
  workerId: string,
  name: string,
  tractorId: string,
): TimelineLoad {
  return buildTimelineLoads([
    withMove(load(id, from, to).shipment, assigned(workerId, name, tractorId, `T-${tractorId}`)),
  ])[0];
}

describe("buildCoverageLanes", () => {
  const capacityDriver = (id: string, name: string, tractorId: string | null) =>
    ({
      id,
      name,
      initials: name.slice(0, 2).toUpperCase(),
      tractorId,
      unitLabel: tractorId ? `T-${id}` : null,
      driveRemainingMs: 6 * HOUR * 1000,
    }) as unknown as CapacityUnit;

  it("puts uncovered and tendered loads in the pool and gives free drivers a lane", () => {
    const lanes = buildCoverageLanes(
      [
        load("open", 14, 19),
        load("tendered", 18, 22, { tenderStatus: "Tendered" }),
        driven("busy", 8, 12, "wrk_1", "Marco Ortiz", "trc_1"),
      ],
      [capacityDriver("wrk_2", "Kim Johnson", "trc_2")],
    );

    expect(lanes.pool.map((l) => l.id)).toEqual(["open", "tendered"]);
    expect(lanes.drivers.map((d) => [d.id, d.name, d.tractorId, d.loads.map((l) => l.id)])).toEqual(
      [
        ["wrk_1", "Marco Ortiz", "trc_1", ["busy"]],
        ["wrk_2", "Kim Johnson", "trc_2", []],
      ],
    );
  });

  it("leads with double-booked drivers", () => {
    const lanes = buildCoverageLanes(
      [
        driven("calm", 6, 8, "wrk_calm", "Lin Chen", "trc_c"),
        driven("x", 9, 14, "wrk_double", "Ray Williams", "trc_d"),
        driven("y", 12, 16, "wrk_double", "Ray Williams", "trc_d"),
      ],
      [],
    );

    expect(lanes.drivers.map((d) => d.id)).toEqual(["wrk_double", "wrk_calm"]);
  });
});

describe("checkAssignment", () => {
  const lane = (tractorId: string | null, loads: TimelineLoad[] = []) => ({
    id: "wrk_1",
    name: "Kim Johnson",
    initials: "KJ",
    tractorId,
    tractorCode: tractorId ? "T-1031" : null,
    driveRemainingMs: null,
    loads,
    packed: packLane(loads),
  });
  const needing = (minutes: number) => ({
    ...load("new", 14, 19),
    driveMinutes: minutes,
  });

  it("refuses a driver with no tractor", () => {
    expect(checkAssignment(needing(60), lane(null), null, NOW).blocked).toBe(true);
  });

  it("lists the driver's loads the new one overlaps", () => {
    const check = checkAssignment(
      needing(60),
      lane("trc_1", [load("early", 8, 12), load("late", 18, 21)]),
      null,
      NOW,
    );

    expect(check.blocked).toBe(false);
    expect(check.overlaps.map((l) => l.id)).toEqual(["late"]);
  });

  it("does not weigh today's drive time against a load that picks up after a reset", () => {
    const tomorrow = { ...load("tomorrow", 30, 36), driveMinutes: 600 };

    expect(checkAssignment(tomorrow, lane("trc_1"), HOUR * 1000, NOW).driveShort).toBeNull();
  });

  it("warns when the load needs more driving than the driver has left", () => {
    expect(checkAssignment(needing(300), lane("trc_1"), 4 * HOUR * 1000, NOW).driveShort).toEqual({
      neededMinutes: 300,
      remainingMs: 4 * HOUR * 1000,
    });
    expect(
      checkAssignment(needing(200), lane("trc_1"), 4 * HOUR * 1000, NOW).driveShort,
    ).toBeNull();
    expect(checkAssignment(needing(200), lane("trc_1"), null, NOW).driveShort).toBeNull();
  });
});
