import { InputField } from "@/components/fields/input-field";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Form } from "@/components/ui/form";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatCurrency, upperFirst } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { Invoice } from "@/types/invoice";
import type {
  InvoiceAdjustment,
  InvoiceAdjustmentLineage,
  InvoiceAdjustmentStatus,
} from "@/types/invoice-adjustment";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  ChevronDownIcon,
  ExternalLinkIcon,
  XIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";

const STATUS_BADGE_VARIANT: Record<InvoiceAdjustmentStatus, string> = {
  Draft: "secondary",
  PendingApproval: "warning",
  Approved: "active",
  Rejected: "inactive",
  Executing: "info",
  Executed: "active",
  ExecutionFailed: "inactive",
};

export function InvoiceAdjustmentRuntimeSection({
  invoice,
  correctionSummary,
  latestAdjustment,
  latestAdjustmentDetail,
}: {
  invoice: Invoice;
  correctionSummary: InvoiceAdjustmentLineage | undefined;
  latestAdjustment: InvoiceAdjustment | null;
  latestAdjustmentDetail: InvoiceAdjustment | null | undefined;
}) {
  const sortedAdjustments = useMemo(
    () =>
      correctionSummary
        ? [...correctionSummary.adjustments].sort((left, right) => right.createdAt - left.createdAt)
        : [],
    [correctionSummary],
  );
  const sortedInvoices = useMemo(
    () =>
      correctionSummary
        ? [...correctionSummary.invoices].sort((left, right) => left.createdAt - right.createdAt)
        : [],
    [correctionSummary],
  );

  if (!latestAdjustment && !correctionSummary?.adjustments.length) {
    return (
      <div className="rounded-lg border border-dashed p-3">
        <p className="text-xs font-medium text-muted-foreground">Invoice Adjustments</p>
        <p className="mt-0.5 text-2xs text-muted-foreground">
          No invoice adjustments have been created for this invoice yet.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {latestAdjustment ? (
        <div className="space-y-1.5">
          <p className="text-xs font-medium text-muted-foreground">Latest Adjustment</p>
          <InvoiceAdjustmentLatestCard
            invoice={invoice}
            latestAdjustment={latestAdjustment}
            latestAdjustmentDetail={latestAdjustmentDetail}
          />
        </div>
      ) : null}
      {correctionSummary ? (
        <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
          <div className="space-y-1.5">
            <p className="text-xs font-medium text-muted-foreground">Invoice Lineage</p>
            <ScrollArea className="h-[260px]">
              <div className="space-y-1.5">
                {sortedInvoices.map((lineageInvoice) => {
                  const isCurrent =
                    correctionSummary.correctionGroup.currentInvoiceId === lineageInvoice.id;
                  return (
                    <div
                      key={lineageInvoice.id}
                      className="flex items-center justify-between gap-2 rounded-md border bg-background px-3 py-2"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-xs font-medium">{lineageInvoice.number}</p>
                        <p className="text-2xs text-muted-foreground">
                          {lineageInvoice.billType} · {lineageInvoice.status}
                        </p>
                      </div>
                      <Badge variant={isCurrent ? "active" : "secondary"} className="shrink-0">
                        {isCurrent ? "Current" : "Historical"}
                      </Badge>
                    </div>
                  );
                })}
              </div>
            </ScrollArea>
          </div>
          <div className="space-y-1.5">
            <p className="text-xs font-medium text-muted-foreground">Adjustment History</p>
            <ScrollArea className="h-[260px]">
              <div className="space-y-1.5">
                {sortedAdjustments.map((adjustment) => (
                  <div key={adjustment.id} className="rounded-md border bg-background px-3 py-2">
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5">
                          <p className="truncate text-xs font-medium">
                            {formatKind(adjustment.kind)}
                          </p>
                          <Badge
                            variant={
                              STATUS_BADGE_VARIANT[adjustment.status] as
                                | "active"
                                | "inactive"
                                | "warning"
                                | "info"
                                | "secondary"
                            }
                          >
                            {formatStatus(adjustment.status)}
                          </Badge>
                        </div>
                        <p className="mt-0.5 truncate text-2xs text-muted-foreground">
                          {adjustment.reason || adjustment.policyReason || "No note"}
                        </p>
                      </div>
                      <p className="shrink-0 text-2xs text-muted-foreground">
                        {new Date(adjustment.createdAt * 1000).toLocaleDateString()}
                      </p>
                    </div>
                    {adjustment.status === "ExecutionFailed" ? (
                      <div className="mt-1.5">
                        <ExecutionFailureCollapsible executionError={adjustment.executionError} />
                      </div>
                    ) : null}
                  </div>
                ))}
              </div>
            </ScrollArea>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function InvoiceAdjustmentLatestCard({
  invoice,
  latestAdjustment,
  latestAdjustmentDetail,
}: {
  invoice: Invoice;
  latestAdjustment: InvoiceAdjustment;
  latestAdjustmentDetail?: InvoiceAdjustment | null;
}) {
  const queryClient = useQueryClient();
  const [showRejectForm, setShowRejectForm] = useState(false);
  const approvalForm = useForm({
    defaultValues: {
      rejectReason: "",
    },
  });

  const approveMutation = useMutation({
    mutationFn: async (adjustmentId: InvoiceAdjustment["id"]) =>
      apiService.invoiceAdjustmentService.approve(adjustmentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["invoice"] });
      void queryClient.invalidateQueries({ queryKey: ["invoice-adjustment"] });
      toast.success("Adjustment approved");
    },
    onError: () => toast.error("Failed to approve invoice adjustment"),
  });

  const rejectMutation = useApiMutation({
    setFormError: approvalForm.setError,
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
      toast.success("Adjustment rejected");
      approvalForm.reset();
      setShowRejectForm(false);
    },
  });

  const handleReject = async ({ rejectReason }: { rejectReason: string }) => {
    await rejectMutation.mutateAsync({
      adjustmentId: latestAdjustment.id,
      rejectReason: rejectReason.trim(),
    });
  };

  const netDelta = Number(latestAdjustment.netDeltaAmount);
  const reason = latestAdjustment.reason || latestAdjustment.policyReason;
  const evidenceDocs =
    latestAdjustmentDetail?.referencedDocuments.map((r) => r.snapshotOriginalName) ?? [];
  const adjustmentDocs =
    latestAdjustmentDetail?.adjustmentDocuments.map((d) => d.originalName) ?? [];
  const allDocs = [...evidenceDocs, ...adjustmentDocs];

  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5">
          <p className="text-sm font-medium">{formatKind(latestAdjustment.kind)}</p>
          <Badge
            variant={
              STATUS_BADGE_VARIANT[latestAdjustment.status] as
                | "active"
                | "inactive"
                | "warning"
                | "info"
                | "secondary"
            }
          >
            {formatStatus(latestAdjustment.status)}
          </Badge>
        </div>
        <p className="text-sm font-semibold tabular-nums">{formatCurrency(netDelta)}</p>
      </div>

      <p className="mt-1.5 text-2xs text-muted-foreground">
        Credit {formatCurrency(Number(latestAdjustment.creditTotalAmount))} · Rebill{" "}
        {formatCurrency(Number(latestAdjustment.rebillTotalAmount))}
      </p>

      {reason ? (
        <p className="mt-2 border-l-2 border-muted-foreground/20 pl-2.5 text-xs text-muted-foreground italic">
          {reason}
        </p>
      ) : null}

      {latestAdjustment.status === "ExecutionFailed" ? (
        <div className="mt-2.5">
          <ExecutionFailureCollapsible executionError={latestAdjustment.executionError} />
        </div>
      ) : null}

      <div className="mt-2.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-2xs">
        <ArtifactLink to={`/billing/invoices?item=${invoice.id}`} label="Current Invoice" />
        {latestAdjustment.creditMemoInvoiceId ? (
          <ArtifactLink
            to={`/billing/invoices?item=${latestAdjustment.creditMemoInvoiceId}`}
            label="Credit Memo"
          />
        ) : null}
        {latestAdjustment.replacementInvoiceId ? (
          <ArtifactLink
            to={`/billing/invoices?item=${latestAdjustment.replacementInvoiceId}`}
            label="Replacement"
          />
        ) : null}
        {latestAdjustment.rebillQueueItemId ? (
          <ArtifactLink
            to={`/billing/queue?item=${latestAdjustment.rebillQueueItemId}&includePosted=true`}
            label="Rebill Queue"
          />
        ) : null}
        {latestAdjustment.batchId ? (
          <ArtifactLink
            to={`/billing/adjustment-batches?item=${latestAdjustment.batchId}`}
            label="Batch"
          />
        ) : null}
      </div>

      {allDocs.length > 0 ? (
        <Collapsible className="mt-2.5">
          <CollapsibleTrigger className="group flex items-center gap-1 text-2xs font-medium text-muted-foreground hover:text-foreground">
            <span>Documents ({allDocs.length})</span>
            <ChevronDownIcon className="size-3 transition-transform group-data-panel-open:rotate-180" />
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="mt-1.5 flex flex-wrap gap-1">
              {allDocs.map((name) => (
                <span key={name} className="rounded border bg-background px-1.5 py-0.5 text-2xs">
                  {name}
                </span>
              ))}
            </div>
          </CollapsibleContent>
        </Collapsible>
      ) : null}

      {latestAdjustment.status === "PendingApproval" ? (
        <>
          <Separator className="my-3" />
          {!showRejectForm ? (
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                type="button"
                className="bg-green-600 text-white hover:bg-green-700"
                onClick={() => approveMutation.mutate(latestAdjustment.id)}
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
            <Form onSubmit={approvalForm.handleSubmit(handleReject)}>
              <div className="space-y-2">
                <InputField
                  control={approvalForm.control}
                  name="rejectReason"
                  label="Rejection Reason"
                  placeholder="Why is this being rejected?"
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
                      approvalForm.reset();
                    }}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            </Form>
          )}
        </>
      ) : null}
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

function ExecutionFailureCollapsible({
  executionError,
}: {
  executionError: InvoiceAdjustment["executionError"];
}) {
  const errorText = executionError?.trim().length
    ? upperFirst(executionError)
    : "Execution failed, but no detailed reason was recorded.";

  return (
    <Collapsible>
      <CollapsibleTrigger className="group flex items-center gap-1 text-2xs font-medium text-destructive hover:underline">
        <AlertTriangleIcon className="size-3" />
        <span>Execution failed</span>
        <ChevronDownIcon className="size-3 transition-transform group-data-[panel-open]:rotate-180" />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <p className="mt-1.5 rounded border border-destructive/20 bg-destructive/5 px-2.5 py-1.5 text-2xs text-destructive">
          {errorText}
        </p>
      </CollapsibleContent>
    </Collapsible>
  );
}

function formatKind(kind: InvoiceAdjustment["kind"]) {
  switch (kind) {
    case "WriteOff":
      return "Write Off";
    case "CreditOnly":
      return "Credit Only";
    case "CreditAndRebill":
      return "Credit & Rebill";
    case "FullReversal":
      return "Full Reversal";
    default:
      return kind;
  }
}

function formatStatus(status: InvoiceAdjustmentStatus) {
  switch (status) {
    case "PendingApproval":
      return "Pending Approval";
    case "ExecutionFailed":
      return "Failed";
    default:
      return status;
  }
}
