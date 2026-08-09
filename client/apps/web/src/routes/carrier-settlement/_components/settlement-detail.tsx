import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import {
  CarrierInvoiceMatchStatusBadge,
  CarrierSettlementStatusBadge,
  RateConfirmationStatusBadge,
} from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea, type ScrollAreaMaskVariant } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  addCarrierSettlementAdjustment,
  approveCarrierSettlement,
  exportCarrierSettlementBatchCsv,
  fetchCarrierInvoiceMatches,
  fetchCarrierSettlementDetail,
  markCarrierSettlementPaid,
  postCarrierSettlement,
  recalculateCarrierSettlement,
  rejectCarrierSettlement,
  removeCarrierSettlementAdjustment,
  submitCarrierSettlement,
  voidCarrierSettlement,
  type CarrierSettlementDetail as SettlementDetailData,
  type CarrierSettlementLineRow,
} from "@/lib/graphql/carrier-settlement";
import { apiService } from "@/services/api";
import { documentContentUrl } from "@/services/document";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { CarrierSettlementStatus } from "@trenova/shared/types/carrier-settlement";
import type { RateConfirmation } from "@trenova/shared/types/rate-confirmation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowUpRight,
  CheckCheck,
  CircleDollarSign,
  Download,
  FileText,
  Plus,
  RefreshCcw,
  Send,
  X,
  Undo2,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

type ReasonAction = "reject" | "void";

const lineTypeLabels: Record<string, string> = {
  LinehaulCost: "Linehaul Cost",
  FuelSurcharge: "Fuel Surcharge",
  Accessorial: "Accessorials",
  Adjustment: "Manual Adjustments",
};

const lineTypeOrder = ["LinehaulCost", "FuelSurcharge", "Accessorial", "Adjustment"];

const MAX_RATE_CON_MOVES = 8;

function formatDate(unix?: number | null): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

export function CarrierSettlementDetail({
  settlementId,
  onClose,
  scrollMaskVariant = "background",
  readOnly = false,
}: {
  settlementId: string;
  onClose: () => void;
  scrollMaskVariant?: ScrollAreaMaskVariant;
  readOnly?: boolean;
}) {
  const queryClient = useQueryClient();
  const { data: settlement, isLoading } = useQuery({
    queryKey: ["carrier-settlement-detail", settlementId],
    queryFn: () => fetchCarrierSettlementDetail(settlementId),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: ["carrier-settlement-detail", settlementId],
    });
    void queryClient.invalidateQueries({ queryKey: ["carrier-settlement-list"] });
    void queryClient.invalidateQueries({ queryKey: ["carrier-settlement-workspace-summary"] });
    void queryClient.invalidateQueries({
      queryKey: ["carrier-settlement-workspace-settlements"],
    });
  };

  if (isLoading || !settlement) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  return (
    <ScrollArea
      className="h-full min-h-0"
      viewportClassName="min-h-0"
      maskVariant={scrollMaskVariant}
      maskHeight={18}
    >
      <div className="flex size-full flex-col gap-4 p-3">
        <SettlementSummary settlement={settlement} />
        {readOnly ? (
          <ReadOnlyNotice settlement={settlement} />
        ) : (
          <SettlementActions settlement={settlement} onChanged={invalidate} onClose={onClose} />
        )}
        <SettlementLines settlement={settlement} onChanged={invalidate} readOnly={readOnly} />
        <RemittanceCard settlement={settlement} />
        <LinkedRateConfirmations settlement={settlement} />
        <LinkedInvoiceMatches settlement={settlement} />
        <SettlementTimeline settlement={settlement} />
      </div>
    </ScrollArea>
  );
}

function ReadOnlyNotice({ settlement }: { settlement: SettlementDetailData }) {
  const isTerminal = settlement.status === "Paid" || settlement.status === "Voided";

  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border bg-muted/30 px-3 py-2">
      <p className="text-[11px] text-muted-foreground">
        {isTerminal
          ? "This settlement is finalized and shown here for record-keeping."
          : "This is a read-only view — process, adjust, or pay this settlement from the workspace."}
      </p>
      {!isTerminal && (
        <Link
          to={`/carrier-settlements/workspace?settlement=${settlement.id}`}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-7 shrink-0 text-xs")}
        >
          <ArrowUpRight className="size-3.5" />
          Open in Workspace
        </Link>
      )}
    </div>
  );
}

