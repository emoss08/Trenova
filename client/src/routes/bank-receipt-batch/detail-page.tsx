import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { EmptyState } from "@/components/empty-state";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import type { BankReceipt } from "@/types/bank-receipt";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeftIcon,
  CheckCircle2Icon,
  FileWarningIcon,
  ReceiptTextIcon,
  UploadCloudIcon,
} from "lucide-react";
import { useNavigate, useParams } from "react-router";

function formatTimestamp(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatDate(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function SummaryCard({
  label,
  count,
  amount,
  icon: Icon,
  colorClass,
}: {
  label: string;
  count: number;
  amount: number;
  icon: React.ComponentType<{ className?: string }>;
  colorClass?: string;
}) {
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="flex items-center gap-2">
        <Icon className={`size-4 ${colorClass ?? "text-muted-foreground"}`} />
        <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
          {label}
        </p>
      </div>
      <p className="mt-1.5 text-2xl font-semibold tabular-nums">{count}</p>
      <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
        {formatCurrency(amount / 100)}
      </p>
    </div>
  );
}

export function BankReceiptBatchDetailPage() {
  const { batchId } = useParams<{ batchId: string }>();
  const navigate = useNavigate();

  const { data, isLoading, isError } = useQuery({
    ...queries.bankReceiptBatch.get(batchId!),
    enabled: Boolean(batchId),
  });

  const batch = data?.batch;
  const receipts = data?.receipts ?? [];

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: "Import Batch",
          description: "Loading...",
        }}
      >
        <div className="mx-4 mt-3 space-y-4">
          <Skeleton className="h-8 w-48" />
          <div className="grid gap-2.5 md:grid-cols-3">
            <Skeleton className="h-24 rounded-lg" />
            <Skeleton className="h-24 rounded-lg" />
            <Skeleton className="h-24 rounded-lg" />
          </div>
          <Skeleton className="h-64 w-full rounded-lg" />
        </div>
      </PageLayout>
    );
  }

  if (isError || !batch) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: "Import Batch",
          description: "Failed to load batch details.",
        }}
      >
        <div className="mx-4 mt-3">
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
            Could not load this import batch. It may have been deleted or you may not have
            permission.
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="mt-3"
            onClick={() => void navigate("/accounting/reconciliation/import-batches")}
          >
            <ArrowLeftIcon className="mr-1.5 size-3.5" />
            Back to Batches
          </Button>
        </div>
      </PageLayout>
    );
  }

  return (
    <PageLayout
      pageHeaderProps={{
        title: batch.reference || "Import Batch",
        description: `Source: ${batch.source}`,
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <div className="flex items-center justify-between">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void navigate("/accounting/reconciliation/import-batches")}
          >
            <ArrowLeftIcon className="mr-1.5 size-3.5" />
            Back to Batches
          </Button>
          <AccountingStatusBadge status={batch.status} />
        </div>

        <div className="grid gap-2.5 md:grid-cols-3">
          <SummaryCard
            label="Imported"
            count={batch.importedCount}
            amount={batch.importedAmountMinor}
            icon={UploadCloudIcon}
          />
          <SummaryCard
            label="Matched"
            count={batch.matchedCount}
            amount={batch.matchedAmountMinor}
            icon={CheckCircle2Icon}
            colorClass="text-green-600 dark:text-green-400"
          />
          <SummaryCard
            label="Exceptions"
            count={batch.exceptionCount}
            amount={batch.exceptionAmountMinor}
            icon={FileWarningIcon}
            colorClass="text-red-600 dark:text-red-400"
          />
        </div>

        <div className="rounded-lg border bg-card p-4">
          <h3 className="mb-3 text-sm font-semibold">Batch Info</h3>
          <div className="grid gap-4 sm:grid-cols-4">
            <div>
              <p className="text-2xs font-medium text-muted-foreground">Source</p>
              <p className="text-xs font-medium">{batch.source}</p>
            </div>
            <div>
              <p className="text-2xs font-medium text-muted-foreground">Reference</p>
              <p className="font-mono text-xs font-medium">{batch.reference || "\u2014"}</p>
            </div>
            <div>
              <p className="text-2xs font-medium text-muted-foreground">Status</p>
              <div className="mt-0.5">
                <AccountingStatusBadge status={batch.status} />
              </div>
            </div>
            <div>
              <p className="text-2xs font-medium text-muted-foreground">Created</p>
              <p className="text-xs">{formatTimestamp(batch.createdAt ?? 0)}</p>
            </div>
          </div>
        </div>

        <div className="rounded-lg border bg-card">
          <div className="flex items-center justify-between border-b px-4 py-3">
            <h3 className="text-sm font-semibold">
              Receipts
              <span className="ml-1.5 text-xs font-normal text-muted-foreground">
                ({receipts.length})
              </span>
            </h3>
          </div>

          {receipts.length === 0 ? (
            <div className="flex justify-center py-12">
              <EmptyState
                title="No receipts"
                description="This batch contains no receipts."
                icons={[ReceiptTextIcon]}
                className="max-w-none border-none shadow-none"
              />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-left text-muted-foreground">
                  <tr>
                    <th className="px-4 py-2.5 text-xs font-medium">Reference #</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Date</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">Amount</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Status</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Memo</th>
                  </tr>
                </thead>
                <tbody>
                  {receipts.map((receipt: BankReceipt) => (
                    <tr key={receipt.id} className="border-t">
                      <td className="px-4 py-2.5 font-mono text-xs font-medium">
                        {receipt.referenceNumber}
                      </td>
                      <td className="px-4 py-2.5 text-xs">
                        {formatDate(receipt.receiptDate)}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        <AmountDisplay
                          value={receipt.amountMinor}
                          className="text-xs font-medium"
                        />
                      </td>
                      <td className="px-4 py-2.5">
                        <AccountingStatusBadge status={receipt.status} />
                      </td>
                      <td className="px-4 py-2.5 text-xs text-muted-foreground">
                        {receipt.memo || "\u2014"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  );
}
