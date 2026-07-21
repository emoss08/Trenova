import type { Shipment } from "@/types/shipment";
import { describe, expect, it } from "vitest";
import {
  buildTimelineData,
  DWELL_CRITICAL_SECONDS,
  DWELL_WATCH_SECONDS,
  sortTimelineRows,
  UNASSIGNED_ROW_KEY,
} from "./use-timeline-data";

const RANGE = { start: 1_000_000, end: 1_086_400 }; // one-day window
const NOW = RANGE.start;

type StopSeed = {
  start: number;
  end?: number | null;
  actualArrival?: number | null;
  actualDeparture?: number | null;
};

type MoveSeed = {
  id: string;
  status?: string;
  workerId?: string;
  workerName?: string;
  workerFirstName?: string;
  workerLastName?: string;
  tractorCode?: string;
  stops: StopSeed[];
};

function makeShipment(id: string, status: string, moves: MoveSeed[]): Shipment {
  return {
    id,
    proNumber: `PRO-${id}`,
    status,
    moves: moves.map((move, moveIndex) => ({
      id: move.id,
      status: move.status ?? "Assigned",
      sequence: moveIndex,
      loaded: true,
      stops: move.stops.map((stop, stopIndex) => ({
        id: `${move.id}-stop-${stopIndex}`,
        locationId: "loc_1",
        status: "New",
        type: stopIndex === 0 ? "Pickup" : "Delivery",
        scheduleType: "Appointment",
        sequence: stopIndex,
        scheduledWindowStart: stop.start,
        scheduledWindowEnd: stop.end ?? null,
        actualArrival: stop.actualArrival ?? null,
        actualDeparture: stop.actualDeparture ?? null,
        location: { code: `LOC${stopIndex}`, name: `Location ${stopIndex}` },
      })),
      assignment: move.workerId
        ? {
            id: `asn-${move.id}`,
            status: "New",
            primaryWorkerId: move.workerId,
            primaryWorker: {
              id: move.workerId,
              wholeName: move.workerName ?? null,
              firstName: move.workerFirstName ?? null,
              lastName: move.workerLastName ?? null,
              profilePicUrl: null,
            },
            tractor: move.tractorCode ? { id: "trk_1", code: move.tractorCode } : null,
          }
        : null,
    })),
  } as unknown as Shipment;
}

