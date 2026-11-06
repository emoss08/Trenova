import { createTimelineScale } from "@/components/timeline/timeline-scale";
import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCommitment,
} from "@/lib/graphql/dispatch-console";
import { describe, expect, it } from "vitest";
import { buildConsoleLanes, revealKeyFor } from "../dispatch-timeline";

const TUE = Date.UTC(2026, 10, 3) / 1000;
const HOUR = 3_600;
const at = (hours: number) => TUE + hours * HOUR;
const scale = createTimelineScale([TUE, at(24)], { pxPerHour: 64, tickHours: 2, hourLabels: true });

// Written from DispatchBoardMove in dispatch_console.graphqls.
function move(id: string, from: number, to: number, overrides: Partial<DispatchBoardMove> = {}) {
  return {
    moveId: id,
    shipmentId: `shp_${id}`,
    proNumber: `PRO-${id}`,
    urgency: "Today",
    isCovered: false,
    coverageType: "unassigned",
    customerName: "Acme",
    originCity: "Dallas",
    originState: "TX",
    originWindowStart: at(from),
    originWindowEnd: null,
    destinationCity: "Waco",
    destinationState: "TX",
    destinationWindowStart: at(to),
    destinationWindowEnd: null,
    ...overrides,
  } as DispatchBoardMove;
}

// Written from DispatchCommitment in dispatch_console.graphqls.
function commitment(id: string, from: number, to: number): DispatchCommitment {
  return {
    moveId: id,
    shipmentId: `shp_${id}`,
    proNumber: `PRO-${id}`,
    moveStatus: "Assigned",
    windowStart: at(from),
    windowEnd: at(to),
    destinationCity: "Austin",
    destinationState: "TX",
    destinationLatitude: null,
    destinationLongitude: null,
    trailerId: null,
  };
}

// Written from DispatchBoardDriver in dispatch_console.graphqls.
function driver(id: string, overrides: Partial<DispatchBoardDriver> = {}): DispatchBoardDriver {
  return {
    workerId: id,
    firstName: id,
    lastName: "Driver",
    tractorId: `trc_${id}`,
    tractorCode: `T-${id}`,
    availability: "Open",
    projectedTimeAvailable: 0,
    commitments: [],
    timeOff: [],
    findings: [],
    ...overrides,
  } as DispatchBoardDriver;
}

describe("buildConsoleLanes", () => {
  it("pools only uncovered moves that touch the window", () => {
    const lanes = buildConsoleLanes(
      [
        move("open", 8, 12),
        move("covered", 9, 13, { isCovered: true }),
        move("tomorrow", 30, 34),
        move("no-appointment", 0, 0, { originWindowStart: 0 }),
      ],
      [],
      scale,
    );

    expect(lanes.moves.map((span) => span.id)).toEqual(["open"]);
  });

  it("groups moves as the queue does, the most urgent group first and the earliest pickup first", () => {
    const lanes = buildConsoleLanes(
      [
        move("later", 18, 20, { urgency: "Planned" }),
        move("today-late", 14, 16),
        move("late", 2, 9, { urgency: "Late" }),
        move("today-early", 9, 12),
      ],
      [],
      scale,
    );

    expect(lanes.groups.map((group) => [group.bucket, group.moves.map((span) => span.id)])).toEqual(
      [
        ["Late", ["late"]],
        ["Today", ["today-early", "today-late"]],
        ["Planned", ["later"]],
      ],
    );
  });

  it("draws a move with one appointment at least half an hour wide", () => {
    const lanes = buildConsoleLanes(
      [move("short", 10, 0, { destinationWindowStart: 0 })],
      [],
      scale,
    );

    expect(lanes.moves[0].end - lanes.moves[0].start).toBe(1_800);
  });

  it("orders drivers as the Drivers tab does: open first, then the soonest free", () => {
    const lanes = buildConsoleLanes(
      [],
      [
        driver("blocked", { availability: "Blocked" }),
        driver("late", { availability: "Working", projectedTimeAvailable: at(15) }),
        driver("soon", { availability: "Working", projectedTimeAvailable: at(11) }),
        driver("open"),
      ],
      scale,
    );

    expect(lanes.drivers.map((lane) => lane.driver.workerId)).toEqual([
      "open",
      "soon",
      "late",
      "blocked",
    ]);
  });

  it("marks a driver's overlapping work as double-booked and keeps only time off in the window", () => {
    const lanes = buildConsoleLanes(
      [],
      [
        driver("busy", {
          commitments: [commitment("a", 8, 14), commitment("b", 12, 18)],
          timeOff: [
            { startDate: at(-48), endDate: at(-24), type: "Vacation" },
            { startDate: at(20), endDate: at(30), type: "Sick" },
          ],
        }),
      ],
      scale,
    );

    const [lane] = lanes.drivers;
    expect(lane.packed.rows).toBe(2);
    expect([...lane.packed.conflicts].sort()).toEqual(["a", "b"]);
    expect(lane.timeOff.map((pto) => pto.type)).toEqual(["Sick"]);
  });
});

describe("revealKeyFor", () => {
  const lanes = buildConsoleLanes(
    [move("open", 8, 12)],
    [driver("kim", { commitments: [commitment("held", 9, 15)] })],
    scale,
  );

  it("reveals the row of a move still needing a driver", () => {
    expect(revealKeyFor(lanes, "open", null)).toBe("move:open");
  });

  it("reveals the lane of the driver who holds the selected move", () => {
    expect(revealKeyFor(lanes, "held", null)).toBe("driver:kim");
  });

  it("reveals a selected driver's lane, and nothing for a move off the sheet", () => {
    expect(revealKeyFor(lanes, null, "kim")).toBe("driver:kim");
    expect(revealKeyFor(lanes, "elsewhere", null)).toBeNull();
  });
});
