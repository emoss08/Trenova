import type { Shipment } from "@/types/shipment";
import { describe, expect, it } from "vitest";
import { buildTimelineData, UNASSIGNED_ROW_KEY } from "./use-timeline-data";

const RANGE = { start: 1_000_000, end: 1_086_400 }; // one-day window

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
              wholeName: move.workerName ?? move.workerId,
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

    const data = buildTimelineData(shipments, 2, RANGE);

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

    const data = buildTimelineData(shipments, 2, RANGE);

    expect(data.rows).toHaveLength(1);
    expect(data.rows[0].laneCount).toBe(2);
    expect(new Set(data.rows[0].bars.map((b) => b.laneIndex))).toEqual(new Set([0, 1]));
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

    const data = buildTimelineData(shipments, 1, RANGE);

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

    const data = buildTimelineData(shipments, 2, RANGE);

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

    const data = buildTimelineData(shipments, 1, RANGE);

    expect(data.rows).toHaveLength(0);
    expect(data.barCount).toBe(0);
  });

  it("flags truncation when the server reports more matches than fetched", () => {
    const data = buildTimelineData([], 400, RANGE);
    expect(data.truncated).toBe(true);
  });
});
