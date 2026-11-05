import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import type { DispatchAssignMoveInput } from "@trenova/graphql/generated/graphql";

/**
 * The driver, tractor and trailer a move holds now, as the input that would put them back.
 * Undo of a reassign restores exactly this, so a mis-drop costs one key rather than a
 * search for who had the load. Absent for a move no driver holds.
 */
export function previousAssignmentOf(move: DispatchBoardMove): DispatchAssignMoveInput | null {
  if (!move.assignedWorkerId || !move.assignedTractorId || move.coverageType === "carrier") {
    return null;
  }

  return {
    moveId: move.moveId,
    primaryWorkerId: move.assignedWorkerId,
    tractorId: move.assignedTractorId,
    trailerId: move.assignedTrailerId ?? undefined,
    reassign: true,
  };
}
