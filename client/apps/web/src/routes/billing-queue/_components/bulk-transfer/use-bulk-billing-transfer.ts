import {
  bulkTransferShipmentsToBillingGraphQL,
  listBillingTransferCandidateIdsGraphQL,
  type BillingTransferCandidateFilters,
  type BulkBillingTransferResult,
} from "@/lib/graphql/billing-transfer";
import { queries } from "@/lib/queries";
import { SHIPMENT_LIST_KEY } from "@/routes/shipment/_components/shipment-queries";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  BILLING_QUEUE_LIST_KEY,
  BILLING_TRANSFER_CANDIDATES_KEY,
} from "../../billing-queue-queries";
import { invalidateStatements } from "../../statement-queries";
import {
  mergeBulkBillingTransferRetry,
  type BulkBillingTransferOutcome,
} from "./bulk-billing-transfer-report";
import { runBulkBillingTransfer } from "./run-bulk-billing-transfer";

export type BulkBillingTransferTarget =
  | { kind: "selected"; shipmentIds: string[] }
  | { kind: "all"; filters: BillingTransferCandidateFilters };

export type BulkBillingTransferState =
  | { phase: "idle"; error: unknown }
  | { phase: "resolving" }
  | {
      phase: "running";
      totalCount: number;
      processedCount: number;
      results: BulkBillingTransferResult[];
      stopRequested: boolean;
    }
  | {
      phase: "finished";
      outcome: BulkBillingTransferOutcome;
      stopped: boolean;
      error: unknown;
      unmatchedCount: number;
    };

const IDLE: BulkBillingTransferState = { phase: "idle", error: null };

export function useBulkBillingTransfer() {
  const queryClient = useQueryClient();
  const [state, setState] = useState<BulkBillingTransferState>(IDLE);
  const stopRef = useRef(false);
  const mountedRef = useRef(true);
  const resolveControllerRef = useRef<AbortController | null>(null);

  const isBusy = state.phase === "resolving" || state.phase === "running";

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      stopRef.current = true;
      resolveControllerRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    if (!isBusy) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [isBusy]);

  const invalidate = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: [BILLING_QUEUE_LIST_KEY] });
    void queryClient.invalidateQueries({ queryKey: queries.billingQueue._def });
    void queryClient.invalidateQueries({ queryKey: [BILLING_TRANSFER_CANDIDATES_KEY] });
    void queryClient.invalidateQueries({ queryKey: [SHIPMENT_LIST_KEY] });
    invalidateStatements(queryClient);
  }, [queryClient]);

  const transfer = useCallback(
    async (
      shipmentIds: string[],
      options: { previous: BulkBillingTransferOutcome | null; unmatchedCount: number },
    ) => {
      stopRef.current = false;
      setState({
        phase: "running",
        totalCount: new Set(shipmentIds).size,
        processedCount: 0,
        results: [],
        stopRequested: false,
      });

      const run = await runBulkBillingTransfer({
        shipmentIds,
        transfer: bulkTransferShipmentsToBillingGraphQL,
        shouldStop: () => stopRef.current,
        onProgress: ({ processedCount, results }) => {
          if (!mountedRef.current) return;
          setState((current) =>
            current.phase === "running" ? { ...current, processedCount, results } : current,
          );
        },
      });

      if (shipmentIds.length > 0) invalidate();
      if (!mountedRef.current) return;

      const outcome = { results: run.results, notProcessedIds: run.notProcessedIds };
      setState({
        phase: "finished",
        outcome: options.previous
          ? mergeBulkBillingTransferRetry(options.previous, outcome)
          : outcome,
        stopped: run.stopped,
        error: run.error,
        unmatchedCount: options.unmatchedCount,
      });
    },
    [invalidate],
  );

  const start = useCallback(
    async (target: BulkBillingTransferTarget) => {
      if (target.kind === "selected") {
        await transfer(target.shipmentIds, { previous: null, unmatchedCount: 0 });
        return;
      }

      setState({ phase: "resolving" });
      const controller = new AbortController();
      resolveControllerRef.current = controller;
      try {
        const candidates = await listBillingTransferCandidateIdsGraphQL(target.filters, {
          signal: controller.signal,
        });
        if (!mountedRef.current) return;
        await transfer(candidates.ids, {
          previous: null,
          unmatchedCount: candidates.truncated
            ? Math.max(candidates.totalCount - candidates.ids.length, 0)
            : 0,
        });
      } catch (error) {
        if (!mountedRef.current || controller.signal.aborted) return;
        setState({ phase: "idle", error });
      } finally {
        resolveControllerRef.current = null;
      }
    },
    [transfer],
  );

  const retry = useCallback(
    async (shipmentIds: string[]) => {
      if (state.phase !== "finished" || shipmentIds.length === 0) return;
      await transfer(shipmentIds, {
        previous: state.outcome,
        unmatchedCount: state.unmatchedCount,
      });
    },
    [state, transfer],
  );

  const stop = useCallback(() => {
    stopRef.current = true;
    setState((current) =>
      current.phase === "running" ? { ...current, stopRequested: true } : current,
    );
  }, []);

  const reset = useCallback(() => {
    stopRef.current = false;
    setState(IDLE);
  }, []);

  return { state, isBusy, start, retry, stop, reset };
}
