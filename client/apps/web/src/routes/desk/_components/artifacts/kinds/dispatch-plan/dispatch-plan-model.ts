import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import type {
  DispatchPlanArtifact,
  DispatchPlanAssignment,
  DispatchPlanLine,
} from "../../artifact-payloads";

/**
 * Where a planned load stands now, read from the move itself rather than the plan:
 * - `planned`: nobody is on it and nothing is proposed;
 * - `proposed`: an agent proposed the planned driver and a person has yet to decide;
 * - `assigned`: the planned driver is on it;
 * - `taken`: someone else covers it, another driver or a carrier;
 * - `closed`: it was delivered or canceled;
 * - `unread`: the move could not be read back (gone, or out of the reader's reach).
 */
export type LoadState = "planned" | "proposed" | "assigned" | "taken" | "closed" | "unread";

const CLOSED_MOVE_STATUSES: ReadonlySet<string> = new Set(["Completed", "Canceled"]);

export function loadStateOf(line: DispatchPlanLine, move: DispatchBoardMove | undefined): LoadState {
  if (move === undefined) {
    return "unread";
  }
  if (CLOSED_MOVE_STATUSES.has(move.moveStatus)) {
    return "closed";
  }
  if (move.assignedWorkerId && move.assignedWorkerId === line.workerId) {
    return "assigned";
  }
  if (move.isCovered) {
    return "taken";
  }
  if (move.pendingProposal && move.pendingProposal.workerId === line.workerId) {
    return "proposed";
  }

  return "planned";
}

/** Who covers a load now when it is not the planned driver: a driver's name or a carrier's. */
export function takenBy(move: DispatchBoardMove | undefined): string {
  return move?.assignedWorkerName || move?.assignedCarrierName || "";
}

/** The moves the card reads back: every planned load and every gap, each once. */
export function planMoveIds(plan: DispatchPlanArtifact): string[] {
  return [...new Set([...plan.lines.map((line) => line.moveId), ...plan.gaps.map((gap) => gap.moveId)])];
}

/**
 * The assignments a person can still make from the plan: a line with its assignment row whose
 * load nobody covers yet and is still open. A load whose plan was proposed can be assigned
 * here too; the proposal then has nothing left to do.
 */
export function assignableRows(
  plan: DispatchPlanArtifact,
  states: ReadonlyMap<string, LoadState>,
): DispatchPlanAssignment[] {
  return plan.assignments.filter((row) => {
    const state = states.get(row.moveId);
    return state === "planned" || state === "proposed";
  });
}

export type PlanStanding = "empty" | "assigned" | "proposed" | "partial" | "plan";

/** Where the plan as a whole stands, for its status pill. */
export function planStanding(
  plan: DispatchPlanArtifact,
  states: ReadonlyMap<string, LoadState>,
): PlanStanding {
  if (plan.lines.length === 0) {
    return "empty";
  }
  const lineStates = plan.lines.map((line) => states.get(line.moveId) ?? "unread");
  if (lineStates.every((state) => state === "assigned")) {
    return "assigned";
  }
  if (lineStates.some((state) => state === "proposed")) {
    return "proposed";
  }
  if (lineStates.some((state) => state === "assigned")) {
    return "partial";
  }

  return "plan";
}

/** Whether a planned line asks for a look before it is assigned: a warning, or unknown hours. */
export function needsCheck(line: DispatchPlanLine): boolean {
  return (
    line.hoursNote !== "" ||
    line.notSendable !== "" ||
    line.findings.some((finding) => finding.severity === "Warn" || finding.severity === "Block")
  );
}

export type PlanTally = {
  loads: number;
  covered: number;
  gaps: number;
  emptyMiles: number | null;
  toCheck: number;
};

export function planTally(plan: DispatchPlanArtifact): PlanTally {
  const miles = plan.lines
    .map((line) => line.deadheadMiles)
    .filter((value): value is number => value !== null);

  return {
    loads: plan.lines.length + plan.gaps.length,
    covered: plan.lines.length,
    gaps: plan.gaps.length,
    emptyMiles: miles.length === 0 ? null : Math.round(miles.reduce((sum, value) => sum + value, 0)),
    toCheck: plan.lines.filter(needsCheck).length,
  };
}
