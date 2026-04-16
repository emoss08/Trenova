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
import { formatCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, ExternalLinkIcon, ReceiptTextIcon, WalletCardsIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { type ReactNode, useDeferredValue, useMemo } from "react";
import { Link } from "react-router";
import { invoiceReconciliationSearchParamsParser } from "./use-invoice-reconciliation-state";

const statusChoices = [
  { label: "Open", value: "Open" },
  { label: "Resolved", value: "Resolved" },
];

export function InvoiceReconciliationPage() {
  const [searchParams, setSearchParams] = useQueryStates(invoiceReconciliationSearchParamsParser);
  const { item: selectedExceptionId, query, status } = searchParams;
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
    ...queries["invoice-adjustment"].reconciliation(params),
  });

  const summaryQuery = useQuery({
    ...queries["invoice-adjustment"].summary(),
  });

  const selectedRow =
    listQuery.data?.results.find((row) => row.exceptionId === selectedExceptionId) ??
    listQuery.data?.results[0] ??
    null;

  const detailQuery = useQuery({
    ...queries["invoice-adjustment"].get(selectedRow?.adjustmentId ?? ""),
    enabled: Boolean(selectedRow?.adjustmentId),
  });

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        title: "Reconciliation Exceptions",
        description: "Investigate adjustment-created finance exceptions and trace them back to source artifacts.",
      }}
      toolbar={
        <div className="mx-4 mt-3 grid gap-3 md:grid-cols-4">
          <SummaryCard label="Open Exceptions" value={String(summaryQuery.data?.reconciliationPending ?? 0)} />
          <SummaryCard label="Pending Approvals" value={String(summaryQuery.data?.approvalsPending ?? 0)} />
          <SummaryCard label="Write-Offs" value={String(summaryQuery.data?.writeOffPending ?? 0)} />
          <SummaryCard label="Batches In Flight" value={String(summaryQuery.data?.batchesInFlight ?? 0)} />
        </div>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-2 border-b p-2">
            <Input
              value={query}
              onChange={(event) => void setSearchParams({ query: event.target.value })}
              placeholder="Search invoice, customer, exception reason..."
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
                    title="No reconciliation exceptions"
                    description="Open issues will appear here when adjustments need finance follow-up."
                    icons={[AlertTriangleIcon, WalletCardsIcon, ReceiptTextIcon]}
                    className="border-none shadow-none"
                  />
                </div>
              ) : null}
              {listQuery.data?.results.map((row) => (
                <button
                  key={row.exceptionId}
                  type="button"
                  onClick={() => void setSearchParams({ item: row.exceptionId })}
                  className={[
                    "rounded-xl border p-3 text-left transition-colors",
                    row.exceptionId === selectedRow?.exceptionId ? "border-primary bg-primary/5" : "hover:bg-muted/40",
                  ].join(" ")}
                >
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">{row.originalInvoiceNumber}</p>
                      <p className="text-xs text-muted-foreground">{row.customerName}</p>
                    </div>
                    <span className="rounded-full border px-2 py-0.5 text-[10px] tracking-[0.16em] uppercase">
                      {row.status}
                    </span>
                  </div>
                  <p className="mt-3 text-sm">{row.reason}</p>
                  <p className="mt-2 text-xs text-muted-foreground">
                    Amount: {formatCurrency(Number(row.amount))}
                  </p>
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
                title="No exception selected"
                description="Select a reconciliation exception to inspect linked adjustment and invoice artifacts."
                icons={[AlertTriangleIcon, ReceiptTextIcon, WalletCardsIcon]}
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
                  <CardTitle>{selectedRow.reason}</CardTitle>
                  <CardDescription>{selectedRow.customerName}</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 pt-4 md:grid-cols-2">
                  <Metric label="Status" value={selectedRow.status} />
                  <Metric label="Amount" value={formatCurrency(Number(selectedRow.amount))} />
                  <Metric label="Adjustment Kind" value={selectedRow.adjustmentKind} />
                  <Metric label="Adjustment Status" value={selectedRow.adjustmentStatus} />
                  <Metric label="Requested By" value={selectedRow.submittedByName || selectedRow.submittedById || "Unknown"} />
                  <Metric label="Submitted At" value={formatTimestamp(selectedRow.submittedAt)} />
                  <Metric label="Policy Source" value={selectedRow.policySource || "Policy-controlled"} />
                  <Metric label="Finance Notes" value={selectedRow.financeNotes || "No finance notes recorded"} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="border-b">
                  <CardTitle>Linked Artifacts</CardTitle>
                  <CardDescription>Jump directly into the related billing and invoice surfaces.</CardDescription>
                </CardHeader>
                <CardContent className="flex flex-wrap gap-2 pt-4">
                  <LinkButton to={`/billing/invoices?item=${selectedRow.originalInvoiceId}`}>
                    Original Invoice
                  </LinkButton>
                  {selectedRow.creditMemoInvoiceId ? (
                    <LinkButton to={`/billing/invoices?item=${selectedRow.creditMemoInvoiceId}`}>
                      Credit Memo
                    </LinkButton>
                  ) : null}
                  {selectedRow.replacementInvoiceId ? (
                    <LinkButton to={`/billing/invoices?item=${selectedRow.replacementInvoiceId}`}>
                      Replacement Invoice
                    </LinkButton>
                  ) : null}
                  {selectedRow.rebillQueueItemId ? (
                    <LinkButton to={`/billing/queue?item=${selectedRow.rebillQueueItemId}&includePosted=true`}>
                      Rebill Queue Item
                    </LinkButton>
                  ) : null}
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="border-b">
                  <CardTitle>Adjustment Detail</CardTitle>
                  <CardDescription>Line-level credit and rebill values that created the exception.</CardDescription>
                </CardHeader>
                <CardContent className="pt-4">
                  <div className="overflow-hidden rounded-xl border">
                    <table className="w-full text-sm">
                      <thead className="bg-muted/40 text-left text-muted-foreground">
                        <tr>
                          <th className="px-4 py-3">Line</th>
                          <th className="px-4 py-3">Description</th>
                          <th className="px-4 py-3 text-right">Credit</th>
                          <th className="px-4 py-3 text-right">Rebill</th>
                        </tr>
                      </thead>
                      <tbody>
                        {detailQuery.data.lines.map((line) => (
                          <tr key={line.id} className="border-t">
                            <td className="px-4 py-3 font-mono text-xs">{line.lineNumber}</td>
                            <td className="px-4 py-3">{line.description}</td>
                            <td className="px-4 py-3 text-right">{formatCurrency(Number(line.creditAmount))}</td>
                            <td className="px-4 py-3 text-right">{formatCurrency(Number(line.rebillAmount))}</td>
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
        <p className="text-[11px] tracking-[0.16em] text-muted-foreground uppercase">{label}</p>
        <p className="mt-1 text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-background px-3 py-2">
      <p className="text-[10px] tracking-[0.16em] text-muted-foreground uppercase">{label}</p>
      <p className="mt-1 text-sm font-medium">{value}</p>
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
