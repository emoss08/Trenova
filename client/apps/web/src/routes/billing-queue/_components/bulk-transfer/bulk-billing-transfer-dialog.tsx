import { downloadCsv, exportFilename } from "@/lib/data-table-export";
import type {
  BillingTransferCandidate,
  BillingTransferCandidateFilters,
  BillingTransferRunItem,
} from "@/lib/graphql/billing-transfer";
import { listBillingTransferRunItemsGraphQL } from "@/lib/graphql/billing-transfer";
import { billingTransferResultKey, markResultSeen, watchResult } from "@/lib/watched-results";
import { useInfiniteQuery } from "@tanstack/react-query";
import { AnimatePresence, m, useReducedMotionConfig } from "motion/react";
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
import { toast } from "@trenova/shared/components/ui/toast";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
import { cn } from "@trenova/shared/lib/utils";
import { Download01Icon, RefreshCcw01Icon, Send01Icon } from "@trenova/shared/components/icons";
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from "react";
import { billingTransferCandidatesQuery } from "../../billing-queue-queries";
import { BulkBillingTransferCandidates } from "./bulk-billing-transfer-candidates";
import { BulkBillingTransferRunSkeleton } from "./bulk-billing-transfer-dialog-skeleton";
import { BulkBillingTransferProgress } from "./bulk-billing-transfer-progress";
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

