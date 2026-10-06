import {
  assignDispatchMovesGraphQL,
  planDispatchAutoAssignGraphQL,
} from "@/lib/graphql/dispatch-console";
import {
  bulkTransferShipmentsToBillingGraphQL,
  listBillingTransferCandidateIdsGraphQL,
} from "@/lib/graphql/billing-transfer";
import {
  decideShipmentSuggestionGraphQL,
  notifyShipmentDelayGraphQL,
  tenderShipmentsGraphQL,
  undoShipmentSuggestionDecisionGraphQL,
} from "@/lib/graphql/shipment-board";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  ShipmentSuggestionDecision,
  TenderShipmentItemInput,
} from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";
import { SHIPMENT_LIST_KEY } from "../shipment-queries";

export type AssignDriverInput = {
  moveId: string;
  workerId: string;
  tractorId?: string | null;
};

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Every write the board makes, each followed by a refresh of the rows and of
 * the board's own aggregates, so a committed choice shows up in the table, the
 * capacity strip, the queue and the watchlist together.
 */
export function useBoardActions() {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [SHIPMENT_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: queries.shipmentBoard._def }),
      queryClient.invalidateQueries({ queryKey: ["shipment-events"] }),
    ]);
  }, [queryClient]);

  const assign = useMutation({
    mutationFn: async (items: AssignDriverInput[]) => {
      const result = await assignDispatchMovesGraphQL(
        items.map((item) => {
          if (!item.tractorId) {
            throw new Error(t("This driver has no tractor to dispatch with"));
          }
          return { moveId: item.moveId, primaryWorkerId: item.workerId, tractorId: item.tractorId };
        }),
      );
      const failure = result.results.find((entry) => !entry.success);
      if (failure) throw new Error(failure.error ?? t("The driver could not be assigned"));
      return result;
    },
    onSettled: refresh,
    onError: (error) => toast.error(t("Assignment failed"), { description: errorMessage(error) }),
  });

  const autoAssign = useMutation({
    mutationFn: async (moveIds: string[]) => {
      const plan = await planDispatchAutoAssignGraphQL({ moveIds, apply: false });
      if (plan.assignments.length === 0) {
        throw new Error(t("No available driver fits the selected loads"));
      }
      return assignDispatchMovesGraphQL(
        plan.assignments
          .filter((assignment) => !!assignment.tractorId)
          .map((assignment) => ({
            moveId: assignment.moveId,
            primaryWorkerId: assignment.workerId,
            tractorId: assignment.tractorId as string,
            ...(assignment.trailerId ? { trailerId: assignment.trailerId } : {}),
          })),
      );
    },
    onSettled: refresh,
    onSuccess: (result) => toast.success(t("Assigned {0} of the selected loads", result.succeeded)),
    onError: (error) => toast.error(t("Assignment failed"), { description: errorMessage(error) }),
  });

  const tender = useMutation({
    mutationFn: (items: TenderShipmentItemInput[]) => tenderShipmentsGraphQL({ items }),
    onSettled: refresh,
    onSuccess: (result) => {
      if (result.failed.length > 0) {
        toast.warning(t("{0} could not be tendered", result.failed.length), {
          description: result.failed[0]?.message,
        });
      }
    },
    onError: (error) => toast.error(t("Tender failed"), { description: errorMessage(error) }),
  });

  const decide = useMutation({
    mutationFn: (input: { key: string; decision: ShipmentSuggestionDecision }) =>
      decideShipmentSuggestionGraphQL(input),
    onSettled: refresh,
  });

  const undo = useMutation({
    mutationFn: (key: string) => undoShipmentSuggestionDecisionGraphQL(key),
    onSettled: refresh,
  });

  const notifyDelay = useMutation({
    mutationFn: (input: { shipmentId: string; message: string }) =>
      notifyShipmentDelayGraphQL(input),
    onSettled: refresh,
    onError: (error) =>
      toast.error(t("The delay notice was not sent"), { description: errorMessage(error) }),
  });

  const approveDetention = useMutation({
    mutationFn: (occurrenceId: string) => apiService.detentionService.approve(occurrenceId),
    onSettled: refresh,
    onError: (error) =>
      toast.error(t("Detention was not approved"), { description: errorMessage(error) }),
  });

  const transferReadyToBill = useMutation({
    mutationFn: async () => {
      const candidates = await listBillingTransferCandidateIdsGraphQL({ query: "", status: null });
      if (candidates.ids.length === 0) return null;
      return bulkTransferShipmentsToBillingGraphQL(candidates.ids);
    },
    onSettled: refresh,
    onError: (error) =>
      toast.error(t("Transfer to billing failed"), { description: errorMessage(error) }),
  });

  return {
    assign,
    autoAssign,
    tender,
    decide,
    undo,
    notifyDelay,
    approveDetention,
    transferReadyToBill,
    refresh,
  };
}
