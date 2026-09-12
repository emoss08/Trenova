import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import type { BankReceiptBatch } from "@/types/bank-receipt-batch";
import { useQuery } from "@tanstack/react-query";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, UploadIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { ImportBatchDialog } from "./_components/import-batch-dialog";

const BATCH_COLUMNS = [
  { label: "Reference" },
  { label: "Source" },
  { label: "Status" },
  { label: "Imported", numeric: true },
  { label: "Matched", numeric: true },
  { label: "Exceptions", numeric: true },
  { label: "Total", numeric: true },
] as const;

function formatTimestamp(unix: number): string {
  return formatUnixDateTimeMedium(unix);
}

function SummaryCard({ label, value, amount }: { label: string; value: string; amount?: number }) {
  return (
    <div className="bg-card rounded-lg border px-3 py-2.5">
      <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
      {amount !== undefined ? (
        <p className="text-muted-foreground mt-0.5 text-xs tabular-nums">
          {formatCurrency(amount / 100)}
        </p>
      ) : null}
    </div>
  );
}

export function BankReceiptBatchPage() {
  const t = useT();

  const navigate = useNavigate();
  const [dialogOpen, setDialogOpen] = useState(false);

  const {
    data: batches,
    isLoading,
    isError,
  } = useQuery({
    ...queries.bankReceiptBatch.list(),
  });

  const stats = useMemo(() => {
    if (!batches) return { total: 0, processing: 0, completed: 0, totalAmount: 0 };
    return {
      total: batches.length,
      processing: batches.filter((b) => b.status === "Processing").length,
      completed: batches.filter((b) => b.status === "Completed").length,
      totalAmount: batches.reduce((sum, b) => sum + b.importedAmountMinor, 0),
    };
  }, [batches]);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Import Batches"),
        description: t("View and create bank receipt import batches."),
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <div className="flex items-center justify-between">
          <div className="grid flex-1 gap-2.5 md:grid-cols-4">
            <SummaryCard
              label={t("Total Batches")}
              value={String(stats.total)}
              amount={stats.totalAmount}
            />
            <SummaryCard label={t("Processing")} value={String(stats.processing)} />
            <SummaryCard label={t("Completed")} value={String(stats.completed)} />
            <div className="flex items-end">
              <Button size="sm" onClick={() => setDialogOpen(true)}>
                <UploadIcon className="mr-1.5 size-3.5" />
                {t("Import Batch")}
              </Button>
            </div>
          </div>
        </div>

        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        ) : null}

        {isError ? (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
            {t("Failed to load import batches. Try refreshing the page.")}
          </div>
        ) : null}

        {!isLoading && !isError && batches && batches.length === 0 ? (
          <EmptyTable
            title={t("No batches yet")}
            description={t(
              "Import a bank receipt file and it becomes a batch here, with every receipt it carried and how many of them matched.",
            )}
            columns={BATCH_COLUMNS}
            action={
              <Button variant="outline" size="sm" onClick={() => setDialogOpen(true)}>
                <UploadIcon className="size-3.5" />
                {t("Import a batch")}
              </Button>
            }
          />
        ) : null}

        {!isLoading && !isError && batches && batches.length > 0 ? (
          <div className="overflow-hidden rounded-lg border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-muted-foreground text-left">
                <tr>
                  <th className="px-3 py-2.5 text-xs font-medium">{t("Reference")}</th>
                  <th className="px-3 py-2.5 text-xs font-medium">{t("Source")}</th>
                  <th className="px-3 py-2.5 text-xs font-medium">{t("Status")}</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">{t("Imported")}</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">{t("Matched")}</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">{t("Exceptions")}</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">
                    {t("Total Amount")}
                  </th>
                  <th className="px-3 py-2.5 text-xs font-medium">{t("Created")}</th>
                  <th className="w-10 px-3 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {batches.map((batch: BankReceiptBatch) => (
                  <tr
                    key={batch.id}
                    className="hover:bg-muted/40 cursor-pointer border-t transition-colors"
                    onClick={() =>
                      void navigate(`/accounting/reconciliation/import-batches/${batch.id}`)
                    }
                  >
                    <td className="px-3 py-2.5">
                      <span className="font-mono text-xs font-medium">
                        {batch.reference || "\u2014"}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-xs">{batch.source}</td>
                    <td className="px-3 py-2.5">
                      <AccountingStatusBadge status={batch.status} />
                    </td>
                    <td className="px-3 py-2.5 text-right tabular-nums">
                      <span className="text-xs font-medium">{batch.importedCount}</span>
                    </td>
                    <td className="px-3 py-2.5 text-right tabular-nums">
                      <span className="text-xs font-medium text-green-600 dark:text-green-400">
                        {batch.matchedCount}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-right tabular-nums">
                      <span className="text-xs font-medium text-red-600 dark:text-red-400">
                        {batch.exceptionCount}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <AmountDisplay
                        value={batch.importedAmountMinor}
                        className="text-xs font-medium"
                      />
                    </td>
                    <td className="text-muted-foreground px-3 py-2.5 text-xs">
                      {formatTimestamp(batch.createdAt ?? 0)}
                    </td>
                    <td className="px-3 py-2.5">
                      <ArrowRightIcon className="text-muted-foreground size-3.5" />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </div>

      <ImportBatchDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </PageLayout>
  );
}
