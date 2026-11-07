import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { describe, expect, it } from "vitest";
import { previousAssignmentOf } from "../assignment-input";

function move(overrides: Partial<DispatchBoardMove>): DispatchBoardMove {
  return {
    moveId: "sm_1",
    coverageType: "driver",
    assignedWorkerId: null,
    assignedTractorId: null,
    assignedTrailerId: null,
    ...overrides,
  } as DispatchBoardMove;
}

describe("previousAssignmentOf", () => {
  it("is the driver, tractor and trailer the move holds, as a reassign", () => {
    expect(
      previousAssignmentOf(
        move({ assignedWorkerId: "wrk_1", assignedTractorId: "trc_1", assignedTrailerId: "trl_1" }),
      ),
    ).toEqual({
      moveId: "sm_1",
      primaryWorkerId: "wrk_1",
      tractorId: "trc_1",
      trailerId: "trl_1",
      reassign: true,
    });
  });

  it("leaves the trailer out when the move holds none", () => {
    expect(
      previousAssignmentOf(move({ assignedWorkerId: "wrk_1", assignedTractorId: "trc_1" })),
    ).toMatchObject({ trailerId: undefined });
  });

  it("is absent for an uncovered move, so undo unassigns instead", () => {
    expect(previousAssignmentOf(move({}))).toBeNull();
  });

  it("is absent for a carrier-covered move, which no driver held", () => {
    expect(
      previousAssignmentOf(
        move({ coverageType: "carrier", assignedWorkerId: "wrk_1", assignedTractorId: "trc_1" }),
      ),
    ).toBeNull();
  });
});
