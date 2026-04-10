import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { EmptyState } from "@/components/empty-state";
import { TextareaField } from "@/components/fields/textarea-field";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  InvoiceAdjustment,
  InvoiceAdjustmentKind,
  InvoiceApprovalQueueItem,
} from "@/types/invoice-adjustment";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  ClipboardListIcon,
  ExternalLinkIcon,
  GitBranchPlusIcon,
  ReceiptTextIcon,
  SearchIcon,
  XIcon,
} from "lucide-react";
import { useQueryStates } from "nuqs";
import { type ReactNode, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { invoiceApprovalSearchParamsParser } from "./use-invoice-approval-state";

const adjustmentKindChoices: Array<{ label: string; value: InvoiceAdjustmentKind }> = [
  { label: "Credit Only", value: "CreditOnly" },
  { label: "Credit & Rebill", value: "CreditAndRebill" },
  { label: "Full Reversal", value: "FullReversal" },
  { label: "Write-Off", value: "WriteOff" },
];

const KIND_LABELS: Record<string, string> = {
  CreditOnly: "Credit Only",
  CreditAndRebill: "Credit & Rebill",
  FullReversal: "Full Reversal",
  WriteOff: "Write-Off",
};

export function InvoiceApprovalPage() {
  const [searchParams, setSearchParams] = useQueryStates(invoiceApprovalSearchParamsParser);
  const { item: selectedAdjustmentId, query, kind } = searchParams;
  const deferredQuery = useDeferredValue(query);
  const queryClient = useQueryClient();
  const [showRejectForm, setShowRejectForm] = useState(false);
  const rejectForm = useForm<{ rejectReason: string }>({
    defaultValues: {
      rejectReason: "",
    },
  });

  const observerTarget = useRef<HTMLDivElement>(null);

  const {
    data: listData,
    isLoading,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useInfiniteQuery({
    queryKey: ["invoice-adjustment-approvals", deferredQuery, kind],
    queryFn: async ({ pageParam }) => {
      const params = new URLSearchParams({
        limit: "20",
        offset: String(pageParam),
      });
      if (deferredQuery.trim()) {
        params.set("query", deferredQuery.trim());
      }
      if (kind) {
        params.set(
          "fieldFilters",
          JSON.stringify([{ field: "kind", operator: "eq", value: kind }]),
        );
      }
      return apiService.invoiceAdjustmentService.listApprovals(
        Object.fromEntries(params.entries()),
      );
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.next || lastPage.results.length === 20) {
        return lastPageParam + 20;
      }
      return undefined;
    },
  });

  const allRows = useMemo(
    () => listData?.pages.flatMap((page) => page.results) ?? [],
    [listData?.pages],
  );

  const selectedRow =
    allRows.find((row) => row.adjustmentId === selectedAdjustmentId) ?? allRows[0] ?? null;

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const detailQuery = useQuery({
    ...queries["invoice-adjustment"].get(selectedRow?.adjustmentId ?? ""),
    enabled: Boolean(selectedRow?.adjustmentId),
  });

  const summaryQuery = useQuery({
    ...queries["invoice-adjustment"].summary(),
  });

  const approveMutation = useMutation({
    mutationFn: async (adjustmentId: string) =>
      apiService.invoiceAdjustmentService.approve(adjustmentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["invoice-adjustment"] });
      void queryClient.invalidateQueries({ queryKey: ["invoice"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      toast.success("Adjustment approved");
      setShowRejectForm(false);
    },
    onError: () => toast.error("Failed to approve invoice adjustment"),
  });

  const rejectMutation = useApiMutation({
    setFormError: rejectForm.setError,
    resourceName: "invoice adjustment reject",
    mutationFn: async ({
      adjustmentId,
      rejectReason,
    }: {
      adjustmentId: InvoiceAdjustment["id"];
      rejectReason: InvoiceAdjustment["rejectionReason"];
    }) => apiService.invoiceAdjustmentService.reject(adjustmentId, rejectReason),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["invoice-adjustment"] });
      rejectForm.reset();
      setShowRejectForm(false);
      toast.success("Adjustment rejected");
    },
  });

  const handleReject = ({ rejectReason }: { rejectReason: string }) => {
    if (!selectedRow) return;
    rejectMutation.mutate({
      adjustmentId: selectedRow.adjustmentId,
      rejectReason: rejectReason.trim(),
    });
  };

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        title: "Pending Approvals",
        description:
          "Review policy-controlled invoice adjustments awaiting finance approval.",
      }}
      toolbar={
        <div className="mx-4 mt-3 grid gap-2.5 md:grid-cols-4">
          <SummaryCard
            label="Pending Approvals"
            value={String(summaryQuery.data?.approvalsPending ?? 0)}
          />
          <SummaryCard
            label="Reconciliation"
            value={String(summaryQuery.data?.reconciliationPending ?? 0)}
          />
          <SummaryCard
            label="Write-Offs"
            value={String(summaryQuery.data?.writeOffPending ?? 0)}
          />
          <SummaryCard
            label="Batch Failures"
            value={String(summaryQuery.data?.failedBatchItems ?? 0)}
          />
        </div>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-1.5 border-b p-2">
            <Input
              value={query}
              onChange={(event) => void setSearchParams({ query: event.target.value })}
              placeholder="Search invoice, customer, reason..."
              leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
              className="h-7 text-xs"
            />
            <Select
              value={kind ?? "all"}
              items={adjustmentKindChoices}
              onValueChange={(value) =>
                void setSearchParams({ kind: value === "all" ? null : value })
              }
            >
              <SelectTrigger className="h-7 text-xs">
                <SelectValue placeholder="All adjustment types" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All adjustment types</SelectItem>
                {adjustmentKindChoices.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <ScrollArea className="flex-1">
            <div
              className={cn(
                "flex flex-col gap-1.5 p-2",
                !isLoading && allRows.length === 0 && "h-full p-0",
              )}
            >
              {isLoading
                ? Array.from({ length: 6 }).map((_, index) => (
                    <Skeleton key={index} className="h-20 w-full rounded-lg" />
                  ))
                : null}
              {!isLoading && allRows.length === 0 ? (
                <div className="flex h-full items-center justify-center">
                  <EmptyState
                    title="No pending approvals"
                    description="Adjust the filters or wait for new submitted adjustments."
                    icons={[ReceiptTextIcon, ClipboardListIcon, GitBranchPlusIcon]}
                    className="flex h-full max-w-none flex-col items-center justify-center rounded-none border-none p-6 shadow-none"
                  />
                </div>
              ) : null}
              {allRows.map((row) => {
                const isSelected = row.adjustmentId === selectedRow?.adjustmentId;
                const netDelta = Number(row.netDeltaAmount);
                return (
                  <button
                    key={row.adjustmentId}
                    type="button"
                    onClick={() => {
                      void setSearchParams({ item: row.adjustmentId });
                      setShowRejectForm(false);
                      rejectForm.reset();
                    }}
                    className={cn(
                      "rounded-lg border p-2.5 text-left transition-colors",
                      isSelected ? "border-primary bg-primary/5" : "hover:bg-muted/40",
                    )}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate text-xs font-medium">
                          {row.originalInvoiceNumber}
                        </p>
                        <p className="truncate text-2xs text-muted-foreground">
                          {row.customerName}
                        </p>
                      </div>
                      <p className="shrink-0 text-xs font-semibold tabular-nums">
                        {formatCurrency(netDelta)}
                      </p>
                    </div>
                    <div className="mt-1.5 flex items-center gap-1.5">
                      <Badge variant="secondary">{KIND_LABELS[row.kind] ?? row.kind}</Badge>
                      <span className="text-2xs text-muted-foreground">
                        {row.submittedByName || "Unknown"}
                      </span>
                    </div>
                    {row.reason ? (
                      <p className="mt-1.5 line-clamp-1 text-2xs text-muted-foreground">
                        {row.reason}
                      </p>
                    ) : null}
                  </button>
                );
              })}
              {isFetchingNextPage ? (
                <div className="flex items-center justify-center py-4">
                  <TextShimmer className="font-mono text-sm" duration={1}>
                    Loading more...
                  </TextShimmer>
                </div>
              ) : null}
              <div ref={observerTarget} className="h-px" />
            </div>
          </ScrollArea>
        </div>
      }
      detail={
        <ScrollArea className="h-full">
          {!selectedRow ? (
            <div className="flex h-full items-center justify-center p-6">
              <EmptyState
                title="No approval selected"
                description="Select a submitted adjustment to review policy context and approve or reject it."
                icons={[GitBranchPlusIcon, ReceiptTextIcon, ClipboardListIcon]}
                className="max-w-xl border-none p-8 shadow-none"
              />
            </div>
          ) : detailQuery.isLoading || !detailQuery.data ? (
            <div className="space-y-4 p-4">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-64 w-full" />
            </div>
          ) : (
            <ApprovalDetail
              selectedRow={selectedRow}
              detail={detailQuery.data}
              showRejectForm={showRejectForm}
              setShowRejectForm={setShowRejectForm}
              rejectForm={rejectForm}
              approveMutation={approveMutation}
              rejectMutation={rejectMutation}
              handleReject={handleReject}
            />
          )}
        </ScrollArea>
      }
    />
  );
}

