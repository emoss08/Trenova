import { SectionPanel } from "@/components/section-panel";
import {
  fetchWorkerPtoLedger,
  WORKER_PTO_LEDGER_KEY,
  type WorkerPTOLedgerEntry,
} from "@/lib/graphql/pto-policy";
import { ptoTypeLabel } from "@/routes/worker/_components/pto/worker-pto-balances";
import { useInfiniteQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { PTO_LEDGER_ENTRY_LABELS } from "@trenova/shared/types/pto-policy";
import { useEffect, useRef } from "react";

export const LEDGER_PAGE_SIZE = 25;

const COLUMNS = [
  { label: "Date" },
  { label: "Type" },
  { label: "Entry" },
  { label: "Days", numeric: true },
  { label: "Balance", numeric: true },
] as const;

/** A ledger amount with its sign: "+1.00", "−5.00". */
export function signedDays(amount: string): string {
  const value = Number(amount);
  if (!Number.isFinite(value)) return amount;
  return `${value > 0 ? "+" : value < 0 ? "−" : ""}${Math.abs(value).toFixed(2)}`;
}

/**
 * Every posting behind the balances, newest first: accruals, time off taken, adjustments. It
 * reads a page at a time and asks for the next as the reader scrolls to the end.
 */
export function LedgerPanel({ workerId }: { workerId: string }) {
  const t = useT();
  const sentinelRef = useRef<HTMLTableRowElement>(null);
  const query = useInfiniteQuery({
    queryKey: [WORKER_PTO_LEDGER_KEY, workerId, "pages"],
    initialPageParam: null as string | null,
    queryFn: ({ pageParam, signal }) =>
      fetchWorkerPtoLedger(
        {
          workerId,
          first: LEDGER_PAGE_SIZE,
          after: pageParam ?? undefined,
          sort: [{ field: "effectiveAt", direction: "desc" }],
        },
        { signal },
      ),
    getNextPageParam: (page) => (page.hasNextPage && page.endCursor ? page.endCursor : null),
  });
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;
  const entries = dedupe(query.data?.pages.flatMap((page) => page.entries) ?? []);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !hasNextPage || typeof IntersectionObserver === "undefined") return;
    const observer = new IntersectionObserver((records) => {
      if (records.some((record) => record.isIntersecting) && !isFetchingNextPage) {
        void fetchNextPage();
      }
    });
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <SectionPanel title={t("Ledger")} hint={t("Newest first")}>
      {query.isPending ? (
        <div aria-busy aria-label={t("Loading the ledger")} className="p-3">
          <div aria-hidden className="h-48 w-full ui-shimmer rounded-md" />
        </div>
      ) : query.isError ? (
        <div className="p-3">
          <ErrorState
            layout="compact"
            error={query.error}
            onRetry={() => void query.refetch()}
            title={t("The ledger could not be loaded")}
          />
        </div>
      ) : entries.length === 0 ? (
        <EmptyTable
          title={t("Nothing posted yet")}
          description={t("Accruals post on the policy's schedule from the effective date.")}
          columns={COLUMNS}
        />
      ) : (
        <ScrollArea className="max-h-88" maskVariant="card">
          <table className="w-full min-w-[29rem] text-sm">
            <thead className="sticky top-0 z-10 bg-card">
              <tr className="border-b text-left text-xs text-foreground-subtle">
                <th className="px-3 py-2 font-medium">{t("Date")}</th>
                <th className="px-3 py-2 font-medium">{t("Type")}</th>
                <th className="px-3 py-2 font-medium">{t("Entry")}</th>
                <th className="px-3 py-2 text-right font-medium">{t("Days")}</th>
                <th className="px-3 py-2 text-right font-medium">{t("Balance")}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <LedgerRow key={entry.id} entry={entry} />
              ))}
              {hasNextPage ? (
                <tr ref={sentinelRef} aria-busy={isFetchingNextPage}>
                  <td colSpan={5} className="px-3 py-2.5">
                    <span
                      aria-hidden
                      className={cn(
                        "block h-2 w-32 rounded-sm",
                        isFetchingNextPage ? "ui-shimmer" : "bg-transparent",
                      )}
                    />
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </ScrollArea>
      )}
    </SectionPanel>
  );
}

function dedupe(entries: readonly WorkerPTOLedgerEntry[]): WorkerPTOLedgerEntry[] {
  const seen = new Set<string>();
  return entries.filter((entry) => {
    if (seen.has(entry.id)) return false;
    seen.add(entry.id);
    return true;
  });
}

function LedgerRow({ entry }: { entry: WorkerPTOLedgerEntry }) {
  const t = useT();
  const amount = Number(entry.amountDays);
  return (
    <tr className="border-b border-border-subtle last:border-0">
      <td className="px-3 py-2 font-mono text-xs tabular-nums">
        {formatUnixDateMedium(entry.effectiveAt)}
      </td>
      <td className="px-3 py-2">{t(ptoTypeLabel(entry.ptoType))}</td>
      <td className="max-w-56 px-3 py-2">
        <span className="block truncate">{PTO_LEDGER_ENTRY_LABELS[entry.entryType]}</span>
        {entry.note ? (
          <span className="block truncate text-xs text-muted-foreground" title={entry.note}>
            {entry.note}
          </span>
        ) : null}
      </td>
      <td
        className={cn(
          "px-3 py-2 text-right font-mono text-xs tabular-nums",
          amount > 0 && "text-success-foreground",
        )}
      >
        {signedDays(entry.amountDays)}
      </td>
      <td className="px-3 py-2 text-right font-mono text-xs tabular-nums">
        {Number(entry.balanceAfterDays).toFixed(2)}
      </td>
    </tr>
  );
}
