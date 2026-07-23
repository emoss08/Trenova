import { AmountDisplay } from "@/components/accounting/amount-display";
import { DriverSettlementStatusBadge, PayeeClassificationBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { usePayCodeOptions } from "@/components/fields/pay-code-select-field";
import { Input } from "@/components/ui/input";
import { ScrollArea, type ScrollAreaMaskVariant } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  addDriverSettlementAdjustment,
  approveDriverSettlement,
  detachPayEventFromSettlement,
  fetchDriverSettlementDetail,
  markDriverSettlementPaid,
  postDriverSettlement,
  recalculateDriverSettlement,
  rejectDriverSettlement,
  removeDriverSettlementAdjustment,
  submitDriverSettlement,
  voidDriverSettlement,
  type DriverSettlementDetail as SettlementDetailData,
  type DriverSettlementLineRow,
} from "@/lib/graphql/driver-settlement";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import type { DriverSettlementStatus, PayeeClassification } from "@/types/driver-pay";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowUpRight,
  CheckCheck,
  CircleDollarSign,
  Plus,
  RefreshCcw,
  Send,
  TriangleAlert,
  Undo2,
  X,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

type ReasonAction = "reject" | "void";

const lineCategoryLabels: Record<string, string> = {
  Earning: "Earnings",
  Reimbursement: "Reimbursements",
  GuaranteeTopUp: "Guarantee Top-Up",
  CarryForward: "Carry Forward",
  Deduction: "Deductions",
  AdvanceRecovery: "Advance Recoveries",
  EscrowContribution: "Escrow Contributions",
  Adjustment: "Manual Adjustments",
};

const lineCategoryOrder = [
  "Earning",
  "GuaranteeTopUp",
  "Reimbursement",
  "CarryForward",
  "Deduction",
  "AdvanceRecovery",
  "EscrowContribution",
  "Adjustment",
];

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function SettlementDetail({
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
    queryKey: ["driver-settlement-detail", settlementId],
    queryFn: () => fetchDriverSettlementDetail(settlementId),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["driver-settlement-detail", settlementId] });
    void queryClient.invalidateQueries({ queryKey: ["driver-settlement-list"] });
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
        {settlement.hasExceptions && <ExceptionsBanner settlement={settlement} />}
        {readOnly ? (
          <ReadOnlyNotice settlement={settlement} />
        ) : (
          <SettlementActions settlement={settlement} onChanged={invalidate} onClose={onClose} />
        )}
        <SettlementLines settlement={settlement} onChanged={invalidate} readOnly={readOnly} />
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
          to={`/payroll/workspace?settlement=${settlement.id}`}
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
        <DriverSettlementStatusBadge status={settlement.status as DriverSettlementStatus} />
        <PayeeClassificationBadge
          classification={settlement.classification as PayeeClassification}
        />
        {settlement.payProfileName && (
          <span className="text-xs text-muted-foreground">
            Pay profile: {settlement.payProfileName}
          </span>
        )}
        <span className="ml-auto text-xs text-muted-foreground">
          {formatDate(settlement.periodStart)} – {formatDate(settlement.periodEnd)} · pays{" "}
          {formatDate(settlement.payDate)}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <SummaryTile label="Gross Earnings">
          <AmountDisplay value={settlement.grossEarningsMinor} currency={settlement.currencyCode} />
        </SummaryTile>
        <SummaryTile label="Deductions">
          <AmountDisplay
            value={-settlement.deductionsMinor}
            variant="negative"
            currency={settlement.currencyCode}
          />
        </SummaryTile>
        <SummaryTile label="Miles / Loads">
          <span className="tabular-nums">
            {Number(settlement.totalMiles).toLocaleString()} mi · {settlement.shipmentCount}
          </span>
        </SummaryTile>
        <SummaryTile label="Net Pay" highlight>
          <AmountDisplay
            value={settlement.netPayMinor}
            variant="positive"
            currency={settlement.currencyCode}
          />
        </SummaryTile>
      </div>
      {settlement.carryForwardOutMinor < 0 && (
        <p className="text-xs text-red-600 dark:text-red-400">
          Deductions exceeded earnings.{" "}
          <AmountDisplay
            value={-settlement.carryForwardOutMinor}
            currency={settlement.currencyCode}
          />{" "}
          will carry forward to the next settlement.
        </p>
      )}
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