function ApprovalDetail({
  selectedRow,
  detail,
  showRejectForm,
  setShowRejectForm,
  rejectForm,
  approveMutation,
  rejectMutation,
  handleReject,
}: {
  selectedRow: InvoiceApprovalQueueItem;
  detail: InvoiceAdjustment;
  showRejectForm: boolean;
  setShowRejectForm: (v: boolean) => void;
  rejectForm: ReturnType<typeof useForm<{ rejectReason: string }>>;
  approveMutation: ReturnType<typeof useMutation<unknown, Error, string>>;
  rejectMutation: { isPending: boolean };
  handleReject: (values: { rejectReason: string }) => void;
}) {
  const netDelta = Number(selectedRow.netDeltaAmount);

  return (
    <div className="flex flex-col gap-5 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">{selectedRow.originalInvoiceNumber}</h2>
          <p className="text-sm text-muted-foreground">{selectedRow.customerName}</p>
        </div>
        <p className="text-2xl font-bold tabular-nums">
          {formatCurrency(netDelta)}
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="warning">Pending Approval</Badge>
        <Badge variant="secondary">{KIND_LABELS[selectedRow.kind] ?? selectedRow.kind}</Badge>
      </div>

      <div className="grid gap-5 xl:grid-cols-2">
        <div className="flex flex-col gap-5">
          <div className="rounded-lg border bg-card p-3">
            <SectionLabel>Financial Impact</SectionLabel>
            <div className="mt-2 space-y-2">
              <ChargeSummaryRow
                label="Credit"
                value={formatCurrency(Number(selectedRow.creditTotalAmount))}
              />
              <ChargeSummaryRow
                label="Rebill"
                value={formatCurrency(Number(selectedRow.rebillTotalAmount))}
              />
              <Separator />
              <ChargeSummaryRow
                label="Net Delta"
                value={formatCurrency(netDelta)}
                bold
              />
            </div>
          </div>

          {selectedRow.reason || selectedRow.policyReason ? (
            <div className="rounded-lg border bg-card p-3">
              <SectionLabel>Reason</SectionLabel>
              {selectedRow.reason ? (
                <p className="mt-1.5 border-l-2 border-muted-foreground/20 pl-2.5 text-xs text-muted-foreground italic">
                  {selectedRow.reason}
                </p>
              ) : null}
              {selectedRow.policyReason ? (
                <p className="mt-1.5 text-2xs text-muted-foreground">
                  Policy: {selectedRow.policyReason}
                </p>
              ) : null}
            </div>
          ) : null}

          <div className="rounded-lg border bg-card p-3">
            <SectionLabel>Context</SectionLabel>
            <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
              <PropertyCell label="Requested By">
                <span className="text-xs font-medium">
                  {selectedRow.submittedByName || "Unknown"}
                </span>
              </PropertyCell>
              <PropertyCell label="Submitted">
                <span className="text-xs font-medium">
                  {formatTimestamp(selectedRow.submittedAt)}
                </span>
              </PropertyCell>
              <PropertyCell label="Policy Source">
                <span className="text-xs font-medium">
                  {selectedRow.policySource || "Policy-controlled"}
                </span>
              </PropertyCell>
              <PropertyCell label="Invoice Status">
                <span className="text-xs font-medium">
                  {selectedRow.originalInvoiceStatus}
                </span>
              </PropertyCell>
            </div>
            {selectedRow.requiresReconciliationException ||
            selectedRow.requiresReplacementInvoiceReview ||
            selectedRow.wouldCreateUnappliedCredit ? (
              <div className="mt-2.5 flex flex-wrap gap-1">
                {selectedRow.requiresReconciliationException ? (
                  <Badge variant="warning">Reconciliation Exception</Badge>
                ) : null}
                {selectedRow.requiresReplacementInvoiceReview ? (
                  <Badge variant="info">Replacement Review</Badge>
                ) : null}
                {selectedRow.wouldCreateUnappliedCredit ? (
                  <Badge variant="orange">Unapplied Credit</Badge>
                ) : null}
              </div>
            ) : null}
          </div>

          <div className="rounded-lg border bg-card p-3">
            <SectionLabel>Linked Artifacts</SectionLabel>
            <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-2xs">
              <ArtifactLink
                to={`/billing/invoices?item=${selectedRow.originalInvoiceId}`}
                label="Original Invoice"
              />
              {selectedRow.creditMemoInvoiceId ? (
                <ArtifactLink
                  to={`/billing/invoices?item=${selectedRow.creditMemoInvoiceId}`}
                  label="Credit Memo"
                />
              ) : null}
              {selectedRow.replacementInvoiceId ? (
                <ArtifactLink
                  to={`/billing/invoices?item=${selectedRow.replacementInvoiceId}`}
                  label="Replacement"
                />
              ) : null}
              {selectedRow.rebillQueueItemId ? (
                <ArtifactLink
                  to={`/billing/queue?item=${selectedRow.rebillQueueItemId}&includePosted=true`}
                  label="Rebill Queue"
                />
              ) : null}
              {selectedRow.batchId ? (
                <ArtifactLink
                  to={`/billing/adjustment-batches?item=${selectedRow.batchId}`}
                  label="Batch"
                />
              ) : null}
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-5">
          <div className="rounded-lg border bg-card p-3">
            <SectionLabel>Charge Detail</SectionLabel>
            <div className="mt-2 overflow-hidden rounded-md border">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-left text-muted-foreground">
                  <tr>
                    <th className="px-3 py-2 text-xs font-medium">Line</th>
                    <th className="px-3 py-2 text-xs font-medium">Description</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">Credit</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">Rebill</th>
                  </tr>
                </thead>
                <tbody>
                  {detail.lines.map((line) => (
                    <tr
                      key={line.id}
                      className="border-t transition-colors hover:bg-muted/50"
                    >
                      <td className="px-3 py-2 font-mono text-2xs">{line.lineNumber}</td>
                      <td className="px-3 py-2 text-xs">{line.description}</td>
                      <td className="px-3 py-2 text-right text-xs tabular-nums">
                        {formatCurrency(Number(line.creditAmount))}
                      </td>
                      <td className="px-3 py-2 text-right text-xs tabular-nums">
                        {formatCurrency(Number(line.rebillAmount))}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="rounded-lg border bg-card p-3">
            <SectionLabel>Decision</SectionLabel>
            <div className="mt-2">
              {!showRejectForm ? (
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    type="button"
                    className="bg-green-600 text-white hover:bg-green-700"
                    onClick={() => approveMutation.mutate(selectedRow.adjustmentId)}
                    disabled={approveMutation.isPending}
                  >
                    <CheckIcon className="size-3.5" />
                    Approve
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    type="button"
                    onClick={() => setShowRejectForm(true)}
                  >
                    <XIcon className="size-3.5" />
                    Reject
                  </Button>
                </div>
              ) : (
                <Form onSubmit={rejectForm.handleSubmit(handleReject)}>
                  <div className="space-y-2">
                    <TextareaField
                      control={rejectForm.control}
                      name="rejectReason"
                      label="Rejection Reason"
                      placeholder="Why is this adjustment being rejected?"
                      minRows={3}
                    />
                    <div className="flex items-center gap-2">
                      <Button
                        size="sm"
                        variant="destructive"
                        type="submit"
                        disabled={rejectMutation.isPending}
                      >
                        Confirm Rejection
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        type="button"
                        onClick={() => {
                          setShowRejectForm(false);
                          rejectForm.reset();
                        }}
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>
                </Form>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-card px-3 py-2.5">
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function SectionLabel({ children }: { children: ReactNode }) {
  return <p className="text-xs font-medium text-muted-foreground">{children}</p>;
}

function PropertyCell({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <p className="text-2xs text-muted-foreground">{label}</p>
      {children}
    </div>
  );
}

function ChargeSummaryRow({
  label,
  value,
  bold = false,
}: {
  label: string;
  value: string;
  bold?: boolean;
}) {
  return (
    <div className="flex items-center justify-between">
      <span
        className={cn("text-sm", bold ? "font-medium text-foreground" : "text-muted-foreground")}
      >
        {label}
      </span>
      <span
        className={cn(
          "tabular-nums tracking-tight",
          bold ? "text-base font-semibold text-foreground" : "text-sm text-muted-foreground",
        )}
      >
        {value}
      </span>
    </div>
  );
}

function ArtifactLink({ to, label }: { to: string; label: string }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center gap-0.5 text-muted-foreground hover:text-foreground hover:underline"
    >
      {label}
      <ExternalLinkIcon className="size-2.5" />
    </Link>
  );
}

function formatTimestamp(value: number | null | undefined) {
  if (!value) return "Not recorded";
  return new Date(value * 1000).toLocaleString();
}
