import { downloadCsv, exportFilename } from "@/lib/data-table-export";
import type {
  BillingTransferCandidate,
  BillingTransferCandidateFilters,
  BillingTransferRunItem,
} from "@/lib/graphql/billing-transfer";
import { listBillingTransferRunItemsGraphQL } from "@/lib/graphql/billing-transfer";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
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
  BulkBillingTransferLoading,
  BulkBillingTransferProgress,
} from "./bulk-billing-transfer-progress";
import { buildBulkBillingTransferReportCsv } from "./bulk-billing-transfer-report";
import { BulkBillingTransferResults } from "./bulk-billing-transfer-results";
import {
  canRetryRun,
  isRunStopping,
  isRunTerminal,
  MAX_BILLING_TRANSFER_CANDIDATE_IDS,
} from "./bulk-billing-transfer-run";
import { useBulkBillingTransfer } from "./use-bulk-billing-transfer";

const EMPTY_FILTERS: BillingTransferCandidateFilters = { query: "", status: null };

const REPORT_PAGE_SIZE = 250;

type BulkBillingTransferDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  runId: string | null;
  onRunIdChange: (runId: string | null) => void;
};

/**
 * Moves many shipments into the billing queue at once.
 *
 * The run itself belongs to the server, so this dialog picks shipments, starts
 * a run and then watches it. Closing it does not stop anything — the biller is
 * told when the run finishes and can reopen this to read the report, which is
 * the point of the report: the shipments that did not transfer are the work
 * they have left to do.
 */
