import {
  assignDispatchMovesGraphQL,
  planDispatchAutoAssignGraphQL,
  unassignDispatchMovesGraphQL,
  type DispatchBulkAssignResult,
  type DispatchPlan,
} from "@/lib/graphql/dispatch-console";
import type { DispatchAssignMoveInput } from "@trenova/graphql/generated/graphql";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";

/**
 * An undoable step. Assignment is reversible in the domain, so the console treats
 * reversing it as a first-class action rather than something a dispatcher has to go
 * hunting for after a mis-drop.
 */
type UndoStep =
  | { kind: "assigned"; moveIds: string[] }
  | { kind: "unassigned"; restore: DispatchAssignMoveInput[] };

const MAX_UNDO_DEPTH = 25;

function describeResult(result: DispatchBulkAssignResult, verb: string): void {
  if (result.failed === 0) {
    toast.success(`${verb} ${result.succeeded} move${result.succeeded === 1 ? "" : "s"}`);
    return;
  }

  // Partial success is the normal case for a bulk action, so it reports what actually
  // happened rather than collapsing to a single failure toast.
  const firstFailure = result.results.find((item) => !item.success);
  toast.warning(
    `${verb} ${result.succeeded}, ${result.failed} failed`,
    firstFailure?.error ? { description: firstFailure.error } : undefined,
  );
}

export function useDispatchActions() {
  const queryClient = useQueryClient();
  const undoStack = useRef<UndoStep[]>([]);
  const [undoDepth, setUndoDepth] = useState(0);

  const invalidateBoard = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["dispatch-board"] });
    void queryClient.invalidateQueries({ queryKey: ["dispatch-move-candidates"] });
    void queryClient.invalidateQueries({ queryKey: ["dispatch-driver-moves"] });
  }, [queryClient]);

  const pushUndo = useCallback((step: UndoStep) => {
    undoStack.current.push(step);
    if (undoStack.current.length > MAX_UNDO_DEPTH) {
      undoStack.current.shift();
    }
    setUndoDepth(undoStack.current.length);
  }, []);

  const assignMutation = useMutation({
    mutationFn: (input: DispatchAssignMoveInput[]) => assignDispatchMovesGraphQL(input),
    onSuccess: (result, input) => {
      describeResult(result, "Assigned");

      // Only successful moves are undoable; queueing a failed one would make undo
      // unassign work that was never assigned here.
      const succeeded = result.results.filter((item) => item.success).map((item) => item.moveId);
      if (succeeded.length > 0) {
        pushUndo({ kind: "assigned", moveIds: succeeded });
      }
      void input;
      invalidateBoard();
    },
    onError: (error: Error) => toast.error("Assignment failed", { description: error.message }),
  });

  const unassignMutation = useMutation({
    mutationFn: (params: { moveIds: string[]; restore?: DispatchAssignMoveInput[] }) =>
      unassignDispatchMovesGraphQL(params.moveIds),
    onSuccess: (result, params) => {
      describeResult(result, "Unassigned");

      if (params.restore && params.restore.length > 0) {
        const succeeded = new Set(
          result.results.filter((item) => item.success).map((item) => item.moveId),
        );
        const restorable = params.restore.filter((item) => succeeded.has(String(item.moveId)));
        if (restorable.length > 0) {
          pushUndo({ kind: "unassigned", restore: restorable });
        }
      }
      invalidateBoard();
    },
    onError: (error: Error) => toast.error("Unassignment failed", { description: error.message }),
  });

  const planMutation = useMutation({
    mutationFn: (input: { windowStart?: number; windowEnd?: number; apply?: boolean }) =>
      planDispatchAutoAssignGraphQL(input),
    onSuccess: (plan: DispatchPlan) => {
      if (plan.shadowMode) {
        toast.info("Shadow mode: nothing was assigned", {
          description: `${plan.assignments.length} pairing(s) would have been proposed.`,
        });
        return;
      }
      const executed = plan.assignments.filter((item) => item.autoExecutable).length;
      toast.success(`Planned coverage for ${plan.assignments.length} move(s)`, {
        description:
          executed > 0
            ? `${executed} assigned automatically, ${plan.assignments.length - executed} awaiting review.`
            : `All pairings await your review. ${plan.uncovered.length} move(s) could not be covered.`,
      });
      invalidateBoard();
    },
    onError: (error: Error) => toast.error("Auto-assign failed", { description: error.message }),
  });

  // The mutation objects themselves are not referentially stable; their mutate functions
  // are, so undo depends on those.
  const runAssign = assignMutation.mutate;
  const runUnassign = unassignMutation.mutate;

  const undo = useCallback(() => {
    const step = undoStack.current.pop();
    setUndoDepth(undoStack.current.length);
    if (!step) return;

    if (step.kind === "assigned") {
      // Reversing an assignment is an unassignment with no restore payload, so undoing an
      // undo is not itself queued — that would loop.
      runUnassign({ moveIds: step.moveIds });
      return;
    }

    runAssign(step.restore);
  }, [runAssign, runUnassign]);

  return {
    assign: runAssign,
    unassign: (moveIds: string[], restore?: DispatchAssignMoveInput[]) =>
      runUnassign({ moveIds, restore }),
    planAutoAssign: planMutation.mutate,
    plan: planMutation.data,
    undo,
    canUndo: undoDepth > 0,
    isAssigning: assignMutation.isPending || unassignMutation.isPending,
    isPlanning: planMutation.isPending,
  };
}
