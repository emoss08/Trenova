import {
  assignDispatchMovesGraphQL,
  assignDispatchMoveToCarrierGraphQL,
  cancelDispatchCarrierAssignmentGraphQL,
  planDispatchAutoAssignGraphQL,
  unassignDispatchMovesGraphQL,
  type DispatchBulkAssignResult,
  type DispatchPlan,
} from "@/lib/graphql/dispatch-console";
import {
  DISPATCH_BOARD_KEY,
  DISPATCH_DRIVER_MOVES_KEY,
  DISPATCH_LIVE_TENDER_KEY,
  DISPATCH_MOVE_CANDIDATES_KEY,
  DISPATCH_SHIPMENT_TENDERS_KEY,
} from "@/lib/queries/dispatch-console";
import { apiService } from "@/services/api";
import { isOverridableInsuranceWarningError } from "@/services/tender";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type {
  DispatchAssignMoveInput,
  DispatchAssignMoveToCarrierInput,
  DispatchPlanInput,
} from "@trenova/graphql/generated/graphql";
import {
  TENDER_MODE_LABEL,
  type RecordTenderResponsePayload,
  type SpotTenderPayload,
  type WaterfallTenderPayload,
} from "@trenova/shared/types/tender";
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

  const firstFailure = result.results.find((item) => !item.success);
  toast.warning(
    `${verb} ${result.succeeded}, ${result.failed} failed`,
    firstFailure?.error ? { description: firstFailure.error } : undefined,
  );
}

const MAX_TENDER_ERROR_MESSAGES = 3;

/**
 * Tender mutations have no form to distribute field errors into — the dialogs
 * close or the action is a bare button — so the backend's field-level messages
 * ("Carrier has no email address on file", per-line failures) are the toast, not
 * the generic wrapper message.
 */
function describeTenderError(title: string, error: unknown): void {
  const fieldMessages =
    error instanceof ApiRequestError
      ? [...new Set(error.getFieldErrors().map((fieldError) => fieldError.message))]
      : [];
  const description =
    fieldMessages.length > 0
      ? fieldMessages.slice(0, MAX_TENDER_ERROR_MESSAGES).join("\n")
      : error instanceof Error
        ? error.message
        : undefined;
  toast.error(title, description ? { description } : undefined);
}

export type DispatchActions = ReturnType<typeof useDispatchActions>;

