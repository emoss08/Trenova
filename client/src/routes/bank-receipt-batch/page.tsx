import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { EmptyState } from "@/components/empty-state";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import type { BankReceiptBatch } from "@/types/bank-receipt-batch";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRightIcon,
  FileStackIcon,
  ImportIcon,
  ReceiptTextIcon,
  UploadIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { ImportBatchDialog } from "./_components/import-batch-dialog";

function formatTimestamp(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function SummaryCard({
  label,
  value,
  amount,
}: {
  label: string;
  value: string;
  amount?: number;
}) {
  return (
    <div className="rounded-lg border bg-card px-3 py-2.5">
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
      {amount !== undefined ? (
        <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
          {formatCurrency(amount / 100)}
        </p>
      ) : null}
    </div>
  );
}

export function BankReceiptBatchPage() {
  const navigate = useNavigate();
  const [dialogOpen, setDialogOpen] = useState(false);

  const { data: batches, isLoading, isError } = useQuery({
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
        title: "Import Batches",
        description: "View and create bank receipt import batches.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <div className="flex items-center justify-between">
          <div className="grid flex-1 gap-2.5 md:grid-cols-4">
            <SummaryCard
              label="Total Batches"
              value={String(stats.total)}
              amount={stats.totalAmount}
            />
            <SummaryCard label="Processing" value={String(stats.processing)} />
            <SummaryCard label="Completed" value={String(stats.completed)} />
            <div className="flex items-end">
              <Button size="sm" onClick={() => setDialogOpen(true)}>
                <UploadIcon className="mr-1.5 size-3.5" />
                Import Batch
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
            Failed to load import batches. Try refreshing the page.
          </div>
        ) : null}

        {!isLoading && !isError && batches && batches.length === 0 ? (
          <div className="flex justify-center pt-12">
            <EmptyState
              title="No import batches"
              description="Import your first bank receipt batch to start reconciliation."
              icons={[FileStackIcon, ReceiptTextIcon, ImportIcon]}
              action={{
                icon: UploadIcon,
                label: "Import Batch",
                onClick: () => setDialogOpen(true),
              }}
            />
          </div>
        ) : null}

        {!isLoading && !isError && batches && batches.length > 0 ? (
          <div className="overflow-hidden rounded-lg border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left text-muted-foreground">
                <tr>
                  <th className="px-3 py-2.5 text-xs font-medium">Reference</th>
                  <th className="px-3 py-2.5 text-xs font-medium">Source</th>
                  <th className="px-3 py-2.5 text-xs font-medium">Status</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">Imported</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">Matched</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">Exceptions</th>
                  <th className="px-3 py-2.5 text-right text-xs font-medium">Total Amount</th>
                  <th className="px-3 py-2.5 text-xs font-medium">Created</th>
                  <th className="w-10 px-3 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {batches.map((batch: BankReceiptBatch) => (
                  <tr
                    key={batch.id}
                    className="cursor-pointer border-t transition-colors hover:bg-muted/40"
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
                    <td className="px-3 py-2.5 text-xs text-muted-foreground">
                      {formatTimestamp(batch.createdAt ?? 0)}
                    </td>
                    <td className="px-3 py-2.5">
                      <ArrowRightIcon className="size-3.5 text-muted-foreground" />
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
