import { useT } from "@trenova/shared/i18n/use-t";
import {
  assignDispatchMovesGraphQL,
  assignDispatchMoveToCarrierGraphQL,
  cancelDispatchCarrierAssignmentGraphQL,
  planDispatchAutoAssignGraphQL,
  unassignDispatchMovesGraphQL,
  type DispatchBulkAssignResult,
  type DispatchPlan,
} from "@/lib/graphql/dispatch-console";
import { decideAgentProposal, undoMyDecision } from "@/lib/graphql/agent-decisions";
import { agentPreview } from "@/lib/queries/agent-preview";
import { isPreviewConflict } from "@/components/assistant/proposal-preview/preview-gate";
import {
  DISPATCH_BOARD_KEY,
  DISPATCH_DRIVER_MOVES_KEY,
  DISPATCH_LIVE_TENDER_KEY,
  DISPATCH_MOVE_CANDIDATES_KEY,
  DISPATCH_SHIPMENT_TENDERS_KEY,
} from "@/lib/queries/dispatch-console";
import { tenderService, isOverridableInsuranceWarningError } from "@/services/tender";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type {
  DispatchAssignMoveInput,
  DispatchAssignMoveToCarrierInput,
  DispatchPlanInput,
} from "@trenova/graphql/generated/graphql";
import type {
  BounceTenderPayload,
  RecordTenderResponsePayloadInput,
  SpotTenderPayload,
  WaterfallTenderPayload,
} from "@trenova/shared/types/tender";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { translate } from "@trenova/shared/i18n/runtime";

/**
 * An undoable step, carrying the words that say what undoing it reverses. Assignment is
 * reversible in the domain, so the console treats reversing it as a first-class action
 * rather than something a dispatcher has to go hunting for after a mis-drop.
 *
 * - `assigned`: a move that had no driver; undo leaves it uncovered again.
 * - `restore`: a move whose previous driver is known (a reassign, an unassign); undo puts
 *   that driver back.
 * - `approved`: a proposal approved here. While it waits out its undo window undo takes
 *   the approval back; once it has run, undo leaves the move uncovered again.
 */
type UndoStep = { label: string; undone: string } & (
  | { kind: "assigned"; moveIds: string[] }
  | { kind: "restore"; restore: DispatchAssignMoveInput[] }
  | { kind: "approved"; proposalId: string; moveId: string; executed: boolean }
);

const MAX_UNDO_DEPTH = 25;

const APPROVE_REASON = "approved_from_dispatch_console";
const DECLINE_REASON = "declined_from_dispatch_console";

/** What the last write here came to, said in place with its Undo. */
export type DispatchOutcome = {
  id: number;
  tone: "ok" | "warn";
  text: string;
  undoable: boolean;
};

/** One move the dispatcher is assigning, and what it held before, so undo can restore it. */
export type AssignRequest = {
  input: DispatchAssignMoveInput;
  /** Names the pairing in the outcome: "1048813 assigned to Dana Ortiz". */
  proNumber: string;
  driverName: string;
  /** The driver, tractor and trailer the move held, for a reassign; absent when uncovered. */
  previous?: DispatchAssignMoveInput | null;
};

export type ProposalDecision = {
  proposalId: string;
  moveId: string;
  proNumber: string;
  driverName: string;
  /** The digest of the preview on screen; absent when it could not be read. */
  previewDigest?: string;
};

/** A bulk write that partly failed says so; one that went through is said in the outcome line. */
function describeResult(result: DispatchBulkAssignResult, action: "assigned" | "unassigned"): void {
  if (result.failed === 0) {
    return;
  }

  const firstFailure = result.results.find((item) => !item.success);
  toast.warning(
    action === "assigned"
      ? translate("Assigned {0}, {1} failed", result.succeeded, result.failed)
      : translate("Unassigned {0}, {1} failed", result.succeeded, result.failed),
    firstFailure?.error ? { description: firstFailure.error } : undefined,
  );
}

function undoLabel(requests: AssignRequest[]): string {
  if (requests.length === 1) {
    const [request] = requests;
    return request.previous
      ? translate("Put the previous driver back on {0}", request.proNumber)
      : translate("Unassign {0}", request.proNumber);
  }
  return translate(
    "{0, plural, one {Undo # assignment} other {Undo # assignments}}",
    requests.length,
  );
}