export function useDispatchActions() {
  const queryClient = useQueryClient();
  const undoStack = useRef<UndoStep[]>([]);
  const [undoDepth, setUndoDepth] = useState(0);

  const invalidateBoard = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_BOARD_KEY] });
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_MOVE_CANDIDATES_KEY] });
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_DRIVER_MOVES_KEY] });
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
    onSuccess: (result) => {
      describeResult(result, "Assigned");

      // Only successful moves are undoable; queueing a failed one would make undo
      // unassign work that was never assigned here.
      const succeeded = result.results.filter((item) => item.success).map((item) => item.moveId);
      if (succeeded.length > 0) {
        pushUndo({ kind: "assigned", moveIds: succeeded });
      }
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

  // Carrier coverage is deliberately outside the undo stack: reversing it requires a
  // cancellation reason the console cannot invent on the dispatcher's behalf.
  const carrierAssignMutation = useMutation({
    mutationFn: (input: DispatchAssignMoveToCarrierInput) =>
      assignDispatchMoveToCarrierGraphQL(input),
    onSuccess: (assignment) => {
      toast.success("Move assigned to carrier", {
        description: assignment.carrier?.name ?? undefined,
      });
      invalidateBoard();
    },
    onError: (error: Error) =>
      toast.error("Carrier assignment failed", { description: error.message }),
  });

  const carrierCancelMutation = useMutation({
    mutationFn: (params: { moveId: string; reason: string }) =>
      cancelDispatchCarrierAssignmentGraphQL(params.moveId, params.reason),
    onSuccess: () => {
      toast.success("Carrier assignment canceled");
      invalidateBoard();
    },
    onError: (error: Error) =>
      toast.error("Carrier cancellation failed", { description: error.message }),
  });

  // Tendering shares the carrier-coverage rule: none of it is undoable. Reversing a
  // tender means withdrawing live offers with a recorded reason, which the console cannot
  // invent on the dispatcher's behalf, so these never enter the undo stack.
  const invalidateTenders = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_LIVE_TENDER_KEY] });
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_SHIPMENT_TENDERS_KEY] });
    invalidateBoard();
  }, [queryClient, invalidateBoard]);

  const waterfallMutation = useMutation({
    mutationFn: (payload: WaterfallTenderPayload) =>
      apiService.tenderService.createWaterfall(payload),
    onSuccess: (result) => {
      // Screened-out routing guide carriers change what the dispatcher actually
      // got, so the outcome is reported as a warning rather than a plain success.
      const skipped = result.screening?.skipped.length ?? 0;
      if (skipped > 0) {
        toast.warning("Waterfall tender started with skipped carriers", {
          description: `${skipped} routing-guide carrier(s) were skipped for eligibility.`,
        });
      } else {
        toast.success("Waterfall tender started", {
          description: "The rank-1 carrier has been offered the move.",
        });
      }
      invalidateTenders();
    },
    onError: (error: unknown) => describeTenderError("Could not start the waterfall", error),
  });

  const spotTenderMutation = useMutation({
    mutationFn: (payload: SpotTenderPayload) => apiService.tenderService.createSpot(payload),
    onSuccess: (tender) => {
      toast.success(`${TENDER_MODE_LABEL[tender.mode]} tender sent`, {
        description:
          tender.mode === "SpotBroadcast"
            ? "Every carrier on the tender has been offered the move."
            : "The first carrier on the tender has been offered the move.",
      });
      invalidateTenders();
    },
    onError: (error: unknown) => {
      // Overridable insurance warnings render as a confirm-override prompt inside the
      // tender dialog, so a toast here would double-report them.
      if (isOverridableInsuranceWarningError(error)) return;
      describeTenderError("Could not send the spot tender", error);
    },
  });

  const cancelTenderMutation = useMutation({
    mutationFn: (params: { tenderId: string; reason: string }) =>
      apiService.tenderService.cancel(params.tenderId, params.reason),
    onSuccess: () => {
      toast.success("Tender canceled", {
        description: "Outstanding offers have been withdrawn.",
      });
      invalidateTenders();
    },
    onError: (error: unknown) => describeTenderError("Could not cancel the tender", error),
  });

  const recordOfferResponseMutation = useMutation({
    mutationFn: (params: { offerId: string; payload: RecordTenderResponsePayload }) =>
      apiService.tenderService.recordResponse(params.offerId, params.payload),
    onSuccess: (_data, params) => {
      toast.success(
        params.payload.action === "Accept" ? "Acceptance recorded" : "Decline recorded",
      );
      invalidateTenders();
    },
    onError: (error: unknown) => describeTenderError("Could not record the response", error),
  });

  const planMutation = useMutation({
    mutationFn: (input: DispatchPlanInput) => planDispatchAutoAssignGraphQL(input),
    onSuccess: (plan: DispatchPlan) => {
      if (plan.shadowMode) {
        toast.info("Shadow mode: nothing was assigned", {
          description: `${plan.assignments.length} pairing(s) would have been proposed.`,
        });
        return;
      }
      const executed = plan.assignments.filter((item) => item.autoExecutable).length;
      if (executed > 0) {
        invalidateBoard();
      }
      if (plan.assignments.length === 0 && plan.uncovered.length === 0) {
        toast.info("Nothing to plan", {
          description: "No uncovered moves in the current window.",
        });
      }
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
    assignToCarrier: carrierAssignMutation.mutateAsync,
    cancelCarrierAssignment: carrierCancelMutation.mutateAsync,
    startWaterfallTender: waterfallMutation.mutateAsync,
    sendSpotTender: spotTenderMutation.mutateAsync,
    cancelTender: cancelTenderMutation.mutateAsync,
    recordOfferResponse: recordOfferResponseMutation.mutateAsync,
    planAutoAssign: planMutation.mutate,
    plan: planMutation.data ?? null,
    clearPlan: planMutation.reset,
    undo,
    canUndo: undoDepth > 0,
    isAssigning:
      assignMutation.isPending ||
      unassignMutation.isPending ||
      carrierAssignMutation.isPending ||
      carrierCancelMutation.isPending,
    isTendering:
      waterfallMutation.isPending ||
      spotTenderMutation.isPending ||
      cancelTenderMutation.isPending ||
      recordOfferResponseMutation.isPending,
    isPlanning: planMutation.isPending,
  };
}
