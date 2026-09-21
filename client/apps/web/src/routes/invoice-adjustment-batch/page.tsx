import { useT } from "@trenova/shared/i18n/use-t";
import { BillingDetailUnselected, BillingListEmpty } from "@/components/billing/billing-empty";
import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { SectionPanel } from "@/components/section-panel";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ExternalLinkIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { type ReactNode, useDeferredValue, useMemo } from "react";
import { Link } from "react-router";
import { invoiceAdjustmentBatchSearchParamsParser } from "./use-invoice-adjustment-batch-state";

const statusChoices = [
  { label: "Queued", value: "Queued" },
  { label: "Submitted", value: "Submitted" },
  { label: "Running", value: "Running" },
  { label: "Completed", value: "Completed" },
  { label: "Failed", value: "Failed" },
  { label: "Partial", value: "PartialSuccess" },
];

export function InvoiceAdjustmentBatchPage() {
  const t = useT();

  const [searchParams, setSearchParams] = useQueryStates(invoiceAdjustmentBatchSearchParamsParser);
  const { item: selectedBatchId, query, status } = searchParams;
  const deferredQuery = useDeferredValue(query);
  const hasActiveFilters = Boolean(query || status);
  const clearFilters = () => void setSearchParams({ query: "", status: null });

  const params = useMemo(() => {
    const next = new URLSearchParams({ limit: "100" });
    if (deferredQuery.trim()) {
      next.set("query", deferredQuery.trim());
    }
    if (status) {
      next.set(
        "fieldFilters",
        JSON.stringify([{ field: "status", operator: "eq", value: status }]),
      );
    }
    return Object.fromEntries(next.entries());
  }, [deferredQuery, status]);

  const listQuery = useQuery({
    ...queries["invoice-adjustment"].batches(params),
  });

  const summaryQuery = useQuery({
    ...queries["invoice-adjustment"].summary(),
  });

  const selectedRow =
    listQuery.data?.results.find((row) => row.batchId === selectedBatchId) ??
    listQuery.data?.results[0] ??
    null;

  const detailQuery = useQuery({
    ...queries["invoice-adjustment"].batch(selectedRow?.batchId ?? ""),
    enabled: Boolean(selectedRow?.batchId),
  });

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        title: t("Batch monitor"),
        description: t(
          "Track bulk adjustment submission progress, failures, and created artifacts.",
        ),
      }}
      toolbar={
        <KpiStrip aria-label={t("Batch totals")}>
          <KpiStripItem
            label={t("Batches in flight")}
            value={String(summaryQuery.data?.batchesInFlight ?? 0)}
          />
          <KpiStripItem
            label={t("Failed items")}
            value={String(summaryQuery.data?.failedBatchItems ?? 0)}
          />
          <KpiStripItem
            label={t("Approvals pending")}
            value={String(summaryQuery.data?.approvalsPending ?? 0)}
          />
          <KpiStripItem
            label={t("Write-offs")}
            value={String(summaryQuery.data?.writeOffPending ?? 0)}
          />
        </KpiStrip>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-2 border-b p-2">
            <Input
              value={query}
              onChange={(event) => void setSearchParams({ query: event.target.value })}
              placeholder={t("Search batch id, submitter, idempotency key...")}
              className="h-8 text-xs"
            />
            <Select
              value={status ?? "all"}
              items={statusChoices}
              onValueChange={(value) =>
                void setSearchParams({ status: value === "all" ? null : value })
              }
            >
              <SelectTrigger className="h-8 text-xs">
                <SelectValue placeholder={t("All statuses")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("All statuses")}</SelectItem>
                {statusChoices.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {t(choice.label)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <ScrollArea className="flex-1">
            <div
              className={cn(
                "flex flex-col gap-1.5 p-2",
                !listQuery.isLoading && listQuery.data?.results.length === 0 && "h-full gap-0 p-0",
              )}
            >
              {listQuery.isLoading
                ? Array.from({ length: 6 }).map((_, index) => (
                    <Skeleton key={index} className="h-24 w-full rounded-xl" />
                  ))
                : null}
              {!listQuery.isLoading && listQuery.data?.results.length === 0 ? (
                <BillingListEmpty
                  title={hasActiveFilters ? "Nothing matches" : "No batches yet"}
                  description={
                    hasActiveFilters
                      ? "No batch fits the search and filters. Widen them, or clear them to see every batch."
                      : "A batch appears here when somebody submits adjustments across several invoices at once, and follows each item as it runs."
                  }
                  onClearFilters={hasActiveFilters ? clearFilters : undefined}
                />
              ) : null}
              {listQuery.data?.results.map((row) => (
                <button
                  key={row.batchId}
                  type="button"
                  onClick={() => void setSearchParams({ item: row.batchId })}
                  className={[
                    "rounded-xl border p-3 text-left transition-colors",
                    row.batchId === selectedRow?.batchId
                      ? "border-primary bg-primary/5"
                      : "hover:bg-muted/40",
                  ].join(" ")}
                >
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">{row.batchId}</p>
                      <p className="text-muted-foreground text-xs">
                        {row.submittedByName || row.submittedById || t("System")}
                      </p>
                    </div>
                    <span className="rounded-full border px-2 py-0.5 text-xs">{row.status}</span>
                  </div>
                  <div className="mt-3 grid grid-cols-4 gap-2 text-xs">
                    <DescriptionItem label={t("Total")}>{String(row.totalCount)}</DescriptionItem>
                    <DescriptionItem label={t("Done")}>
                      {String(row.processedCount)}
                    </DescriptionItem>
                    <DescriptionItem label={t("Failed")}>{String(row.failedCount)}</DescriptionItem>
                    <DescriptionItem label={t("Pending")}>
                      {String(row.pendingCount)}
                    </DescriptionItem>
                  </div>
                  {row.lastFailure ? (
                    <p className="text-destructive mt-3 line-clamp-2 text-xs">{row.lastFailure}</p>
                  ) : null}
                </button>
              ))}
            </div>
          </ScrollArea>
        </div>
      }
      detail={
        <ScrollArea className="h-full">
          {!selectedRow ? (
            <BillingDetailUnselected
              layout="cards"
              title={t("Nothing open")}
              description={t(
                "Pick a batch from the list to see how each item ran and what it created.",
              )}
            />
          ) : detailQuery.isLoading || !detailQuery.data ? (
            <div className="space-y-4 p-4">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-64 w-full" />
            </div>
          ) : (
            <div className="space-y-4 p-4">
              <SectionPanel
                title={selectedRow.batchId}
                hint={selectedRow.submittedByName || selectedRow.submittedById || t("System batch")}
              >
                <DescriptionList columns={4} className="p-3">
                  <DescriptionItem label={t("Total")}>
                    {String(selectedRow.totalCount)}
                  </DescriptionItem>
                  <DescriptionItem label={t("Succeeded")}>
                    {String(selectedRow.succeededCount)}
                  </DescriptionItem>
                  <DescriptionItem label={t("Failed")}>
                    {String(selectedRow.failedCount)}
                  </DescriptionItem>
                  <DescriptionItem label={t("Pending")}>
                    {String(selectedRow.pendingCount)}
                  </DescriptionItem>
                  <DescriptionItem label={t("Submitted At")}>
                    {formatTimestamp(selectedRow.submittedAt)}
                  </DescriptionItem>
                  <DescriptionItem label={t("Status")}>{selectedRow.status}</DescriptionItem>
                  <DescriptionItem label={t("Last failure count")}>
                    {String(selectedRow.lastFailureCount)}
                  </DescriptionItem>
                  <DescriptionItem label={t("Idempotency key")} valueClassName="break-all">
                    {selectedRow.idempotencyKey}
                  </DescriptionItem>
                </DescriptionList>
              </SectionPanel>

              <SectionPanel
                title={t("Item results")}
                help={t("Per-item outcome, failure reason, and created artifacts.")}
              >
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="bg-muted/40 text-muted-foreground text-left">
                      <tr>
                        <th className="px-4 py-3">{t("Invoice")}</th>
                        <th className="px-4 py-3">{t("Status")}</th>
                        <th className="px-4 py-3">{t("Failure")}</th>
                        <th className="px-4 py-3">{t("Artifacts")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {detailQuery.data.items.map((item) => (
                        <tr key={item.id} className="border-t align-top">
                          <td className="px-4 py-3 font-mono text-xs">{item.invoiceId}</td>
                          <td className="px-4 py-3">{item.status}</td>
                          <td className="text-muted-foreground px-4 py-3 text-xs">
                            {item.errorMessage || t("No failure recorded")}
                          </td>
                          <td className="px-4 py-3">
                            <div className="flex flex-wrap gap-2">
                              <LinkButton to={`/billing/invoices?item=${item.invoiceId}`}>
                                {t("Invoice")}
                              </LinkButton>
                              {item.adjustmentId ? (
                                <LinkButton
                                  to={`/billing/pending-approvals?item=${item.adjustmentId}`}
                                >
                                  {t("Adjustment")}
                                </LinkButton>
                              ) : null}
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </SectionPanel>
            </div>
          )}
        </ScrollArea>
      }
    />
  );
}

function LinkButton({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link
      to={to}
      className="hover:bg-muted inline-flex items-center gap-1 rounded-full border px-3 py-1.5 text-xs"
    >
      <ExternalLinkIcon className="size-3.5" />
      {children}
    </Link>
  );
}

function formatTimestamp(value: number | null | undefined) {
  return formatUnixDateTime(value, { fallback: "Not recorded" });
}