/** The stage folding away and the results opening in its place: one eased height change each. */
const HANDOVER = { duration: 0.56, ease: [0.65, 0, 0.35, 1] } as const;

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

  // A run that finishes while its stage is on screen keeps the stage until the last bill
  // lands, then hands over to the results. A run that had already finished when the dialog
  // opened goes straight to its results.
  const [stagedRunId, setStagedRunId] = useState<string | null>(null);
  const [settledRunId, setSettledRunId] = useState<string | null>(null);
  if (open && run !== null && !finished && stagedRunId !== run.id) {
    setStagedRunId(run.id);
  }
  const staged = run !== null && stagedRunId === run.id;
  const act: "stage" | "results" | null =
    run === null ? null : !finished || (staged && settledRunId !== run.id) ? "stage" : "results";
  const activeRunId = run?.id ?? null;
  const handOver = useCallback(() => setSettledRunId(activeRunId), [activeRunId]);
  // While the dialog shows a run, the run's finished notice is not news: it waits, and is
  // dropped once the results are on screen. Closing before then lets it through.
  useEffect(() => {
    if (!open || activeRunId === null) return;
    return watchResult(billingTransferResultKey(activeRunId));
  }, [open, activeRunId]);

  useEffect(() => {
    if (open && act === "results" && activeRunId !== null) {
      markResultSeen(billingTransferResultKey(activeRunId));
    }
  }, [open, act, activeRunId]);

  const reduceMotion = useReducedMotionConfig() ?? false;
  const handover = reduceMotion ? { duration: 0 } : HANDOVER;

  const candidatesQuery = useInfiniteQuery({
    ...billingTransferCandidatesQuery(listFilters),
    enabled: open && showPicker,
  });

  const candidates = useMemo<BillingTransferCandidate[]>(
    () => candidatesQuery.data?.pages.flatMap((page) => page.edges.map((edge) => edge.node)) ?? [],
    [candidatesQuery.data],
  );
  const totalCount = candidatesQuery.data?.pages[0]?.totalCount ?? null;
  const totalCountExact = candidatesQuery.data?.pages[0]?.totalCountExact !== false;

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
        setStagedRunId(null);
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
    } catch {
      toast.error(t("The transfer report could not be downloaded. Try again."));
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
            {act === "stage"
              ? t(
                  "The transfer runs in the background. You can close this window and you'll be notified when it finishes.",
                )
              : act === "results"
                ? t("What moved into the billing queue, and what is still holding the rest back.")
                : t(
                    "Each shipment runs the billing readiness check before it moves into the queue. Completed shipments that pass are marked Ready to Invoice first.",
                  )}
          </DialogDescription>
        </DialogHeader>

        {startError ? (
          <Alert variant="destructive" size="sm">
            <AlertTitle>
              {finished
                ? t("The retry could not be started")
                : t("The transfer could not be started")}
            </AlertTitle>
            {startError instanceof Error && startError.message ? (
              <AlertDescription>{startError.message}</AlertDescription>
            ) : null}
          </Alert>
        ) : null}

        {run === null ? (
          loadingRun ? (
            <BulkBillingTransferRunSkeleton />
          ) : (
            <BulkBillingTransferCandidates
              filters={filters}
              onFiltersChange={handleFiltersChange}
              candidates={candidates}
              totalCount={totalCount}
              totalCountExact={totalCountExact}
              isLoading={candidatesQuery.isPending || isReattaching}
              isError={candidatesQuery.isError}
              onRetryLoad={() => void candidatesQuery.refetch()}
              hasNextPage={candidatesQuery.hasNextPage}
              isFetchingNextPage={candidatesQuery.isFetchingNextPage}
              onLoadMore={() => void candidatesQuery.fetchNextPage()}
              selectedIds={selectedIds}
              onSelectionChange={handleSelectionChange}
            />
          )
        ) : null}

        <AnimatePresence mode="wait" initial={false}>
          {run !== null && act === "stage" ? (
            <m.div
              key={`stage-${run.id}`}
              className="overflow-hidden"
              exit={{ height: 0, opacity: 0 }}
              transition={handover}
            >
              <BulkBillingTransferProgress run={run} onSettled={handOver} />
            </m.div>
          ) : null}
          {run !== null && act === "results" ? (
            <m.div
              key={`results-${run.id}`}
              className="flex min-h-0 flex-col overflow-hidden"
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              transition={handover}
            >
              <BulkBillingTransferResults run={run} animateIn={staged} />
            </m.div>
          ) : null}
        </AnimatePresence>

        <DialogFooter
          className={cn(
            "flex-wrap items-center gap-2",
            act === "stage" ? "sm:justify-end" : "sm:justify-between",
          )}
        >
          {loadingRun ? null : showPicker && confirmingAll && totalCount !== null ? (
            <>
              <div className="flex flex-col text-xs">
                <span className="font-medium">
                  {totalCountExact
                    ? t(
                        "{0, plural, one {Transfer all # shipment that matches the current search and status?} other {Transfer all # shipments that match the current search and status?}}",
                        totalCount,
                      )
                    : t(
                        "Transfer all of about {0} shipments that match the current search and status?",
                        formatRowTotal(totalCount, false),
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
                  <Send01Icon className="size-3.5" />
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
                    {totalCountExact
                      ? t("Transfer all {0}", totalCount)
                      : t("Transfer all of about {0}", formatRowTotal(totalCount, false))}
                  </Button>
                ) : null}
                <Button onClick={transferSelected} disabled={selectedIds.size === 0 || isStarting}>
                  <Send01Icon className="size-3.5" />
                  {t("Transfer {0} selected", selectedIds.size)}
                </Button>
              </div>
            </>
          ) : act === "results" && run !== null ? (
            <>
              <Button
                variant="outline"
                onClick={() => void downloadReport()}
                disabled={downloading}
              >
                <Download01Icon className="size-3.5" />
                {downloading ? t("Preparing...") : t("Download report")}
              </Button>
              <div className="flex items-center gap-2">
                {canRetryRun(run) ? (
                  <Button variant="outline" onClick={() => void retry()} disabled={isStarting}>
                    <RefreshCcw01Icon className="size-3.5" />
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
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                onClick={stop}
                disabled={finished || isStopping || isRunStopping(run)}
              >
                {isRunStopping(run) ? t("Stopping after this batch...") : t("Stop transfer")}
              </Button>
              <Button onClick={() => handleOpenChange(false)}>{t("Run in background")}</Button>
            </div>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
