import { downloadCsv, exportFilename } from "@/lib/data-table-export";
import type {
  BillingTransferCandidate,
  BillingTransferCandidateFilters,
} from "@/lib/graphql/billing-transfer";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { DownloadIcon, RotateCcwIcon, SendIcon } from "lucide-react";
import { useCallback, useDeferredValue, useMemo, useState } from "react";
import { billingTransferCandidatesQuery } from "../../billing-queue-queries";
import { BulkBillingTransferCandidates } from "./bulk-billing-transfer-candidates";
import {
  BulkBillingTransferProgress,
  BulkBillingTransferResolving,
} from "./bulk-billing-transfer-progress";
import {
  buildBulkBillingTransferReportCsv,
  retryableShipmentIds,
} from "./bulk-billing-transfer-report";
import { BulkBillingTransferResults } from "./bulk-billing-transfer-results";
import { useBulkBillingTransfer } from "./use-bulk-billing-transfer";
import { MAX_BILLING_TRANSFER_CANDIDATE_IDS } from "./run-bulk-billing-transfer";

const EMPTY_FILTERS: BillingTransferCandidateFilters = { query: "", status: null };

type BulkBillingTransferDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/**
 * Moves many shipments into the billing queue at once.
 *
 * Every shipment still goes through the same readiness check a single transfer
 * does, so the outcome is per shipment: the report is the point of the dialog,
 * not an afterthought, because the shipments that did not transfer are the work
 * the biller has left to do.
 */