function ExceptionsBanner({ settlement }: { settlement: SettlementDetailData }) {
  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50/60 p-3 dark:border-amber-900 dark:bg-amber-950/30">
      <div className="flex items-center gap-2 text-sm font-medium text-amber-800 dark:text-amber-300">
        <TriangleAlert className="size-4" />
        Review required before approval
      </div>
      <ul className="mt-2 flex flex-col gap-1">
        {(settlement.exceptions ?? []).map((exception) => (
          <li key={exception.code} className="flex items-start gap-2 text-xs">
            <span
              className={cn(
                "mt-0.5 inline-flex rounded-full px-1.5 py-px text-[10px] font-medium",
                exception.severity === "Critical"
                  ? "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300"
                  : "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
              )}
            >
              {exception.severity}
            </span>
            <span className="text-amber-900 dark:text-amber-200">{exception.message}</span>
          </li>
        ))}
      </ul>
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
          return submitDriverSettlement(actionInput);
        case "approve":
          return approveDriverSettlement(actionInput);
        case "post":
          return postDriverSettlement(actionInput);
        case "recalculate":
          return recalculateDriverSettlement(actionInput);
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
          recalculate: "Settlement recalculated from current pay events",
        }[action] ?? "Settlement updated",
      );
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Settlement action failed"),
  });

  const status = settlement.status as DriverSettlementStatus;
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
        settlementId={settlement.id}
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
        ? rejectDriverSettlement({ settlementId, reason })
        : voidDriverSettlement({ settlementId, reason }),
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
              : "Voiding releases pay events back to the accrual pool and reverses any GL postings."}
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
  settlementId,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  settlementId: string;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const [paymentMethod, setPaymentMethod] = useState("ACH");
  const [paymentReference, setPaymentReference] = useState("");

  const mutation = useMutation({
    mutationFn: () => markDriverSettlementPaid({ settlementId, paymentMethod, paymentReference }),
    onSuccess: () => {
      toast.success("Settlement marked paid");
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
          <DialogDescription>Record how the net pay was disbursed to the driver.</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div>
            <p className="mb-1 text-xs font-medium">Payment method</p>
            <p className="mb-1 text-[11px] text-muted-foreground">
              How the net pay was sent to the driver — recorded on the statement and audit trail.
            </p>
            <div className="flex gap-2">
              {["ACH", "Check", "InstantPay", "Other"].map((method) => (
                <Button
                  key={method}
                  size="sm"
                  variant={paymentMethod === method ? "default" : "outline"}
                  onClick={() => setPaymentMethod(method)}
                >
                  {method === "InstantPay" ? "Instant Pay" : method}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <p className="mb-1 text-xs font-medium">Reference</p>
            <Input
              value={paymentReference}
              onChange={(e) => setPaymentReference(e.target.value)}
              placeholder="ACH trace / check number (optional)"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              The ACH trace or check number so the payment can be reconciled with the bank.
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
  const [payCodeId, setPayCodeId] = useState<string>("none");
  const { data: payCodes } = usePayCodeOptions();

  const parsedAmount = Number(amount);
  const valid = description.trim() !== "" && !Number.isNaN(parsedAmount) && parsedAmount !== 0;

  const mutation = useMutation({
    mutationFn: () =>
      addDriverSettlementAdjustment({
        settlementId,
        description: description.trim(),
        amountMinor: Math.round(parsedAmount * 100),
        payCodeId: payCodeId === "none" ? undefined : payCodeId,
      }),
    onSuccess: () => {
      toast.success("Adjustment added");
      setDescription("");
      setAmount("");
      setPayCodeId("none");
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
            Positive amounts add pay (layover, breakdown, bonus); negative amounts deduct.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Description (e.g. Layover pay - Detroit 6/12)"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              Appears as the line item on the driver&apos;s statement — say what and when.
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
              Dollars, not cents; positive adds pay, negative deducts.
            </p>
          </div>
          <div>
            <Select
              value={payCodeId}
              items={(payCodes ?? []).map((code) => ({
                label: `${code.code} — ${code.name} (${code.direction})`,
                value: code.id,
              }))}
              onValueChange={(value) => setPayCodeId(value ?? "none")}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Pay code (optional)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">No pay code</SelectItem>
                {(payCodes ?? []).map((code) => (
                  <SelectItem key={code.id} value={code.id}>
                    {code.code} — {code.name} ({code.direction})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Optional — tagging a pay code posts the adjustment to that code&apos;s GL account
              instead of the default expense account.
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
    const groups = new Map<string, DriverSettlementLineRow[]>();
    for (const line of settlement.lines ?? []) {
      const list = groups.get(line.category) ?? [];
      list.push(line);
      groups.set(line.category, list);
    }
    return lineCategoryOrder
      .filter((category) => groups.has(category))
      .map((category) => ({ category, lines: groups.get(category) ?? [] }));
  }, [settlement.lines]);

  const editable =
    !readOnly && (settlement.status === "Draft" || settlement.status === "PendingApproval");

  const removeMutation = useMutation({
    mutationFn: (lineId: string) =>
      removeDriverSettlementAdjustment({ settlementId: settlement.id, lineId }),
    onSuccess: () => {
      toast.success("Adjustment removed");
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to remove adjustment"),
  });

  const detachMutation = useMutation({
    mutationFn: (payEventId: string) =>
      detachPayEventFromSettlement({ settlementId: settlement.id, payEventId }),
    onSuccess: () => {
      toast.success("Pay event returned to the unsettled pool");
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to remove pay event"),
  });
  const canDetach = !readOnly && settlement.status === "Draft";

  if (grouped.length === 0) {
    return <p className="text-sm text-muted-foreground">This settlement has no line items.</p>;
  }

  return (
    <div className="flex flex-col gap-4">
      {grouped.map(({ category, lines }) => {
        const subtotal = lines.reduce((sum, line) => sum + line.amountMinor, 0);
        return (
          <div key={category}>
            <div className="mb-1 flex items-baseline justify-between">
              <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                {lineCategoryLabels[category] ?? category}
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
                      <td className="px-3 py-2 text-right text-muted-foreground tabular-nums">
                        {Number(line.quantity) > 0 && (
                          <>
                            {Number(line.quantity).toLocaleString()}
                            {Number(line.rate) > 0 && <> × {Number(line.rate).toFixed(4)}</>}
                          </>
                        )}
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
                          {line.category === "Adjustment" && line.id && (
                            <Button
                              size="icon"
                              variant="ghost"
                              className="size-6"
                              disabled={removeMutation.isPending}
                              onClick={() => removeMutation.mutate(line.id as string)}
                              aria-label="Remove adjustment"
                            >
                              <X className="size-3" />
                            </Button>
                          )}
                          {canDetach && line.category === "Earning" && line.payEventId && (
                            <Button
                              size="icon"
                              variant="ghost"
                              className="size-6"
                              disabled={detachMutation.isPending}
                              onClick={() => detachMutation.mutate(line.payEventId as string)}
                              aria-label="Remove pay event from settlement"
                              title="Remove this pay event — it returns to the unsettled pool for a later settlement"
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
