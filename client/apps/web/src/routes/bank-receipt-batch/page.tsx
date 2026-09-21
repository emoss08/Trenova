import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
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
        title: t("Import batches"),
        description: t("View and create bank receipt import batches."),
        actions: (
          <Button size="sm" onClick={() => setDialogOpen(true)}>
            <UploadIcon className="size-3.5" />
            {t("Import batch")}
          </Button>
        ),
      }}
    >
      <KpiStrip aria-label={t("Import batch summary")}>
        <KpiStripItem
          label={t("Total batches")}
          value={String(stats.total)}
          sub={formatCurrency(stats.totalAmount / 100)}
        />
        <KpiStripItem label={t("Processing")} value={String(stats.processing)} />
        <KpiStripItem label={t("Completed")} value={String(stats.completed)} />
      </KpiStrip>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-lg" />
          ))}
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-lg border border-danger-border bg-danger-subtle p-4 text-sm text-danger-foreground dark:border-danger-border dark:bg-danger-subtle dark:text-danger-foreground">
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
                <th className="px-3 py-2.5 text-right text-xs font-medium">{t("Total amount")}</th>
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
                    <span className="text-xs font-medium text-success-foreground">
                      {batch.matchedCount}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-right tabular-nums">
                    <span className="text-xs font-medium text-danger-foreground">
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

      <ImportBatchDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </PageLayout>
  );
}