function undoneLabel(requests: AssignRequest[]): string {
  if (requests.length === 1) {
    const [request] = requests;
    return request.previous
      ? translate("The previous driver is back on {0}", request.proNumber)
      : translate("{0} is uncovered again", request.proNumber);
  }
  return translate(
    "{0, plural, one {# assignment undone} other {# assignments undone}}",
    requests.length,
  );
}

export type RecordOfferResponseParams = {
  offerId: string;
  payload: RecordTenderResponsePayloadInput;
};

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
  const t = useT();

  const queryClient = useQueryClient();
  const undoStack = useRef<UndoStep[]>([]);
  const [undoTop, setUndoTop] = useState<{ depth: number; label: string | null }>({
    depth: 0,
    label: null,
  });
  const [outcome, setOutcome] = useState<DispatchOutcome | null>(null);
  // Two keypresses can land before the button re-renders disabled; the ref refuses the
  // second write while the first is still in flight.
  const writing = useRef(false);

  const invalidateBoard = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_BOARD_KEY] });
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_MOVE_CANDIDATES_KEY] });
    void queryClient.invalidateQueries({ queryKey: [DISPATCH_DRIVER_MOVES_KEY] });
  }, [queryClient]);

  const say = useCallback((tone: DispatchOutcome["tone"], text: string, undoable: boolean) => {
    setOutcome({ id: Date.now(), tone, text, undoable });
  }, []);

  const syncUndo = useCallback(() => {
    const stack = undoStack.current;
    setUndoTop({ depth: stack.length, label: stack.at(-1)?.label ?? null });
  }, []);

  const pushUndo = useCallback(
    (step: UndoStep) => {
      undoStack.current.push(step);
      if (undoStack.current.length > MAX_UNDO_DEPTH) {
        undoStack.current.shift();
      }
      syncUndo();
    },
    [syncUndo],
  );

  const assignMutation = useMutation({
    mutationFn: (params: { requests: AssignRequest[]; isUndo: boolean }) =>
      assignDispatchMovesGraphQL(params.requests.map((request) => request.input)),
    onSuccess: (result, { requests, isUndo }) => {
      invalidateBoard();
      // Undo says what came of it in the outcome line, so it raises no toast of its own.
      if (isUndo) return;
      describeResult(result, "assigned");

      // Only moves that went through are undoable; queueing a failed one would make undo
      // change work that was never changed here.
      const succeeded = new Set(
        result.results.filter((item) => item.success).map((item) => item.moveId),
      );
      const done = requests.filter((request) => succeeded.has(String(request.input.moveId)));
      const fresh = done.filter((request) => !request.previous);
      const replaced = done.filter((request) => request.previous);

      if (fresh.length > 0) {
        pushUndo({
          kind: "assigned",
          moveIds: fresh.map((request) => String(request.input.moveId)),
          label: undoLabel(fresh),
          undone: undoneLabel(fresh),
        });
      }
      if (replaced.length > 0) {
        pushUndo({
          kind: "restore",
          restore: replaced.map((request) => ({ ...request.previous!, reassign: true })),
          label: undoLabel(replaced),
          undone: undoneLabel(replaced),
        });
      }
      if (done.length === 1) {
        const [request] = done;
        say(
          "ok",
          request.previous
            ? t("{0} reassigned to {1}", request.proNumber, request.driverName)
            : t("{0} assigned to {1}", request.proNumber, request.driverName),
          true,
        );
      } else if (done.length > 1) {
        say(
          "ok",
          t("{0, plural, one {# move assigned} other {# moves assigned}}", done.length),
          true,
        );
      }
    },
    onError: (error: Error) => toast.error(t("Assignment failed"), { description: error.message }),
  });

  const unassignMutation = useMutation({
    mutationFn: (params: {
      moveIds: string[];
      restore?: DispatchAssignMoveInput[];
      isUndo?: boolean;
    }) => unassignDispatchMovesGraphQL(params.moveIds),
    onSuccess: (result, params) => {
      if (!params.isUndo) describeResult(result, "unassigned");

      if (params.restore && params.restore.length > 0) {
        const succeeded = new Set(
          result.results.filter((item) => item.success).map((item) => item.moveId),
        );
        const restorable = params.restore.filter((item) => succeeded.has(String(item.moveId)));
        if (restorable.length > 0) {
          pushUndo({
            kind: "restore",
            restore: restorable,
            label: t(
              "{0, plural, one {Put the driver back on # move} other {Put the drivers back on # moves}}",
              restorable.length,
            ),
            undone: t(
              "{0, plural, one {The driver is back on # move} other {The drivers are back on # moves}}",
              restorable.length,
            ),
          });
        }
      }
      invalidateBoard();
    },
    onError: (error: Error) =>
      toast.error(t("Unassignment failed"), { description: error.message }),
  });

  const invalidateDecisions = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: agentPreview._def });
    invalidateBoard();
  }, [invalidateBoard, queryClient]);

  const approveMutation = useMutation({
    mutationFn: (decision: ProposalDecision) =>
      decideAgentProposal(decision.proposalId, {
        decision: "Accepted",
        reasonCode: APPROVE_REASON,
        previewDigest: decision.previewDigest,
      }),
    onSuccess: (result, decision) => {
      if (result.error) {
        say("warn", t("{0} was not assigned: {1}", decision.proNumber, result.error), false);
        invalidateDecisions();
        return;
      }
      pushUndo({
        kind: "approved",
        proposalId: decision.proposalId,
        moveId: decision.moveId,
        executed: result.executed,
        label: result.executed
          ? t("Unassign {0}", decision.proNumber)
          : t("Take back the approval for {0}", decision.proNumber),
        undone: result.executed
          ? t("{0} is uncovered again", decision.proNumber)
          : t("Approval for {0} taken back; the proposal waits on you again", decision.proNumber),
      });
      say(
        "ok",
        result.executed
          ? t("{0} assigned to {1}", decision.proNumber, decision.driverName)
          : t(
              "Approved {1} for {0}; it goes through when the undo window ends",
              decision.proNumber,
              decision.driverName,
            ),
        true,
      );
      invalidateDecisions();
    },
    onError: (error: unknown, decision) => {
      if (isPreviewConflict(error)) {
        say(
          "warn",
          t("{0} changed since Nova proposed this. Check the proposal again.", decision.proNumber),
          false,
        );
        invalidateDecisions();
        return;
      }
      toast.error(t("Approval failed"), {
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const declineMutation = useMutation({
    mutationFn: (params: ProposalDecision & { note: string }) =>
      decideAgentProposal(params.proposalId, {
        decision: "Rejected",
        reasonCode: DECLINE_REASON,
        note: params.note,
      }),
    onSuccess: (result, params) => {
      if (result.error) {
        say(
          "warn",
          t("Couldn't decline the proposal for {0}: {1}", params.proNumber, result.error),
          false,
        );
      } else {
        say("ok", t("Declined Nova's driver for {0}", params.proNumber), false);
      }
      invalidateDecisions();
    },
    onError: (error: unknown) =>
      toast.error(t("Decline failed"), {
        description: error instanceof Error ? error.message : undefined,
      }),
  });

  const undoApprovalMutation = useMutation({
    mutationFn: (proposalId: string) => undoMyDecision({ proposalId }),
    onSuccess: () => {
      invalidateDecisions();
    },
    onError: (error: unknown) =>
      toast.error(t("Couldn't take the approval back"), {
        description: error instanceof Error ? error.message : undefined,
      }),
  });

  // Carrier coverage is deliberately outside the undo stack: reversing it requires a
  // cancellation reason the console cannot invent on the dispatcher's behalf.
  const carrierAssignMutation = useMutation({
    mutationFn: (input: DispatchAssignMoveToCarrierInput) =>
      assignDispatchMoveToCarrierGraphQL(input),
    onSuccess: () => {
      invalidateBoard();
    },
    onError: (error: Error) =>
      toast.error(t("Carrier assignment failed"), { description: error.message }),
  });

  const carrierCancelMutation = useMutation({
    mutationFn: (params: { moveId: string; reason: string }) =>
      cancelDispatchCarrierAssignmentGraphQL(params.moveId, params.reason),
    onSuccess: () => {
      invalidateBoard();
    },
    onError: (error: Error) =>
      toast.error(t("Carrier cancellation failed"), { description: error.message }),
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
    mutationFn: (payload: WaterfallTenderPayload) => tenderService.createWaterfall(payload),
    onSuccess: (result) => {
      // Screened-out routing guide carriers change what the dispatcher actually
      // got, so the outcome is reported as a warning rather than a plain success.
      const skipped = result.screening?.skipped.length ?? 0;
      if (skipped > 0) {
        toast.warning(t("Waterfall tender started with skipped carriers"), {
          description: t(
            "{0, plural, one {# routing-guide carrier was skipped for eligibility.} other {# routing-guide carriers were skipped for eligibility.}}",
            skipped,
          ),
        });
      } else {
        toast.success(t("Waterfall tender started"), {
          description: t("The rank-1 carrier has been offered the move."),
        });
      }
      invalidateTenders();
    },
    onError: (error: unknown) => describeTenderError(t("Could not start the waterfall"), error),
  });

  const spotTenderMutation = useMutation({
    mutationFn: (payload: SpotTenderPayload) => tenderService.createSpot(payload),
    onSuccess: (tender) => {
      toast.success(
        tender.mode === "SpotBroadcast"
          ? t("Spot broadcast tender sent")
          : tender.mode === "SpotSequential"
            ? t("Spot sequential tender sent")
            : t("Waterfall tender sent"),
        {
          description:
            tender.mode === "SpotBroadcast"
              ? t("Every carrier on the tender has been offered the move.")
              : t("The first carrier on the tender has been offered the move."),
        },
      );
      invalidateTenders();
    },
    onError: (error: unknown) => {
      // Overridable insurance warnings render as a confirm-override prompt inside the
      // tender dialog, so a toast here would double-report them.
      if (isOverridableInsuranceWarningError(error)) return;
      describeTenderError(t("Could not send the spot tender"), error);
    },
  });

  const cancelTenderMutation = useMutation({
    mutationFn: (params: { tenderId: string; reason: string }) =>
      tenderService.cancel(params.tenderId, params.reason),
    onSuccess: () => {
      invalidateTenders();
    },
    onError: (error: unknown) => describeTenderError(t("Could not cancel the tender"), error),
  });

  const recordOfferResponseMutation = useMutation({
    mutationFn: (params: RecordOfferResponseParams) =>
      tenderService.recordResponse(params.offerId, params.payload),
    onSuccess: () => {
      invalidateTenders();
    },
    onError: (error: unknown) => describeTenderError(t("Could not record the response"), error),
  });

  const bounceTenderMutation = useMutation({
    mutationFn: (params: { tenderId: string; payload: BounceTenderPayload }) =>
      tenderService.bounce(params.tenderId, params.payload),
    onSuccess: () => {
      toast.success(t("Carrier backout recorded"), {
        description: t(
          "The carrier assignment was canceled and the tender went on to the next carrier.",
        ),
      });
      invalidateTenders();
    },
    onError: (error: unknown) => describeTenderError(t("Could not record the backout"), error),
  });

  const planMutation = useMutation({
    mutationFn: (input: DispatchPlanInput) => planDispatchAutoAssignGraphQL(input),
    onSuccess: (plan: DispatchPlan) => {
      // A shadow plan wrote nothing; it is kept for review rather than told in a toast.
      if (plan.shadowMode) return;

      // A plan asked for here is only ever proposed: each pairing waits on its move in the
      // queue for a decision, whatever the autonomy tier would allow.
      invalidateBoard();
      if (plan.assignments.length === 0 && plan.uncovered.length === 0) {
        say("ok", t("No uncovered moves in this window"), false);
        return;
      }
      say(
        plan.uncovered.length > 0 ? "warn" : "ok",
        t(
          "{0, plural, =0 {Nova found no driver to propose} one {Nova proposed a driver for # move} other {Nova proposed drivers for # moves}}; {1, plural, =0 {every move has one} one {# has no eligible driver} other {# have no eligible driver}}",
          plan.assignments.length,
          plan.uncovered.length,
        ),
        false,
      );
    },
    onError: (error: Error) =>
      toast.error(t("Couldn't propose coverage"), { description: error.message }),
  });

  const runAssign = assignMutation.mutateAsync;
  const runUnassign = unassignMutation.mutateAsync;

  const guarded = useCallback(async <T>(write: () => Promise<T>): Promise<T | null> => {
    if (writing.current) return null;
    writing.current = true;
    try {
      return await write();
    } catch {
      // The mutation's onError has already said what went wrong.
      return null;
    } finally {
      writing.current = false;
    }
  }, []);

  const assign = useCallback(
    async (requests: AssignRequest[]): Promise<boolean> => {
      const result = await guarded(() => runAssign({ requests, isUndo: false }));
      return result !== null && result.failed === 0;
    },
    [guarded, runAssign],
  );

  const unassign = useCallback(
    async (moveIds: string[], restore?: DispatchAssignMoveInput[]): Promise<boolean> => {
      const result = await guarded(() => runUnassign({ moveIds, restore }));
      return result !== null && result.failed === 0;
    },
    [guarded, runUnassign],
  );

  const approveProposal = useCallback(
    async (decision: ProposalDecision): Promise<boolean> => {
      const result = await guarded(() => approveMutation.mutateAsync(decision));
      return result !== null && !result.error;
    },
    [approveMutation, guarded],
  );

  const declineProposal = useCallback(
    async (decision: ProposalDecision, note: string): Promise<boolean> => {
      const result = await guarded(() => declineMutation.mutateAsync({ ...decision, note }));
      return result !== null && !result.error;
    },
    [declineMutation, guarded],
  );

  const undo = useCallback(async () => {
    const step = undoStack.current.pop();
    syncUndo();
    if (!step) return;
    setOutcome(null);

    // Reversing a step is never itself queued: undoing an undo would loop. A write the
    // server refuses is said in place with its reason; one that threw has already been
    // reported by its mutation.
    const refusal = (result: DispatchBulkAssignResult | null): string | null | undefined => {
      if (result === null) return undefined;
      if (result.failed === 0) return null;
      return result.results.find((item) => !item.success)?.error ?? t("The server refused it.");
    };

    let outcome: string | null | undefined;
    switch (step.kind) {
      case "assigned":
        outcome = refusal(
          await guarded(() => runUnassign({ moveIds: step.moveIds, isUndo: true })),
        );
        break;
      case "restore":
        outcome = refusal(
          await guarded(() =>
            runAssign({
              requests: step.restore.map((input) => ({ input, proNumber: "", driverName: "" })),
              isUndo: true,
            }),
          ),
        );
        break;
      case "approved":
        if (step.executed) {
          outcome = refusal(
            await guarded(() => runUnassign({ moveIds: [step.moveId], isUndo: true })),
          );
        } else {
          const taken = await guarded(() => undoApprovalMutation.mutateAsync(step.proposalId));
          outcome = taken === null ? undefined : null;
        }
        break;
    }
    if (outcome === null) {
      say("ok", step.undone, false);
    } else if (outcome !== undefined) {
      say("warn", t("Couldn't undo: {0}", outcome), false);
    }
  }, [guarded, runAssign, runUnassign, say, syncUndo, t, undoApprovalMutation]);

  return {
    assign,
    unassign,
    approveProposal,
    declineProposal,
    assignToCarrier: carrierAssignMutation.mutateAsync,
    cancelCarrierAssignment: carrierCancelMutation.mutateAsync,
    startWaterfallTender: waterfallMutation.mutateAsync,
    sendSpotTender: spotTenderMutation.mutateAsync,
    cancelTender: cancelTenderMutation.mutateAsync,
    recordOfferResponse: recordOfferResponseMutation.mutateAsync,
    bounceTender: bounceTenderMutation.mutateAsync,
    planAutoAssign: planMutation.mutate,
    /** A shadow plan, held for read-only review; a live plan's proposals are on the board. */
    shadowPlan: planMutation.data?.shadowMode ? planMutation.data : null,
    clearPlan: planMutation.reset,
    undo,
    canUndo: undoTop.depth > 0,
    /** What undo would reverse next, for its label. */
    undoLabel: undoTop.label,
    outcome,
    clearOutcome: () => setOutcome(null),
    isAssigning:
      assignMutation.isPending ||
      unassignMutation.isPending ||
      carrierAssignMutation.isPending ||
      carrierCancelMutation.isPending,
    isDeciding:
      approveMutation.isPending || declineMutation.isPending || undoApprovalMutation.isPending,
    isTendering:
      waterfallMutation.isPending ||
      spotTenderMutation.isPending ||
      cancelTenderMutation.isPending ||
      recordOfferResponseMutation.isPending ||
      bounceTenderMutation.isPending,
    isPlanning: planMutation.isPending,
  };
}
