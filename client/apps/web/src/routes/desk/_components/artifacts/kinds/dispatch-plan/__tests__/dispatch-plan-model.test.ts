import { describe, expect, it } from "vitest";
import { dispatchPlanFrom } from "../../../artifact-payloads";
import {
  assignableRows,
  loadStateOf,
  needsCheck,
  planMoveIds,
  planStanding,
  planTally,
  takenBy,
} from "../dispatch-plan-model";
import { boardMove, line, planArtifact, threeLoads } from "./fixtures";

const plan = dispatchPlanFrom(planArtifact(threeLoads));

describe("dispatchPlanFrom", () => {
  it("reads the tool's lines, gaps and assignment rows as written", () => {
    expect(plan.cost).toBe("$1,602");
    expect(plan.lines.map((entry) => [entry.moveId, entry.workerName, entry.driveHours])).toEqual([
      ["smv_1", "Adam Stephens", 9.5],
      ["smv_2", "Jose Daniels", null],
    ]);
    expect(plan.lines[0]?.confidence).toBe(0.91);
    expect(plan.lines[0]?.nextBest).toEqual([{ workerName: "Cynthia Daniels", extraCost: "$126" }]);
    expect(plan.gaps).toEqual([
      {
        moveId: "smv_3",
        proNumber: "S-3",
        reason: "No driver could take this move.",
        findings: [{ code: "hos.cycle", severity: "Block", message: "60/70-hour cycle exhausted" }],
      },
    ]);
    expect(plan.assignments).toEqual([
      { moveId: "smv_1", workerId: "wrk_1", tractorId: "trc_1", trailerId: null },
      { moveId: "smv_2", workerId: "wrk_2", tractorId: "trc_2", trailerId: "trl_2" },
    ]);
    expect(plan.notPlanned).toBe(1);
  });

  it("drops a line without a driver and an assignment without a tractor, and reads an empty payload as an empty plan", () => {
    const partial = dispatchPlanFrom(
      planArtifact({
        lines: [{ shipmentMoveId: "smv_9", proNumber: "S-9" }],
        assignments: [{ shipmentMoveId: "smv_9", primaryWorkerId: "wrk_9" }],
      }),
    );
    expect(partial.lines).toEqual([]);
    expect(partial.assignments).toEqual([]);

    const empty = dispatchPlanFrom(planArtifact({}));
    expect(empty).toEqual({
      planningMode: "",
      cost: "",
      lines: [],
      gaps: [],
      assignments: [],
      notPlanned: 0,
    });
  });
});

describe("loadStateOf", () => {
  const planned = plan.lines[0]!;

  it("reads each load from the move as it stands now", () => {
    expect(loadStateOf(planned, undefined)).toBe("unread");
    expect(loadStateOf(planned, boardMove("smv_1", "S-1"))).toBe("planned");
    expect(
      loadStateOf(
        planned,
        boardMove("smv_1", "S-1", {
          pendingProposal: { workerId: "wrk_1" } as never,
        }),
      ),
    ).toBe("proposed");
    expect(
      loadStateOf(
        planned,
        boardMove("smv_1", "S-1", { isCovered: true, assignedWorkerId: "wrk_1", moveStatus: "Assigned" } as never),
      ),
    ).toBe("assigned");
    expect(
      loadStateOf(
        planned,
        boardMove("smv_1", "S-1", { isCovered: true, assignedWorkerId: "wrk_7", assignedWorkerName: "Lee Park" } as never),
      ),
    ).toBe("taken");
    expect(loadStateOf(planned, boardMove("smv_1", "S-1", { moveStatus: "Canceled" } as never))).toBe("closed");
  });

  it("counts a proposal for another driver as no proposal for the planned one", () => {
    expect(
      loadStateOf(planned, boardMove("smv_1", "S-1", { pendingProposal: { workerId: "wrk_9" } as never })),
    ).toBe("planned");
  });

  it("names whoever covers a load now, a driver before a carrier", () => {
    expect(takenBy(boardMove("smv_1", "S-1", { assignedCarrierName: "Swift" } as never))).toBe("Swift");
    expect(takenBy(undefined)).toBe("");
  });
});

describe("what the card offers and says", () => {
  it("reads every planned load and gap back once", () => {
    expect(planMoveIds(plan)).toEqual(["smv_1", "smv_2", "smv_3"]);
  });

  it("offers only the open, uncovered loads to assign, a proposed one included", () => {
    const states = new Map([
      ["smv_1", "proposed" as const],
      ["smv_2", "assigned" as const],
    ]);
    expect(assignableRows(plan, states).map((row) => row.moveId)).toEqual(["smv_1"]);
  });

  it("stands as planned, proposed, partly assigned, assigned or empty", () => {
    expect(planStanding(plan, new Map([["smv_1", "planned"], ["smv_2", "planned"]]))).toBe("plan");
    expect(planStanding(plan, new Map([["smv_1", "proposed"], ["smv_2", "planned"]]))).toBe("proposed");
    expect(planStanding(plan, new Map([["smv_1", "assigned"], ["smv_2", "planned"]]))).toBe("partial");
    expect(planStanding(plan, new Map([["smv_1", "assigned"], ["smv_2", "assigned"]]))).toBe("assigned");
    expect(planStanding(dispatchPlanFrom(planArtifact({ uncovered: threeLoads.uncovered })), new Map())).toBe("empty");
  });

  it("tallies covered loads, gaps, empty miles and what to check", () => {
    expect(planTally(plan)).toEqual({ loads: 3, covered: 2, gaps: 1, emptyMiles: 501, toCheck: 1 });
  });

  it("asks for a look at unknown hours, a warning or a line that cannot be sent", () => {
    const clean = dispatchPlanFrom(planArtifact({ lines: [line("smv_1", "S-1", "wrk_1", "A B")] })).lines[0]!;
    expect(needsCheck(clean)).toBe(false);
    expect(needsCheck({ ...clean, notSendable: "no tractor" })).toBe(true);
    expect(needsCheck({ ...clean, findings: [{ code: "x", severity: "Warn", message: "MVR overdue" }] })).toBe(true);
  });
});