export function BulkBillingTransferDialog({
  open,
  onOpenChange,
  runId,
  onRunIdChange,
}: BulkBillingTransferDialogProps) {
  const t = useT();

  const [filters, setFilters] = useState<BillingTransferCandidateFilters>(EMPTY_FILTERS);
  const deferredQuery = useDeferredValue(filters.query);
  const listFilters = useMemo<BillingTransferCandidateFilters>(
    () => ({ query: deferredQuery.trim(), status: filters.status }),
    [deferredQuery, filters.status],
  );

  const [selectedIds, setSelectedIds] = useState<ReadonlySet<string>>(() => new Set());
  const [confirmingAll, setConfirmingAll] = useState(false);
  const [downloading, setDownloading] = useState(false);

  const { run, isReattaching, isStarting, isStopping, startError, start, retry, stop, clear } =
    useBulkBillingTransfer(runId, onRunIdChange);

  // Arriving with a run id — from the completion toast, or a refresh mid-run —
  // must not flash the shipment picker on the way to the report.
  const loadingRun = Boolean(runId) && !run;
  const showPicker = !runId && !run;
  const finished = run !== null && isRunTerminal(run);

  const candidatesQuery = useInfiniteQuery({
    ...billingTransferCandidatesQuery(listFilters),
    enabled: open && showPicker,
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
    },
    [],
  );

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      onOpenChange(nextOpen);
      if (!nextOpen) {
        setFilters(EMPTY_FILTERS);
        setSelectedIds(new Set());
        setConfirmingAll(false);
      }
    },
    [onOpenChange],
  );

  const transferSelected = () => {
    void start({ kind: "selected", shipmentIds: [...selectedIds] });
  };

  const transferAll = () => {
    setConfirmingAll(false);
    void start({ kind: "all", filters: listFilters });
  };

  // The report can run to thousands of rows, so it is paged out of the server
  // on demand rather than held in memory the whole time the dialog is open.
  const downloadReport = async () => {
    if (!run) return;

    setDownloading(true);
    try {
      const rows: BillingTransferRunItem[] = [];
      let after: string | null = null;
      do {
        const page = await listBillingTransferRunItemsGraphQL({
          runId: run.id,
          first: REPORT_PAGE_SIZE,
          after,
        });
        rows.push(...page.edges.map((edge) => edge.node));
        after = page.pageInfo.hasNextPage ? page.pageInfo.endCursor : null;
      } while (after);

      downloadCsv(
        buildBulkBillingTransferReportCsv(rows, t),
        exportFilename("billing-transfer-report"),
      );
    } finally {
      setDownloading(false);
    }
  };

  const startNewTransfer = () => {
    setSelectedIds(new Set());
    setConfirmingAll(false);
    clear();
    onRunIdChange(null);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent size="xl" className="flex max-h-[90vh] flex-col">
        <DialogHeader>
          <DialogTitle>{t("Transfer to billing")}</DialogTitle>
          <DialogDescription>
            {t(
              "Each shipment runs the billing readiness check before it moves into the queue. Completed shipments that pass are marked Ready to Invoice first.",
            )}
          </DialogDescription>
        </DialogHeader>

        {run === null ? (
          loadingRun ? (
            <BulkBillingTransferLoading />
          ) : (
            <>
              {startError ? (
                <Alert variant="destructive" size="sm">
                  <AlertTitle>{t("The transfer could not be started")}</AlertTitle>
                  {startError instanceof Error && startError.message ? (
                    <AlertDescription>{startError.message}</AlertDescription>
                  ) : null}
                </Alert>
              ) : null}
              <BulkBillingTransferCandidates
                filters={filters}
                onFiltersChange={handleFiltersChange}
                candidates={candidates}
                totalCount={totalCount}
                isLoading={candidatesQuery.isPending || isReattaching}
                isError={candidatesQuery.isError}
                onRetryLoad={() => void candidatesQuery.refetch()}
                hasNextPage={candidatesQuery.hasNextPage}
                isFetchingNextPage={candidatesQuery.isFetchingNextPage}
                onLoadMore={() => void candidatesQuery.fetchNextPage()}
                selectedIds={selectedIds}
                onSelectionChange={handleSelectionChange}
              />
            </>
          )
        ) : finished ? (
          <BulkBillingTransferResults run={run} />
        ) : (
          <BulkBillingTransferProgress run={run} />
        )}

        <DialogFooter className="flex-wrap items-center gap-2 sm:justify-between">
          {loadingRun ? null : showPicker && confirmingAll && totalCount !== null ? (
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
                <Button onClick={transferAll} disabled={isStarting}>
                  <SendIcon className="size-3.5" />
                  {t("Start transfer")}
                </Button>
              </div>
            </>
          ) : showPicker ? (
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
                <Button onClick={transferSelected} disabled={selectedIds.size === 0 || isStarting}>
                  <SendIcon className="size-3.5" />
                  {t("Transfer {0} selected", selectedIds.size)}
                </Button>
              </div>
            </>
          ) : finished ? (
            <>
              <Button
                variant="outline"
                onClick={() => void downloadReport()}
                disabled={downloading}
              >
                <DownloadIcon className="size-3.5" />
                {downloading ? t("Preparing...") : t("Download report")}
              </Button>
              <div className="flex items-center gap-2">
                {canRetryRun(run) ? (
                  <Button variant="outline" onClick={() => void retry()} disabled={isStarting}>
                    <RotateCcwIcon className="size-3.5" />
                    {t("Retry {0}", run.retryableCount + run.skippedCount)}
                  </Button>
                ) : null}
                <Button variant="outline" onClick={startNewTransfer}>
                  {t("Transfer more")}
                </Button>
                <Button onClick={() => handleOpenChange(false)}>{t("Done")}</Button>
              </div>
            </>
          ) : (
            <>
              <span className="text-muted-foreground text-xs">
                {t("This transfer keeps running if you close this window.")}
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  onClick={stop}
                  disabled={isStopping || isRunStopping(run)}
                >
                  {isRunStopping(run) ? t("Stopping after this batch...") : t("Stop")}
                </Button>
                <Button onClick={() => handleOpenChange(false)}>{t("Run in background")}</Button>
              </div>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