function SettlementSummary({ settlement }: { settlement: SettlementDetailData }) {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <CarrierSettlementStatusBadge status={settlement.status as CarrierSettlementStatus} />
        {settlement.carrier && (
          <span className="text-xs text-muted-foreground">
            {settlement.carrier.name}
            {settlement.carrier.scac ? ` (${settlement.carrier.scac})` : ""}
          </span>
        )}
        <span className="ml-auto text-xs text-muted-foreground">
          {formatDate(settlement.periodStart)} – {formatDate(settlement.periodEnd)} · pays{" "}
          {formatDate(settlement.payDate)}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <SummaryTile label="Gross Cost">
          <AmountDisplay value={settlement.grossCostMinor} currency={settlement.currencyCode} />
        </SummaryTile>
        <SummaryTile label="Adjustments">
          <AmountDisplay
            value={settlement.adjustmentsMinor}
            variant="auto"
            currency={settlement.currencyCode}
          />
        </SummaryTile>
        <SummaryTile label="Loads">
          <span className="tabular-nums">{settlement.shipmentCount}</span>
        </SummaryTile>
        <SummaryTile label="Net Payable" highlight>
          <AmountDisplay
            value={settlement.netPayableMinor}
            variant="positive"
            currency={settlement.currencyCode}
          />
        </SummaryTile>
      </div>
    </div>
  );
}

function SummaryTile({
  label,
  highlight,
  children,
}: {
  label: string;
  highlight?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "rounded-lg border p-3",
        highlight
          ? "border-green-200 bg-green-50/50 dark:border-green-900 dark:bg-green-950/30"
          : "bg-muted/30",
      )}
    >
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <div className="mt-1 text-sm font-semibold">{children}</div>
    </div>
  );
}

