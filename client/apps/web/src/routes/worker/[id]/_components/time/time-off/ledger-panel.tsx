import { SectionPanel } from "@/components/section-panel";
import {
  fetchWorkerPtoLedger,
  WORKER_PTO_LEDGER_KEY,
  type WorkerPTOLedgerEntry,
} from "@/lib/graphql/pto-policy";
import { ptoTypeLabel } from "@/routes/worker/_components/pto/worker-pto-balances";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { PTO_LEDGER_ENTRY_LABELS } from "@trenova/shared/types/pto-policy";
import {
  BOUNDED_CELL_CLASS,
  BoundedTable,
  type BoundedColumn,
  type BoundedRowProps,
} from "../../bounded-table";
import { useWorkerList } from "../../use-worker-list";

export const LEDGER_PAGE_SIZE = 25;

const COLUMNS = [
  { label: "Date" },
  { label: "Type" },
  { label: "Entry" },
  { label: "Days", numeric: true },
  { label: "Balance", numeric: true },
] as const;

function ledgerColumns(t: TranslateFn): BoundedColumn[] {
  return [
    { key: "date", label: t("Date") },
    { key: "type", label: t("Type") },
    { key: "entry", label: t("Entry") },
    { key: "days", label: t("Days"), align: "end" },
    { key: "balance", label: t("Balance"), align: "end" },
  ];
}

/** A ledger amount with its sign: "+1.00", "−5.00". */
export function signedDays(amount: string): string {
  const value = Number(amount);
  if (!Number.isFinite(value)) return amount;
  return `${value > 0 ? "+" : value < 0 ? "−" : ""}${Math.abs(value).toFixed(2)}`;
}

/**
 * Every posting behind the balances, newest first: accruals, time off taken, adjustments. It
 * reads a page at a time as the reader scrolls to the end, and mounts only the rows in view.
 */
export function LedgerPanel({ workerId }: { workerId: string }) {
  const t = useT();
  const ledger = useWorkerList({
    root: WORKER_PTO_LEDGER_KEY,
    workerId,
    pageSize: LEDGER_PAGE_SIZE,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerPtoLedger(
        {
          workerId,
          first,
          after: after ?? undefined,
          sort: [{ field: "effectiveAt", direction: "desc" }],
        },
        { signal },
      ),
  });

  return (
    <SectionPanel title={t("Ledger")} hint={t("Newest first")}>
      {ledger.isPending ? (
        <div aria-busy aria-label={t("Loading the ledger")} className="p-3">
          <div aria-hidden className="h-48 w-full ui-shimmer rounded-md" />
        </div>
      ) : ledger.isError ? (
        <div className="p-3">
          <ErrorState
            layout="compact"
            error={ledger.error}
            onRetry={ledger.refetch}
            title={t("The ledger could not be loaded")}
          />
        </div>
      ) : ledger.rows.length === 0 ? (
        <EmptyTable
          title={t("Nothing posted yet")}
          description={t("Accruals post on the policy's schedule from the effective date.")}
          columns={COLUMNS}
        />
      ) : (
        <BoundedTable
          label={t("Ledger")}
          columns={ledgerColumns(t)}
          maxHeightClassName="max-h-88"
          rows={ledger.rows.map((entry) => ({
            key: entry.id,
            render: (props) => <LedgerRow props={props} entry={entry} />,
          }))}
          more={ledger}
        />
      )}
    </SectionPanel>
  );
}

function LedgerRow({ entry, props }: { entry: WorkerPTOLedgerEntry; props: BoundedRowProps }) {
  const t = useT();
  const amount = Number(entry.amountDays);
  return (
    <tr {...props} data-glide-target>
      <td className={cn(BOUNDED_CELL_CLASS, "font-mono text-xs tabular-nums")}>
        {formatUnixDateMedium(entry.effectiveAt)}
      </td>
      <td className={BOUNDED_CELL_CLASS}>{t(ptoTypeLabel(entry.ptoType))}</td>
      <td className={cn(BOUNDED_CELL_CLASS, "max-w-56")}>
        <span className="block truncate">{PTO_LEDGER_ENTRY_LABELS[entry.entryType]}</span>
        {entry.note ? (
          <span className="block truncate text-xs text-muted-foreground" title={entry.note}>
            {entry.note}
          </span>
        ) : null}
      </td>
      <td
        className={cn(
          BOUNDED_CELL_CLASS,
          "text-right font-mono text-xs tabular-nums",
          amount > 0 && "text-success-foreground",
        )}
      >
        {signedDays(entry.amountDays)}
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "text-right font-mono text-xs tabular-nums")}>
        {Number(entry.balanceAfterDays).toFixed(2)}
      </td>
    </tr>
  );
}
