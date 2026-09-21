import { useT } from "@trenova/shared/i18n/use-t";
import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel } from "@/components/section-panel";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { BankReceipt } from "@/types/bank-receipt";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { formatUnixDateMedium, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";

const RECEIPT_COLUMNS = [
  { label: "Reference" },
  { label: "Date" },
  { label: "Amount", numeric: true },
  { label: "Status" },
  { label: "Memo" },
] as const;

function formatTimestamp(unix: number): string {
  return formatUnixDateTimeMedium(unix);
}

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix);
}

export function BankReceiptBatchDetailPage() {
  const t = useT();

  const { batchId } = useParams<{ batchId: string }>();
  const navigate = useNavigate();

  const { data, isLoading, isError } = useQuery({
    ...queries.bankReceiptBatch.get(batchId!),
    enabled: Boolean(batchId),
  });

  const batch = data?.batch;
  const receipts = data?.receipts ?? [];

  const backButton = (
    <Button
      variant="outline"
      size="sm"
      onClick={() => void navigate("/accounting/reconciliation/import-batches")}
    >
      <ArrowLeftIcon className="size-3.5" />
      {t("Back to Batches")}
    </Button>
  );

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Import Batch"),
          description: t("Loading..."),
        }}
      >
        <Skeleton className="h-20 w-full rounded-lg" />
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </PageLayout>
    );
  }

  if (isError || !batch) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Import Batch"),
          description: t("Failed to load batch details."),
          actions: backButton,
        }}
      >
        <Alert variant="destructive" size="sm">
          <AlertDescription>
            {t(
              "Could not load this import batch. It may have been deleted or you may not have permission.",
            )}
          </AlertDescription>
        </Alert>
      </PageLayout>
    );
  }

  return (
    <PageLayout
      pageHeaderProps={{
        title: batch.reference || "Import Batch",
        description: `Source: ${batch.source}`,
        context: <AccountingStatusBadge status={batch.status} />,
        actions: backButton,
      }}
    >
      <KpiStrip aria-label={t("Batch totals")}>
        <KpiStripItem
          label={t("Imported")}
          value={batch.importedCount}
          sub={formatCurrency(batch.importedAmountMinor / 100)}
        />
        <KpiStripItem
          label={t("Matched")}
          value={batch.matchedCount}
          sub={formatCurrency(batch.matchedAmountMinor / 100)}
          tone="success"
        />
        <KpiStripItem
          label={t("Exceptions")}
          value={batch.exceptionCount}
          sub={formatCurrency(batch.exceptionAmountMinor / 100)}
          tone="danger"
        />
      </KpiStrip>

      <SectionPanel title={t("Batch info")}>
        <DescriptionList columns={4} className="p-3">
          <DescriptionItem label={t("Source")}>{batch.source}</DescriptionItem>
          <DescriptionItem label={t("Reference")} valueClassName="font-mono">
            {batch.reference || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Status")}>
            <AccountingStatusBadge status={batch.status} />
          </DescriptionItem>
          <DescriptionItem label={t("Created")}>
            {formatTimestamp(batch.createdAt ?? 0)}
          </DescriptionItem>
        </DescriptionList>
      </SectionPanel>

      <SectionPanel title={t("Receipts")} count={receipts.length}>
        {receipts.length === 0 ? (
          <EmptyTable
            title={t("No receipts in this batch")}
            description={t(
              "The file imported with nothing in it, or every line was rejected. Import it again once it has rows.",
            )}
            columns={RECEIPT_COLUMNS}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-muted-foreground text-left">
                <tr>
                  <th className="px-4 py-2.5 text-xs font-medium">{t("Reference #")}</th>
                  <th className="px-4 py-2.5 text-xs font-medium">{t("Date")}</th>
                  <th className="px-4 py-2.5 text-right text-xs font-medium">{t("Amount")}</th>
                  <th className="px-4 py-2.5 text-xs font-medium">{t("Status")}</th>
                  <th className="px-4 py-2.5 text-xs font-medium">{t("Memo")}</th>
                </tr>
              </thead>
              <tbody>
                {receipts.map((receipt: BankReceipt) => (
                  <tr key={receipt.id} className="border-t">
                    <td className="px-4 py-2.5 font-mono text-xs font-medium">
                      {receipt.referenceNumber}
                    </td>
                    <td className="px-4 py-2.5 text-xs">{formatDate(receipt.receiptDate)}</td>
                    <td className="px-4 py-2.5 text-right">
                      <AmountDisplay value={receipt.amountMinor} className="text-xs font-medium" />
                    </td>
                    <td className="px-4 py-2.5">
                      <AccountingStatusBadge status={receipt.status} />
                    </td>
                    <td className="text-muted-foreground px-4 py-2.5 text-xs">
                      {receipt.memo || "\u2014"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </SectionPanel>
    </PageLayout>
  );
}
