import { EmptyState } from "@/components/empty-state";
import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, CheckCircle2Icon, ExternalLinkIcon, Layers3Icon } from "lucide-react";
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
  const [searchParams, setSearchParams] = useQueryStates(invoiceAdjustmentBatchSearchParamsParser);
  const { item: selectedBatchId, query, status } = searchParams;
  const deferredQuery = useDeferredValue(query);

  const params = useMemo(() => {
    const next = new URLSearchParams({ limit: "100" });
    if (deferredQuery.trim()) {
      next.set("query", deferredQuery.trim());
    }
    if (status) {
      next.set("fieldFilters", JSON.stringify([{ field: "status", operator: "eq", value: status }]));
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
    listQuery.data?.results.find((row) => row.batchId === selectedBatchId) ?? listQuery.data?.results[0] ?? null;

  const detailQuery = useQuery({
    ...queries["invoice-adjustment"].batch(selectedRow?.batchId ?? ""),
    enabled: Boolean(selectedRow?.batchId),
  });

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        title: "Batch Monitor",
        description: "Track bulk adjustment submission progress, failures, and created artifacts.",
      }}
      toolbar={
        <div className="mx-4 mt-3 grid gap-3 md:grid-cols-4">
          <SummaryCard label="Batches In Flight" value={String(summaryQuery.data?.batchesInFlight ?? 0)} />
          <SummaryCard label="Failed Items" value={String(summaryQuery.data?.failedBatchItems ?? 0)} />
          <SummaryCard label="Approvals Pending" value={String(summaryQuery.data?.approvalsPending ?? 0)} />
          <SummaryCard label="Write-Offs" value={String(summaryQuery.data?.writeOffPending ?? 0)} />
        </div>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-2 border-b p-2">
            <Input
              value={query}
              onChange={(event) => void setSearchParams({ query: event.target.value })}
              placeholder="Search batch id, submitter, idempotency key..."
              className="h-8 text-xs"
            />
            <Select
              value={status ?? "all"}
              items={statusChoices}
              onValueChange={(value) => void setSearchParams({ status: value === "all" ? null : value })}
            >
              <SelectTrigger className="h-8 text-xs">
                <SelectValue placeholder="All statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                {statusChoices.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <ScrollArea className="flex-1">
            <div className="flex flex-col gap-2 p-2">
              {listQuery.isLoading
                ? Array.from({ length: 6 }).map((_, index) => (
                    <Skeleton key={index} className="h-24 w-full rounded-xl" />
                  ))
                : null}
              {!listQuery.isLoading && listQuery.data?.results.length === 0 ? (
                <div className="flex h-full items-center justify-center">
                  <EmptyState
                    title="No batches found"
                    description="Batch submissions will appear here when operators submit multi-invoice adjustments."
                    icons={[Layers3Icon, CheckCircle2Icon, AlertTriangleIcon]}
                    className="border-none shadow-none"
                  />
                </div>
              ) : null}
              {listQuery.data?.results.map((row) => (
                <button
                  key={row.batchId}
                  type="button"
                  onClick={() => void setSearchParams({ item: row.batchId })}
                  className={[
                    "rounded-xl border p-3 text-left transition-colors",
                    row.batchId === selectedRow?.batchId ? "border-primary bg-primary/5" : "hover:bg-muted/40",
                  ].join(" ")}
                >
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">{row.batchId}</p>
                      <p className="text-xs text-muted-foreground">{row.submittedByName || row.submittedById || "System"}</p>
                    </div>
                    <span className="rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.16em]">
                      {row.status}
                    </span>
                  </div>
                  <div className="mt-3 grid grid-cols-4 gap-2 text-xs">
                    <Metric label="Total" value={String(row.totalCount)} />
                    <Metric label="Done" value={String(row.processedCount)} />
                    <Metric label="Failed" value={String(row.failedCount)} />
                    <Metric label="Pending" value={String(row.pendingCount)} />
                  </div>
                  {row.lastFailure ? (
                    <p className="mt-3 line-clamp-2 text-xs text-destructive">{row.lastFailure}</p>
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
            <div className="flex h-full items-center justify-center p-6">
              <EmptyState
                title="No batch selected"
                description="Select a batch to inspect item-level execution results and linked artifacts."
                icons={[Layers3Icon, CheckCircle2Icon, AlertTriangleIcon]}
                className="border-none shadow-none"
              />
            </div>
          ) : detailQuery.isLoading || !detailQuery.data ? (
            <div className="space-y-4 p-4">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-64 w-full" />
            </div>
          ) : (
            <div className="space-y-4 p-4">
              <Card>
                <CardHeader className="border-b">
                  <CardTitle>{selectedRow.batchId}</CardTitle>
                  <CardDescription>{selectedRow.submittedByName || selectedRow.submittedById || "System batch"}</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 pt-4 md:grid-cols-4">
                  <Metric label="Total" value={String(selectedRow.totalCount)} />
                  <Metric label="Succeeded" value={String(selectedRow.succeededCount)} />
                  <Metric label="Failed" value={String(selectedRow.failedCount)} />
                  <Metric label="Pending" value={String(selectedRow.pendingCount)} />
                  <Metric label="Submitted At" value={formatTimestamp(selectedRow.submittedAt)} />
                  <Metric label="Status" value={selectedRow.status} />
                  <Metric label="Last Failure Count" value={String(selectedRow.lastFailureCount)} />
                  <Metric label="Idempotency Key" value={selectedRow.idempotencyKey} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="border-b">
                  <CardTitle>Item Results</CardTitle>
                  <CardDescription>Per-item outcome, failure reason, and created artifacts.</CardDescription>
                </CardHeader>
                <CardContent className="pt-4">
                  <div className="overflow-hidden rounded-xl border">
                    <table className="w-full text-sm">
                      <thead className="bg-muted/40 text-left text-muted-foreground">
                        <tr>
                          <th className="px-4 py-3">Invoice</th>
                          <th className="px-4 py-3">Status</th>
                          <th className="px-4 py-3">Failure</th>
                          <th className="px-4 py-3">Artifacts</th>
                        </tr>
                      </thead>
                      <tbody>
                        {detailQuery.data.items.map((item) => (
                          <tr key={item.id} className="border-t align-top">
                            <td className="px-4 py-3 font-mono text-xs">{item.invoiceId}</td>
                            <td className="px-4 py-3">{item.status}</td>
                            <td className="px-4 py-3 text-xs text-muted-foreground">
                              {item.errorMessage || "No failure recorded"}
                            </td>
                            <td className="px-4 py-3">
                              <div className="flex flex-wrap gap-2">
                                <LinkButton to={`/billing/invoices?item=${item.invoiceId}`}>Invoice</LinkButton>
                                {item.adjustmentId ? (
                                  <LinkButton to={`/billing/pending-approvals?item=${item.adjustmentId}`}>
                                    Adjustment
                                  </LinkButton>
                                ) : null}
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </ScrollArea>
      }
    />
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className="shadow-none">
      <CardContent className="px-4 py-3">
        <p className="text-[11px] uppercase tracking-[0.16em] text-muted-foreground">{label}</p>
        <p className="mt-1 text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-background px-3 py-2">
      <p className="text-[10px] uppercase tracking-[0.16em] text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm font-medium break-all">{value}</p>
    </div>
  );
}

function LinkButton({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link to={to} className="inline-flex items-center gap-1 rounded-full border px-3 py-1.5 text-xs hover:bg-muted">
      <ExternalLinkIcon className="size-3.5" />
      {children}
    </Link>
  );
}

function formatTimestamp(value: number | null | undefined) {
  if (!value) {
    return "Not recorded";
  }
  return new Date(value * 1000).toLocaleString();
}
