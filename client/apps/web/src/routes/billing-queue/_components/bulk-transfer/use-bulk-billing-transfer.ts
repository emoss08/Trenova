import {
  cancelBillingTransferRunGraphQL,
  retryBillingTransferRunGraphQL,
  startBillingTransferRunGraphQL,
  type BillingTransferCandidateFilters,
  type BillingTransferRun,
} from "@/lib/graphql/billing-transfer";
import { queries } from "@/lib/queries";
import { SHIPMENT_LIST_KEY } from "@/routes/shipment/_components/shipment-queries";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  BILLING_QUEUE_LIST_KEY,
  BILLING_TRANSFER_ACTIVE_RUN_KEY,
  BILLING_TRANSFER_CANDIDATES_KEY,
  BILLING_TRANSFER_RUN_ITEMS_KEY,
  BILLING_TRANSFER_RUN_KEY,
  billingTransferRunQuery,
  myActiveBillingTransferRunQuery,
} from "../../billing-queue-queries";
import { invalidateStatements } from "../../statement-queries";
import { isRunTerminal } from "./bulk-billing-transfer-run";

export type BulkBillingTransferTarget =
  | { kind: "selected"; shipmentIds: string[] }
  | { kind: "all"; filters: BillingTransferCandidateFilters };

/**
 * Watches a transfer run.
 *
 * The run belongs to the server, so this hook starts one, reads it, and asks it
 * to stop — it never does the work itself. That is what lets the dialog be
 * closed, the tab be refreshed, and a second tab pick the same run up, none of
 * which the old client-driven loop could survive.
 */
export function useBulkBillingTransfer(runId: string | null, onRunIdChange: (id: string) => void) {
  const queryClient = useQueryClient();
  const [startError, setStartError] = useState<unknown>(null);
  const settledRunRef = useRef<string | null>(null);

  const runQuery = useQuery(billingTransferRunQuery(runId));
  const run = runQuery.data ?? null;

  // Only consulted when there is nothing to show yet, so an explicitly opened
  // run is never overridden by whatever else the user has going.
  const activeQuery = useQuery({
    ...myActiveBillingTransferRunQuery(),
    enabled: !runId,
  });

  useEffect(() => {
    const active = activeQuery.data;
    if (!runId && active) {
      onRunIdChange(active.id);
    }
  }, [activeQuery.data, runId, onRunIdChange]);

  const invalidateBillingViews = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: [BILLING_QUEUE_LIST_KEY] });
    void queryClient.invalidateQueries({ queryKey: queries.billingQueue._def });
    void queryClient.invalidateQueries({ queryKey: [BILLING_TRANSFER_CANDIDATES_KEY] });
    void queryClient.invalidateQueries({ queryKey: [SHIPMENT_LIST_KEY] });
    invalidateStatements(queryClient);
  }, [queryClient]);

  // The queue only settles once, when the run reaches a terminal state: every
  // batch moves shipments, but refetching the whole billing queue after each of
  // them is the throttling this change exists to remove.
  useEffect(() => {
    if (!run || !isRunTerminal(run)) return;
    if (settledRunRef.current === run.id) return;

    settledRunRef.current = run.id;
    invalidateBillingViews();
    void queryClient.invalidateQueries({ queryKey: [BILLING_TRANSFER_ACTIVE_RUN_KEY] });
  }, [run, invalidateBillingViews, queryClient]);

  const adoptRun = useCallback(
    (next: BillingTransferRun) => {
      queryClient.setQueryData([BILLING_TRANSFER_RUN_KEY, next.id], next);
      void queryClient.invalidateQueries({ queryKey: [BILLING_TRANSFER_ACTIVE_RUN_KEY] });
      onRunIdChange(next.id);
    },
    [queryClient, onRunIdChange],
  );

  const startMutation = useMutation({
    mutationFn: (target: BulkBillingTransferTarget) =>
      target.kind === "selected"
        ? startBillingTransferRunGraphQL({
            scope: "Selected",
            shipmentIds: target.shipmentIds,
          })
        : startBillingTransferRunGraphQL({
            scope: "AllMatching",
            query: target.filters.query,
            status: target.filters.status,
          }),
    onSuccess: adoptRun,
  });

  const retryMutation = useMutation({
    mutationFn: (id: string) => retryBillingTransferRunGraphQL(id),
    onSuccess: adoptRun,
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => cancelBillingTransferRunGraphQL(id),
    onSuccess: (next) => {
      queryClient.setQueryData([BILLING_TRANSFER_RUN_KEY, next.id], next);
    },
  });

  const start = useCallback(
    async (target: BulkBillingTransferTarget) => {
      setStartError(null);
      try {
        await startMutation.mutateAsync(target);
      } catch (error) {
        setStartError(error);
      }
    },
    [startMutation],
  );

  const retry = useCallback(async () => {
    if (!run) return;
    setStartError(null);
    try {
      await retryMutation.mutateAsync(run.id);
    } catch (error) {
      setStartError(error);
    }
  }, [run, retryMutation]);

  const stop = useCallback(() => {
    if (!run) return;
    cancelMutation.mutate(run.id);
  }, [run, cancelMutation]);

  // Clearing only drops what this dialog is looking at. The run itself, if one
  // is still going, keeps going — that is the point of it living server side.
  const clear = useCallback(() => {
    setStartError(null);
    settledRunRef.current = null;
    void queryClient.invalidateQueries({ queryKey: [BILLING_TRANSFER_RUN_ITEMS_KEY] });
  }, [queryClient]);

  return {
    run,
    isLoadingRun: Boolean(runId) && runQuery.isPending,
    isReattaching: !runId && activeQuery.isPending,
    isStarting: startMutation.isPending || retryMutation.isPending,
    isStopping: cancelMutation.isPending,
    startError,
    runError: runQuery.error,
    start,
    retry,
    stop,
    clear,
  };
}