export function BulkBillingTransferDialog({ open, onOpenChange }: BulkBillingTransferDialogProps) {
  const t = useT();

  const [filters, setFilters] = useState<BillingTransferCandidateFilters>(EMPTY_FILTERS);
  const deferredQuery = useDeferredValue(filters.query);
  const listFilters = useMemo<BillingTransferCandidateFilters>(
    () => ({ query: deferredQuery.trim(), status: filters.status }),
    [deferredQuery, filters.status],
  );

  const [selectedIds, setSelectedIds] = useState<ReadonlySet<string>>(() => new Set());
  const [proNumbers, setProNumbers] = useState<ReadonlyMap<string, string>>(() => new Map());
  const [confirmingAll, setConfirmingAll] = useState(false);

  const { state, isBusy, start, retry, stop, reset } = useBulkBillingTransfer();

  const candidatesQuery = useInfiniteQuery({
    ...billingTransferCandidatesQuery(listFilters),
    enabled: open && state.phase === "idle",
  });

  const candidates = useMemo<BillingTransferCandidate[]>(
    () => candidatesQuery.data?.pages.flatMap((page) => page.edges.map((edge) => edge.node)) ?? [],
    [candidatesQuery.data],
  );
  const totalCount = candidatesQuery.data?.pages[0]?.totalCount ?? null;

  const handleFiltersChange = useCallback((next: BillingTransferCandidateFilters) => {
    setFilters(next);
    setConfirmingAll(false);
  }, []);

  const handleSelectionChange = useCallback(
    (changed: BillingTransferCandidate[], selected: boolean) => {
      setSelectedIds((current) => {
        const next = new Set(current);
        for (const candidate of changed) {
          if (selected) next.add(candidate.id);
          else next.delete(candidate.id);
        }
        return next;
      });
      if (selected) {
        setProNumbers((current) => {
          const next = new Map(current);
          for (const candidate of changed) next.set(candidate.id, candidate.proNumber);
          return next;
        });
      }
    },
    [],
  );

  const clearAll = useCallback(() => {
    setFilters(EMPTY_FILTERS);
    setSelectedIds(new Set());
    setProNumbers(new Map());
    setConfirmingAll(false);
    reset();
  }, [reset]);

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen && isBusy) return;
      onOpenChange(nextOpen);
      if (!nextOpen) clearAll();
    },
    [clearAll, isBusy, onOpenChange],
  );

  const transferSelected = () => {
    void start({ kind: "selected", shipmentIds: [...selectedIds] });
  };

  const transferAll = () => {
    setConfirmingAll(false);
    void start({ kind: "all", filters: listFilters });
  };

  const finished = state.phase === "finished" ? state : null;
  const retryIds = finished ? retryableShipmentIds(finished.outcome) : [];

  const downloadReport = () => {
    if (!finished) return;
    downloadCsv(
      buildBulkBillingTransferReportCsv(finished.outcome, t),
      exportFilename("billing-transfer-report"),
    );
  };

  const startNewTransfer = () => {
    setSelectedIds(new Set());
    setConfirmingAll(false);
    reset();
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="flex max-h-[90vh] flex-col sm:max-w-4xl" showCloseButton={!isBusy}>
        <DialogHeader>
          <DialogTitle>{t("Transfer to Billing")}</DialogTitle>
          <DialogDescription>
            {t(
              "Each shipment runs the billing readiness check before it moves into the queue. Completed shipments that pass are marked Ready to Invoice first.",
            )}
          </DialogDescription>
        </DialogHeader>

        {state.phase === "idle" ? (
          <>
            {state.error ? (
              <div
                role="alert"
                className="border-destructive/40 bg-destructive/5 rounded-lg border px-3 py-2 text-xs"
              >
                <p className="font-medium">{t("The eligible shipments could not be found")}</p>
                {state.error instanceof Error && state.error.message ? (
                  <p className="text-muted-foreground">{state.error.message}</p>
                ) : null}
              </div>
            ) : null}
            <BulkBillingTransferCandidates
              filters={filters}
              onFiltersChange={handleFiltersChange}
              candidates={candidates}
              totalCount={totalCount}
              isLoading={candidatesQuery.isPending}
              isError={candidatesQuery.isError}
              onRetryLoad={() => void candidatesQuery.refetch()}
              hasNextPage={candidatesQuery.hasNextPage}
              isFetchingNextPage={candidatesQuery.isFetchingNextPage}
              onLoadMore={() => void candidatesQuery.fetchNextPage()}
              selectedIds={selectedIds}
              onSelectionChange={handleSelectionChange}
            />
          </>
        ) : state.phase === "resolving" ? (
          <BulkBillingTransferResolving />
        ) : state.phase === "running" ? (
          <BulkBillingTransferProgress
            totalCount={state.totalCount}
            processedCount={state.processedCount}
            results={state.results}
          />
        ) : (
          <BulkBillingTransferResults
            outcome={state.outcome}
            stopped={state.stopped}
            error={state.error}
            unmatchedCount={state.unmatchedCount}
            proNumbers={proNumbers}
          />
        )}

        <DialogFooter className="flex-wrap items-center gap-2 sm:justify-between">
          {state.phase === "idle" && confirmingAll && totalCount !== null ? (
            <>
              <div className="flex flex-col text-xs">
                <span className="font-medium">
                  {t(
                    "{0, plural, one {Transfer all # shipment that matches the current search and status?} other {Transfer all # shipments that match the current search and status?}}",
                    totalCount,
                  )}
                </span>
                {totalCount > MAX_BILLING_TRANSFER_CANDIDATE_IDS ? (
                  <span className="text-muted-foreground">
                    {t(
                      "One run transfers the oldest {0}. Run it again for the rest.",
                      formatNumber(MAX_BILLING_TRANSFER_CANDIDATE_IDS),
                    )}
                  </span>
                ) : null}
              </div>
              <div className="flex items-center gap-2">
                <Button variant="outline" onClick={() => setConfirmingAll(false)}>
                  {t("Back")}
                </Button>
                <Button onClick={transferAll}>
                  <SendIcon className="size-3.5" />
                  {t("Start transfer")}
                </Button>
              </div>
            </>
          ) : state.phase === "idle" ? (
            <>
              <div className="text-muted-foreground flex items-center gap-2 text-xs">
                <span>{t("{0} selected", selectedIds.size)}</span>
                {selectedIds.size > 0 ? (
                  <Button variant="ghost" size="xs" onClick={() => setSelectedIds(new Set())}>
                    {t("Clear selection")}
                  </Button>
                ) : null}
              </div>
              <div className="flex items-center gap-2">
                <Button variant="outline" onClick={() => handleOpenChange(false)}>
                  {t("Cancel")}
                </Button>
                {totalCount ? (
                  <Button variant="outline" onClick={() => setConfirmingAll(true)}>
                    {t("Transfer all {0}", totalCount)}
                  </Button>
                ) : null}
                <Button onClick={transferSelected} disabled={selectedIds.size === 0}>
                  <SendIcon className="size-3.5" />
                  {t("Transfer {0} selected", selectedIds.size)}
                </Button>
              </div>
            </>
          ) : state.phase === "running" ? (
            <Button
              variant="outline"
              className="ml-auto"
              onClick={stop}
              disabled={state.stopRequested}
            >
              {state.stopRequested ? t("Stopping after this batch...") : t("Stop")}
            </Button>
          ) : state.phase === "finished" ? (
            <>
              <Button variant="outline" onClick={downloadReport}>
                <DownloadIcon className="size-3.5" />
                {t("Download report")}
              </Button>
              <div className="flex items-center gap-2">
                {retryIds.length > 0 ? (
                  <Button variant="outline" onClick={() => void retry(retryIds)}>
                    <RotateCcwIcon className="size-3.5" />
                    {t("Retry {0}", retryIds.length)}
                  </Button>
                ) : null}
                <Button variant="outline" onClick={startNewTransfer}>
                  {t("Transfer more")}
                </Button>
                <Button onClick={() => handleOpenChange(false)}>{t("Done")}</Button>
              </div>
            </>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