function SettlementActions({
  settlement,
  onChanged,
  onClose,
}: {
  settlement: SettlementDetailData;
  onChanged: () => void;
  onClose: () => void;
}) {
  const [reasonAction, setReasonAction] = useState<ReasonAction | null>(null);
  const [payDialogOpen, setPayDialogOpen] = useState(false);
  const [adjustDialogOpen, setAdjustDialogOpen] = useState(false);

  const actionInput = { settlementId: settlement.id };
  const runAction = useMutation({
    mutationFn: async (action: string) => {
      switch (action) {
        case "submit":
          return submitCarrierSettlement(actionInput);
        case "approve":
          return approveCarrierSettlement(actionInput);
        case "post":
          return postCarrierSettlement(actionInput);
        case "recalculate":
          return recalculateCarrierSettlement(actionInput);
        default:
          throw new Error(`Unknown action ${action}`);
      }
    },
    onSuccess: (_data, action) => {
      toast.success(
        {
          submit: "Settlement submitted for approval",
          approve: "Settlement approved",
          post: "Settlement posted to the general ledger",
          recalculate: "Settlement recalculated from current cost events",
        }[action] ?? "Settlement updated",
      );
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Settlement action failed"),
  });

  const status = settlement.status as CarrierSettlementStatus;
  const busy = runAction.isPending;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {status === "Draft" && (
        <>
          <Button size="sm" disabled={busy} onClick={() => runAction.mutate("submit")}>
            <Send className="size-3.5" />
            Submit for Approval
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={busy}
            onClick={() => runAction.mutate("recalculate")}
          >
            <RefreshCcw className="size-3.5" />
            Recalculate
          </Button>
        </>
      )}
      {status === "PendingApproval" && (
        <>
          <Button size="sm" disabled={busy} onClick={() => runAction.mutate("approve")}>
            <CheckCheck className="size-3.5" />
            Approve
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={busy}
            onClick={() => setReasonAction("reject")}
          >
            <Undo2 className="size-3.5" />
            Reject
          </Button>
        </>
      )}
      {status === "Approved" && (
        <Button size="sm" disabled={busy} onClick={() => runAction.mutate("post")}>
          <CheckCheck className="size-3.5" />
          Post to GL
        </Button>
      )}
      {status === "Posted" && (
        <Button size="sm" disabled={busy} onClick={() => setPayDialogOpen(true)}>
          <CircleDollarSign className="size-3.5" />
          Mark Paid
        </Button>
      )}
      {(status === "Draft" || status === "PendingApproval") && (
        <Button
          size="sm"
          variant="outline"
          disabled={busy}
          onClick={() => setAdjustDialogOpen(true)}
        >
          <Plus className="size-3.5" />
          Add Adjustment
        </Button>
      )}
      {settlement.batchId && <BatchCsvExportButton batchId={settlement.batchId} />}
      {status !== "Paid" && status !== "Voided" && (
        <Button
          size="sm"
          variant="ghost"
          className="text-red-600 hover:text-red-700 dark:text-red-400"
          disabled={busy}
          onClick={() => setReasonAction("void")}
        >
          <X className="size-3.5" />
          Void
        </Button>
      )}

      <ReasonDialog
        action={reasonAction}
        settlementId={settlement.id}
        onOpenChange={(open) => !open && setReasonAction(null)}
        onChanged={onChanged}
      />
      <MarkPaidDialog
        open={payDialogOpen}
        settlement={settlement}
        onOpenChange={setPayDialogOpen}
        onChanged={onChanged}
      />
      <AddAdjustmentDialog
        open={adjustDialogOpen}
        settlementId={settlement.id}
        onOpenChange={setAdjustDialogOpen}
        onChanged={onChanged}
      />
      {status === "Voided" && settlement.voidReason && (
        <span className="text-xs text-muted-foreground">Voided: {settlement.voidReason}</span>
      )}
      <span className="sr-only">
        <Button variant="ghost" onClick={onClose}>
          Close
        </Button>
      </span>
    </div>
  );
}

function BatchCsvExportButton({ batchId }: { batchId: string }) {
  const exportMutation = useMutation({
    mutationFn: () => exportCarrierSettlementBatchCsv(batchId),
    onSuccess: (csv) => {
      const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `carrier-settlement-batch-${batchId}.csv`;
      anchor.click();
      URL.revokeObjectURL(url);
      toast.success("Remittance CSV downloaded");
    },
    onError: (error: Error) => toast.error(error.message || "Export failed"),
  });

  return (
    <Button
      size="sm"
      variant="outline"
      disabled={exportMutation.isPending}
      onClick={() => exportMutation.mutate()}
      title="Download the remittance CSV for this settlement's batch"
    >
      <Download className="size-3.5" />
      Batch CSV
    </Button>
  );
}

