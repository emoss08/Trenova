import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import type { AssistantArtifact } from "@/types/assistant";

/** A dispatch_plan artifact as toolartifacts/dispatch_plan_artifact.go stores it. */
export function planArtifact(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "art_1",
    threadId: "thr_1",
    kind: "dispatch_plan",
    status: "Ready",
    title: "Drivers for 3 loads",
    payload: { display: 1, tool: "dispatch_plan", ...payload },
    sourceToolCallId: "call_1",
    pinned: false,
  } as unknown as AssistantArtifact;
}

/** A plan line in the tool's wire form (dispatch_tools.go plannedLineView). */
export function line(moveId: string, proNumber: string, workerId: string, workerName: string, extra: Record<string, unknown> = {}) {
  return {
    shipmentMoveId: moveId,
    proNumber,
    lane: "Omaha, NE to Dayton, OH",
    pickupOpens: "2026-11-25 10:00 EST (tomorrow)",
    pickupAt: 1_795_964_400,
    workerId,
    workerName,
    driveRemainingHours: 9.5,
    cost: "$412",
    confidence: "0.91",
    rationale: `${workerName} costs the plan $412 to cover ${proNumber}`,
    deadheadMiles: 61.4,
    ...extra,
  };
}

/** Three loads: two planned (one with unknown hours and a runner-up), one nobody can take. */
export const threeLoads = {
  planningMode: "Immediate",
  cost: "$1,602",
  lines: [
    line("smv_1", "S-1", "wrk_1", "Adam Stephens", {
      nextBest: [{ workerName: "Cynthia Daniels", extraCost: "$126" }],
    }),
    line("smv_2", "S-2", "wrk_2", "Jose Daniels", {
      driveRemainingHours: undefined,
      hours: "No ELD has reported this driver's hours, so they are unknown, not zero.",
      deadheadMiles: 440,
      cost: "$1,190",
    }),
  ],
  uncovered: [
    {
      shipmentMoveId: "smv_3",
      proNumber: "S-3",
      reason: "No driver could take this move.",
      findings: [{ code: "hos.cycle", severity: "Block", message: "60/70-hour cycle exhausted" }],
    },
  ],
  assignments: [
    { shipmentMoveId: "smv_1", primaryWorkerId: "wrk_1", tractorId: "trc_1", reason: "closest" },
    { shipmentMoveId: "smv_2", primaryWorkerId: "wrk_2", tractorId: "trc_2", trailerId: "trl_2", reason: "only one" },
  ],
  notPlannedCount: 1,
};

/** A move as the board's fragment reads it, open and uncovered unless told otherwise. */
export function boardMove(moveId: string, proNumber: string, extra: Partial<DispatchBoardMove> = {}): DispatchBoardMove {
  return {
    moveId,
    proNumber,
    shipmentId: `shp_${moveId}`,
    moveStatus: "New",
    isCovered: false,
    originCity: "Stockton",
    destinationCity: "Fresno",
    originWindowStart: 1_796_220_000,
    assignedWorkerId: null,
    assignedWorkerName: null,
    assignedCarrierName: null,
    pendingProposal: null,
    ...extra,
  } as unknown as DispatchBoardMove;
}