describe("buildTimelineData", () => {
  it("groups moves into driver rows and an unassigned row", () => {
    const shipments = [
      makeShipment("shp_1", "InTransit", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          workerName: "Alice Driver",
          tractorCode: "TRK-100",
          stops: [
            { start: RANGE.start + 3600, end: RANGE.start + 7200 },
            { start: RANGE.start + 20_000, end: RANGE.start + 24_000 },
          ],
        },
      ]),
      makeShipment("shp_2", "New", [
        {
          id: "mov_2",
          stops: [{ start: RANGE.start + 10_000, end: RANGE.start + 14_000 }],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 2, RANGE, NOW);

    expect(data.rows).toHaveLength(1);
    expect(data.rows[0].workerName).toBe("Alice Driver");
    expect(data.rows[0].equipmentCodes).toEqual(["TRK-100"]);
    expect(data.rows[0].bars[0].start).toBe(RANGE.start + 3600);
    expect(data.rows[0].bars[0].end).toBe(RANGE.start + 24_000);
    expect(data.unassignedRow?.key).toBe(UNASSIGNED_ROW_KEY);
    expect(data.unassignedRow?.bars).toHaveLength(1);
    expect(data.truncated).toBe(false);
  });

  it("packs overlapping bars on the same driver into separate lanes", () => {
    const shipments = [
      makeShipment("shp_1", "Assigned", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          stops: [{ start: RANGE.start + 1000, end: RANGE.start + 30_000 }],
        },
      ]),
      makeShipment("shp_2", "Assigned", [
        {
          id: "mov_2",
          workerId: "wkr_a",
          stops: [{ start: RANGE.start + 5000, end: RANGE.start + 20_000 }],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 2, RANGE, NOW);

    expect(data.rows).toHaveLength(1);
    expect(data.rows[0].laneCount).toBe(2);
    expect(new Set(data.rows[0].bars.map((b) => b.laneIndex))).toEqual(new Set([0, 1]));
  });

  it("composes the driver name from first/last when wholeName is missing", () => {
    const shipments = [
      makeShipment("shp_1", "InTransit", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          workerFirstName: "Alice",
          workerLastName: "Driver",
          stops: [{ start: RANGE.start + 3600, end: RANGE.start + 7200 }],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 1, RANGE, NOW);

    expect(data.rows).toHaveLength(1);
    expect(data.rows[0].workerName).toBe("Alice Driver");
  });

  it("prefers actuals over scheduled windows for the span", () => {
    const shipments = [
      makeShipment("shp_1", "InTransit", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          stops: [
            {
              start: RANGE.start + 10_000,
              end: RANGE.start + 12_000,
              actualArrival: RANGE.start + 8000,
              actualDeparture: RANGE.start + 13_000,
            },
          ],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 1, RANGE, NOW);

    expect(data.rows[0].bars[0].start).toBe(RANGE.start + 8000);
    expect(data.rows[0].bars[0].end).toBe(RANGE.start + 13_000);
  });

  it("drops canceled moves on live shipments but keeps bars for canceled shipments", () => {
    const shipments = [
      makeShipment("shp_1", "InTransit", [
        {
          id: "mov_replanned",
          status: "Canceled",
          workerId: "wkr_a",
          stops: [{ start: RANGE.start + 1000, end: RANGE.start + 2000 }],
        },
        {
          id: "mov_live",
          workerId: "wkr_a",
          stops: [{ start: RANGE.start + 5000, end: RANGE.start + 9000 }],
        },
      ]),
      makeShipment("shp_2", "Canceled", [
        {
          id: "mov_canceled",
          status: "Canceled",
          workerId: "wkr_b",
          stops: [{ start: RANGE.start + 1000, end: RANGE.start + 2000 }],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 2, RANGE, NOW);

    const aliceRow = data.rows.find((r) => r.key === "wkr_a");
    expect(aliceRow?.bars.map((b) => b.moveId)).toEqual(["mov_live"]);

    const canceledRow = data.rows.find((r) => r.key === "wkr_b");
    expect(canceledRow?.bars[0].isCanceled).toBe(true);
    expect(canceledRow?.bars[0].tone).toBe("pending");
  });

  it("skips moves without any usable stop times or outside the window", () => {
    const shipments = [
      makeShipment("shp_1", "New", [
        { id: "mov_unscheduled", workerId: "wkr_a", stops: [{ start: 0 }] },
        {
          id: "mov_out_of_range",
          workerId: "wkr_a",
          stops: [{ start: RANGE.end + 5000, end: RANGE.end + 9000 }],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 1, RANGE, NOW);

    expect(data.rows).toHaveLength(0);
    expect(data.barCount).toBe(0);
  });

  it("flags truncation when the server reports more matches than fetched", () => {
    const data = buildTimelineData([], 400, RANGE, NOW);
    expect(data.truncated).toBe(true);
  });

  it("detects dwell on arrived-but-not-departed stops with severity tiers", () => {
    const arrival = RANGE.start + 1000;
    const shipments = [
      makeShipment("shp_1", "InTransit", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          stops: [{ start: RANGE.start + 1000, end: RANGE.start + 5000, actualArrival: arrival }],
        },
      ]),
    ];

    const belowThreshold = buildTimelineData(
      shipments,
      1,
      RANGE,
      arrival + DWELL_WATCH_SECONDS - 60,
    );
    expect(belowThreshold.rows[0].bars[0].dwell).toBeNull();
    expect(belowThreshold.exceptions.dwelling).toBe(0);

    const watching = buildTimelineData(shipments, 1, RANGE, arrival + DWELL_WATCH_SECONDS + 60);
    expect(watching.rows[0].bars[0].dwell?.severity).toBe("watch");
    expect(watching.rows[0].bars[0].dwell?.locationCode).toBe("LOC0");
    expect(watching.exceptions.dwelling).toBe(1);
    expect(watching.rows[0].alert).toBe("watch");

    const critical = buildTimelineData(shipments, 1, RANGE, arrival + DWELL_CRITICAL_SECONDS + 60);
    expect(critical.rows[0].bars[0].dwell?.severity).toBe("critical");
    expect(critical.rows[0].alert).toBe("late");
  });

  it("does not flag dwell once the stop has departed", () => {
    const arrival = RANGE.start + 1000;
    const shipments = [
      makeShipment("shp_1", "InTransit", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          stops: [
            {
              start: RANGE.start + 1000,
              end: RANGE.start + 5000,
              actualArrival: arrival,
              actualDeparture: arrival + 600,
            },
          ],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 1, RANGE, arrival + DWELL_CRITICAL_SECONDS * 2);
    expect(data.rows[0].bars[0].dwell).toBeNull();
  });

  it("flags overlapping live moves on a driver but never the unassigned lane", () => {
    const shipments = [
      makeShipment("shp_1", "Assigned", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          stops: [{ start: RANGE.start + 1000, end: RANGE.start + 30_000 }],
        },
      ]),
      makeShipment("shp_2", "Assigned", [
        {
          id: "mov_2",
          workerId: "wkr_a",
          stops: [{ start: RANGE.start + 5000, end: RANGE.start + 20_000 }],
        },
      ]),
      makeShipment("shp_3", "New", [
        { id: "mov_3", stops: [{ start: RANGE.start + 1000, end: RANGE.start + 30_000 }] },
      ]),
      makeShipment("shp_4", "New", [
        { id: "mov_4", stops: [{ start: RANGE.start + 5000, end: RANGE.start + 20_000 }] },
      ]),
    ];

    const data = buildTimelineData(shipments, 4, RANGE, NOW);

    expect(data.rows[0].bars.every((bar) => bar.hasOverlap)).toBe(true);
    expect(data.rows[0].stats.overlaps).toBe(2);
    expect(data.unassignedRow?.bars.every((bar) => !bar.hasOverlap)).toBe(true);
    expect(data.exceptions.overlaps).toBe(2);
    expect(data.exceptions.unassigned).toBe(2);
  });

  it("counts late loads and marks the row alert", () => {
    const shipments = [
      makeShipment("shp_1", "Delayed", [
        {
          id: "mov_1",
          workerId: "wkr_a",
          workerName: "Alice Driver",
          stops: [{ start: RANGE.start + 1000, end: RANGE.start + 5000 }],
        },
      ]),
    ];

    const data = buildTimelineData(shipments, 1, RANGE, NOW);

    expect(data.rows[0].bars[0].tone).toBe("late");
    expect(data.rows[0].stats.late).toBe(1);
    expect(data.rows[0].alert).toBe("late");
    expect(data.exceptions.late).toBe(1);
  });
});

describe("sortTimelineRows", () => {
  const shipments = [
    makeShipment("shp_1", "Delayed", [
      {
        id: "mov_1",
        workerId: "wkr_a",
        workerName: "Alice Driver",
        stops: [{ start: RANGE.start + 1000, end: RANGE.start + 5000 }],
      },
    ]),
    makeShipment("shp_2", "Assigned", [
      {
        id: "mov_2",
        workerId: "wkr_b",
        workerName: "Aaron Hauler",
        stops: [{ start: RANGE.start + 1000, end: RANGE.start + 5000 }],
      },
      {
        id: "mov_3",
        workerId: "wkr_b",
        workerName: "Aaron Hauler",
        stops: [{ start: RANGE.start + 40_000, end: RANGE.start + 50_000 }],
      },
    ]),
  ];

  it("keeps the name order untouched", () => {
    const data = buildTimelineData(shipments, 2, RANGE, NOW);
    expect(sortTimelineRows(data.rows, "name")).toBe(data.rows);
  });

  it("ranks drivers with exceptions first", () => {
    const data = buildTimelineData(shipments, 2, RANGE, NOW);
    const sorted = sortTimelineRows(data.rows, "exceptions");
    expect(sorted.map((row) => row.workerName)).toEqual(["Alice Driver", "Aaron Hauler"]);
    expect(data.rows.map((row) => row.workerName)).toEqual(["Aaron Hauler", "Alice Driver"]);
  });

  it("ranks the busiest drivers first", () => {
    const data = buildTimelineData(shipments, 2, RANGE, NOW);
    const sorted = sortTimelineRows(data.rows, "loads");
    expect(sorted.map((row) => row.workerName)).toEqual(["Aaron Hauler", "Alice Driver"]);
  });
});