function ReasonDialog({
  action,
  settlementId,
  onOpenChange,
  onChanged,
}: {
  action: ReasonAction | null;
  settlementId: string;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const [reason, setReason] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      action === "reject"
        ? rejectCarrierSettlement({ settlementId, reason })
        : voidCarrierSettlement({ settlementId, reason }),
    onSuccess: () => {
      toast.success(action === "reject" ? "Settlement rejected" : "Settlement voided");
      setReason("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Action failed"),
  });

  return (
    <Dialog open={action != null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{action === "reject" ? "Reject settlement" : "Void settlement"}</DialogTitle>
          <DialogDescription>
            {action === "reject"
              ? "The settlement will return to draft for corrections."
              : "Voiding releases cost events back to the accrual pool and reverses any GL postings."}
          </DialogDescription>
        </DialogHeader>
        <Textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Reason (required)"
          rows={3}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant={action === "void" ? "destructive" : "default"}
            disabled={!reason.trim() || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {action === "reject" ? "Reject" : "Void"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MarkPaidDialog({
  open,
  settlement,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  settlement: SettlementDetailData;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const [paymentMethod, setPaymentMethod] = useState(() =>
    settlement.carrier?.paymentMethod === "ACHManual" ? "ACHManual" : "Check",
  );
  const [paymentReference, setPaymentReference] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      markCarrierSettlementPaid({
        settlementId: settlement.id,
        paymentMethod,
        paymentReference: paymentReference || undefined,
      }),
    onSuccess: () => {
      toast.success("Settlement marked paid — cash journal and ledger payment recorded");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to mark settlement paid"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Mark settlement paid</DialogTitle>
          <DialogDescription>
            Records the disbursement to the carrier and posts the cash journal (debit accounts
            payable, credit cash) plus the ledger payment entry.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div>
            <p className="mb-1 text-xs font-medium">Payment method</p>
            <p className="mb-1 text-[11px] text-muted-foreground">
              How the payable was remitted — defaults to the carrier&apos;s preferred method.
            </p>
            <div className="flex gap-2">
              {[
                { value: "Check", label: "Check" },
                { value: "ACHManual", label: "ACH (Manual)" },
                { value: "Other", label: "Other" },
              ].map((method) => (
                <Button
                  key={method.value}
                  size="sm"
                  variant={paymentMethod === method.value ? "default" : "outline"}
                  onClick={() => setPaymentMethod(method.value)}
                >
                  {method.label}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <p className="mb-1 text-xs font-medium">Reference</p>
            <Input
              value={paymentReference}
              onChange={(e) => setPaymentReference(e.target.value)}
              placeholder="Check number / ACH trace (optional)"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              The check number or ACH trace so the payment reconciles against the bank.
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={mutation.isPending} onClick={() => mutation.mutate()}>
            Mark Paid
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AddAdjustmentDialog({
  open,
  settlementId,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  settlementId: string;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");

  const parsedAmount = Number(amount);
  const valid = description.trim() !== "" && !Number.isNaN(parsedAmount) && parsedAmount !== 0;

  const mutation = useMutation({
    mutationFn: () =>
      addCarrierSettlementAdjustment({
        settlementId,
        description: description.trim(),
        amountMinor: Math.round(parsedAmount * 100),
      }),
    onSuccess: () => {
      toast.success("Adjustment added");
      setDescription("");
      setAmount("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to add adjustment"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add manual adjustment</DialogTitle>
          <DialogDescription>
            Positive amounts add cost owed to the carrier (detention, lumper); negative amounts
            deduct (damage claim, advance recovery).
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Description (e.g. Detention - Chicago 6/12)"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              Appears as the line item on the carrier&apos;s statement — say what and when.
            </p>
          </div>
          <div>
            <Input
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="Amount (e.g. 150.00 or -75.00)"
              inputMode="decimal"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              Dollars, not cents; positive increases the payable, negative reduces it.
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={!valid || mutation.isPending} onClick={() => mutation.mutate()}>
            Add Adjustment
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SettlementLines({
  settlement,
  onChanged,
  readOnly,
}: {
  settlement: SettlementDetailData;
  onChanged: () => void;
  readOnly?: boolean;
}) {
  const grouped = useMemo(() => {
    const groups = new Map<string, CarrierSettlementLineRow[]>();
    for (const line of settlement.lines ?? []) {
      const list = groups.get(line.eventType) ?? [];
      list.push(line);
      groups.set(line.eventType, list);
    }
    return lineTypeOrder
      .filter((eventType) => groups.has(eventType))
      .map((eventType) => ({ eventType, lines: groups.get(eventType) ?? [] }));
  }, [settlement.lines]);

  const editable =
    !readOnly && (settlement.status === "Draft" || settlement.status === "PendingApproval");

  const removeMutation = useMutation({
    mutationFn: (lineId: string) =>
      removeCarrierSettlementAdjustment({ settlementId: settlement.id, lineId }),
    onSuccess: () => {
      toast.success("Adjustment removed");
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to remove adjustment"),
  });

  if (grouped.length === 0) {
    return <p className="text-sm text-muted-foreground">This settlement has no line items.</p>;
  }

  return (
    <div className="flex flex-col gap-4">
      {grouped.map(({ eventType, lines }) => {
        const subtotal = lines.reduce((sum, line) => sum + line.amountMinor, 0);
        return (
          <div key={eventType}>
            <div className="mb-1 flex items-baseline justify-between">
              <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                {lineTypeLabels[eventType] ?? eventType}
              </h4>
              <AmountDisplay value={subtotal} currency={settlement.currencyCode} />
            </div>
            <div className="overflow-hidden rounded-lg border">
              <table className="w-full text-xs">
                <tbody>
                  {lines.map((line) => (
                    <tr key={line.id ?? `${line.lineNumber}`} className="border-b last:border-b-0">
                      <td className="px-3 py-2">
                        <span className="font-medium">{line.description}</span>
                        {line.proNumber && (
                          <span className="ml-2 font-mono text-muted-foreground">
                            {line.proNumber}
                          </span>
                        )}
                      </td>
                      <td className="px-3 py-2 text-right text-muted-foreground">
                        {line.costEventId ? (
                          <span title="Traced back to the accrued cost event">cost event</span>
                        ) : null}
                      </td>
                      <td className="w-28 px-3 py-2 text-right">
                        <AmountDisplay
                          value={line.amountMinor}
                          variant="auto"
                          currency={settlement.currencyCode}
                        />
                      </td>
                      {editable && (
                        <td className="w-8 px-1 py-2">
                          {line.eventType === "Adjustment" && !line.costEventId && line.id && (
                            <Button
                              size="icon"
                              variant="ghost"
                              className="size-6"
                              disabled={removeMutation.isPending}
                              onClick={() => removeMutation.mutate(line.id)}
                              aria-label="Remove adjustment"
                            >
                              <X className="size-3" />
                            </Button>
                          )}
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function RemittanceCard({ settlement }: { settlement: SettlementDetailData }) {
  const carrier = settlement.carrier;
  if (!carrier) return null;

  const remitLines = [
    carrier.remitToName || carrier.name,
    carrier.remitAddressLine1,
    carrier.remitAddressLine2,
    [carrier.remitCity, carrier.remitState?.abbreviation, carrier.remitPostalCode]
      .filter(Boolean)
      .join(", "),
  ].filter((line): line is string => !!line && line.trim() !== "");

  return (
    <div className="rounded-lg border bg-muted/30 p-3">
      <h4 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
        Remittance
      </h4>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <p className="text-[11px] text-muted-foreground">Remit to</p>
          {remitLines.length > 0 ? (
            remitLines.map((line) => (
              <p key={line} className="text-xs font-medium">
                {line}
              </p>
            ))
          ) : (
            <p className="text-xs text-muted-foreground">
              No remit-to address on file — set it on the carrier record.
            </p>
          )}
        </div>
        <div className="flex flex-col gap-1.5">
          <div>
            <p className="text-[11px] text-muted-foreground">Payment method</p>
            <p className="text-xs font-medium">
              {settlement.paymentMethod ||
                (carrier.paymentMethod === "ACHManual" ? "ACH (Manual)" : carrier.paymentMethod)}
              {settlement.paymentReference ? ` · ${settlement.paymentReference}` : ""}
            </p>
          </div>
          <div>
            <p className="text-[11px] text-muted-foreground">Payment terms</p>
            <p className="text-xs font-medium">Net {carrier.paymentTermDays} days</p>
          </div>
        </div>
      </div>
    </div>
  );
}

function LinkedRateConfirmations({ settlement }: { settlement: SettlementDetailData }) {
  const moveIds = useMemo(() => {
    const ids = new Set<string>();
    for (const line of settlement.lines ?? []) {
      if (line.moveId) ids.add(line.moveId);
    }
    return [...ids].slice(0, MAX_RATE_CON_MOVES);
  }, [settlement.lines]);

  const { data: rateConfirmations, isLoading } = useQuery({
    queryKey: ["carrier-settlement-rate-cons", settlement.id, moveIds],
    queryFn: async () => {
      const lists = await Promise.all(
        moveIds.map((moveId) => apiService.rateConfirmationService.listByMove(moveId)),
      );
      return lists.flat();
    },
    enabled: moveIds.length > 0,
  });

  if (moveIds.length === 0) return null;

  const active = (rateConfirmations ?? []).filter((rateCon) => rateCon.status !== "Voided");

  return (
    <div>
      <h4 className="mb-1 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
        Rate Confirmations
      </h4>
      {isLoading ? (
        <Skeleton className="h-10 w-full" />
      ) : active.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">
          No rate confirmations on the covered moves.
        </p>
      ) : (
        <ul className="flex flex-col gap-1">
          {active.map((rateCon: RateConfirmation) => (
            <li
              key={rateCon.id}
              className="flex items-center gap-2 rounded-md border px-2 py-1.5 text-xs"
            >
              <RateConfirmationStatusBadge status={rateCon.status} />
              <span className="text-muted-foreground tabular-nums">rev {rateCon.revision}</span>
              {rateCon.confirmedByName && (
                <span className="truncate text-muted-foreground">
                  confirmed by {rateCon.confirmedByName}
                </span>
              )}
              {rateCon.documentId && (
                <a
                  href={documentContentUrl(rateCon.documentId, "view")}
                  target="_blank"
                  rel="noreferrer"
                  className="ml-auto inline-flex items-center gap-1 font-medium hover:underline"
                >
                  <FileText className="size-3" aria-hidden />
                  View PDF
                </a>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function LinkedInvoiceMatches({ settlement }: { settlement: SettlementDetailData }) {
  const { data, isLoading } = useQuery({
    queryKey: ["carrier-settlement-invoice-matches", settlement.carrierId],
    queryFn: () => fetchCarrierInvoiceMatches({ carrierId: settlement.carrierId, limit: 100 }),
  });

  const matches = useMemo(
    () => (data?.items ?? []).filter((match) => match.carrierSettlementId === settlement.id),
    [data, settlement.id],
  );

  if (isLoading) {
    return <Skeleton className="h-10 w-full" />;
  }

  if (matches.length === 0) return null;

  return (
    <div>
      <h4 className="mb-1 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
        Invoice Matches
      </h4>
      <ul className="flex flex-col gap-1">
        {matches.map((match) => (
          <li
            key={match.id}
            className="flex items-center gap-2 rounded-md border px-2 py-1.5 text-xs"
          >
            <CarrierInvoiceMatchStatusBadge status={match.status} />
            <span className="font-mono">{match.invoiceNumber || "No invoice #"}</span>
            <span className="text-muted-foreground">
              billed <AmountDisplay value={match.invoiceTotalMinor} currency={match.currencyCode} />
            </span>
            {match.varianceMinor !== 0 && (
              <span
                className={cn(
                  "ml-auto inline-flex rounded-full px-1.5 py-px text-[10px] font-medium",
                  "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
                )}
                title="Invoice total minus the expected buy rate"
              >
                variance{" "}
                <AmountDisplay
                  value={match.varianceMinor}
                  variant="auto"
                  currency={match.currencyCode}
                />
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}

function SettlementTimeline({ settlement }: { settlement: SettlementDetailData }) {
  const events = [
    { label: "Created", at: settlement.createdAt },
    { label: "Submitted", at: settlement.submittedAt },
    { label: "Approved", at: settlement.approvedAt },
    { label: "Posted", at: settlement.postedAt },
    {
      label: settlement.paymentMethod
        ? `Paid via ${settlement.paymentMethod}${
            settlement.paymentReference ? ` (${settlement.paymentReference})` : ""
          }`
        : "Paid",
      at: settlement.paidAt,
    },
    { label: "Voided", at: settlement.voidedAt },
  ].filter((event) => event.at);

  if (events.length <= 1) return null;

  return (
    <div className="border-t pt-3">
      <h4 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
        History
      </h4>
      <ol className="flex flex-col gap-1">
        {events.map((event) => (
          <li key={event.label} className="flex justify-between text-xs">
            <span>{event.label}</span>
            <span className="text-muted-foreground">{formatDate(event.at)}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
