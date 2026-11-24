import { Link } from "@trenova/shared/components/link";
import { AccountingSyncStateLine } from "@/components/accounting-sync/sync-state-line";
import { ExchangeRateLine } from "@/components/accounting/exchange-rate-line";
import { type TranslateFn, useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { ReasonDialog as ChoiceReasonDialog } from "@/components/reason-dialog";
import { SettlementHoldBadge } from "@/components/settlements/settlement-hold-badge";
import { usePermission } from "@/hooks/use-permission";
import {
  canBeHeld,
  HOLD_REASON_LABELS,
  HOLD_REASONS,
  HOLD_REASONS_NEEDING_A_NOTE,
  holdReasonLabel,
  isHeld,
  MAX_HOLD_NOTE_LENGTH,
} from "@/lib/settlement-hold";
import type { DriverSettlementHoldReason } from "@trenova/graphql/generated/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  DriverSettlementStatusBadge,
  PayeeClassificationBadge,
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
import { ControlledPayCodeAutocompleteField } from "@/components/autocomplete-fields";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea, type ScrollAreaMaskVariant } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  addDriverSettlementAdjustment,
  approveDriverSettlement,
  detachPayEventFromSettlement,
  fetchDriverSettlementDetail,
  holdDriverSettlement,
  markDriverSettlementPaid,
  postDriverSettlement,
  recalculateDriverSettlement,
  rejectDriverSettlement,
  releaseDriverSettlementHold,
  removeDriverSettlementAdjustment,
  submitDriverSettlement,
  voidDriverSettlement,
  type DriverSettlementDetail as SettlementDetailData,
  type DriverSettlementLineRow,
} from "@/lib/graphql/driver-settlement";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import type { DriverSettlementStatus, PayeeClassification } from "@trenova/shared/types/driver-pay";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  ArrowUpRightIcon,
  CheckDoubleIcon,
  CurrencyDollarCircleIcon,
  FlipBackwardIcon,
  PauseCircleIcon,
  PlayCircleIcon,
  PlusIcon,
  RefreshCcw02Icon,
  Send01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";

export type ReasonAction = "reject" | "void";

const lineCategoryLabels: Record<string, string> = defineLabels({
  Earning: "Earnings",
  Reimbursement: "Reimbursements",
  GuaranteeTopUp: "Guarantee Top-Up",
  CarryForward: "Carry Forward",
  Deduction: "Deductions",
  AdvanceRecovery: "Advance Recoveries",
  EscrowContribution: "Escrow Contributions",
  Adjustment: "Manual Adjustments",
});

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
  return formatUnixDateMedium(unix, { fallback: "—" });
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
    queryFn: ({ signal }) => fetchDriverSettlementDetail(settlementId, { signal }),
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
        {isHeld(settlement) && (
          <HoldNotice settlement={settlement} onChanged={invalidate} readOnly={readOnly} />
        )}
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
  const t = useT();

  const isTerminal = settlement.status === "Paid" || settlement.status === "Voided";

  return (
    <div className="bg-muted/30 flex items-center justify-between gap-3 rounded-lg border px-3 py-2">
      <p className="text-muted-foreground text-xs">
        {isTerminal
          ? t("This settlement is finalized and shown here for record-keeping.")
          : t(
              "This is a read-only view — process, adjust, or pay this settlement from the workspace.",
            )}
      </p>
      {!isTerminal && (
        <Link
          to="/payroll/workspace"
          search={{ settlement: settlement.id }}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-7 shrink-0 text-xs")}
        >
          <ArrowUpRightIcon className="size-3.5" />
          {t("Open in workspace")}
        </Link>
      )}
    </div>
  );
}

function SettlementSummary({ settlement }: { settlement: SettlementDetailData }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <DriverSettlementStatusBadge status={settlement.status as DriverSettlementStatus} />
        {isHeld(settlement) && (
          <SettlementHoldBadge reason={settlement.holdReason} note={settlement.holdNote} />
        )}
        <PayeeClassificationBadge
          classification={settlement.classification as PayeeClassification}
        />
        {settlement.payProfileName && (
          <span className="text-muted-foreground text-xs">
            {t("Pay profile: {0}", settlement.payProfileName)}
          </span>
        )}
        <span className="text-muted-foreground ml-auto text-xs">
          {t(
            "{0} – {1} · pays {2}",
            formatDate(settlement.periodStart),
            formatDate(settlement.periodEnd),
            formatDate(settlement.payDate),
          )}
        </span>
      </div>
      <AccountingSyncStateLine objectId={settlement.id} className="text-xs" />
      <ExchangeRateLine
        currency={settlement.currencyCode}
        rate={settlement.exchangeRate}
        quotedOn={settlement.exchangeRateDate}
        paidRate={settlement.paidExchangeRate}
        paidQuotedOn={settlement.paidExchangeRateDate}
      />
      <KpiStrip minItemWidth="8rem">
        <KpiStripItem
          label={t("Gross earnings")}
          value={
            <AmountDisplay
              value={settlement.grossEarningsMinor}
              currency={settlement.currencyCode}
            />
          }
        />
        <KpiStripItem
          label={t("Deductions")}
          value={
            <AmountDisplay
              value={-settlement.deductionsMinor}
              variant="negative"
              currency={settlement.currencyCode}
            />
          }
        />
        <KpiStripItem
          label={t("Miles / loads")}
          value={
            <span className="tabular-nums">
              {t(
                "{0} mi · {1}",
                Number(settlement.totalMiles).toLocaleString(),
                settlement.shipmentCount,
              )}
            </span>
          }
        />
        <KpiStripItem
          label={t("Net pay")}
          tone="success"
          value={
            <AmountDisplay
              value={settlement.netPayMinor}
              variant="positive"
              currency={settlement.currencyCode}
            />
          }
        />
      </KpiStrip>
      {settlement.carryForwardOutMinor < 0 && (
        <p className="text-xs text-danger-foreground">
          {t("Deductions exceeded earnings.")}{" "}
          <AmountDisplay
            value={-settlement.carryForwardOutMinor}
            currency={settlement.currencyCode}
          />{" "}
          {t("will carry forward to the next settlement.")}
        </p>
      )}
    </div>
  );
}

function HoldNotice({
  settlement,
  onChanged,
  readOnly,
}: {
  settlement: SettlementDetailData;
  onChanged: () => void;
  readOnly: boolean;
}) {
  const t = useT();
  const { allowed: canHold } = usePermission(Resource.DriverSettlement, Operation.Hold);
  const release = useMutation({
    mutationFn: () => releaseDriverSettlementHold(settlement.id),
    onSuccess: () => onChanged(),
    onError: (error: Error) =>
      toast.error(error.message || translate("The hold could not be released")),
  });
  const label = holdReasonLabel(settlement.holdReason);
  const heldBy = settlement.heldBy?.name;

  return (
    <Alert variant="warning" size="sm">
      <PauseCircleIcon />
      <AlertTitle>{label ? t("On hold: {0}", label) : t("On hold")}</AlertTitle>
      <AlertDescription>
        {settlement.holdNote && <p>{settlement.holdNote}</p>}
        <p className="text-muted-foreground">
          {heldBy
            ? t(
                "Held by {0} on {1}. Approving, posting and paying it wait until the hold is released.",
                heldBy,
                formatDate(settlement.heldAt),
              )
            : t(
                "Held on {0}. Approving, posting and paying it wait until the hold is released.",
                formatDate(settlement.heldAt),
              )}
        </p>
      </AlertDescription>
      {!readOnly && canHold && (
        <AlertAction>
          <Button size="sm" isLoading={release.isPending} onClick={() => release.mutateAsync()}>
            <PlayCircleIcon className="size-3.5" />
            {t("Release hold")}
          </Button>
        </AlertAction>
      )}
    </Alert>
  );
}

function ExceptionsBanner({ settlement }: { settlement: SettlementDetailData }) {
  const t = useT();

  return (
    <Alert variant="warning">
      <AlertTriangleIcon />
      <AlertTitle>{t("Review required before approval")}</AlertTitle>
      <AlertDescription>
        <ul className="flex flex-col gap-1">
          {(settlement.exceptions ?? []).map((exception) => (
            <li key={exception.code} className="flex items-start gap-2 text-xs">
              <span
                className={cn(
                  "mt-0.5 inline-flex rounded-badge px-1.5 py-px text-2xs font-medium",
                  exception.severity === "Critical"
                    ? "bg-danger-subtle text-danger-subtle-foreground"
                    : "bg-warning-subtle text-warning-subtle-foreground",
                )}
              >
                {exception.severity}
              </span>
              <span>{exception.message}</span>
            </li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
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
  const t = useT();

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
    onSuccess: () => {
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || translate("Settlement action failed")),
  });

  const { allowed: canHold } = usePermission(Resource.DriverSettlement, Operation.Hold);
  const [holdDialogOpen, setHoldDialogOpen] = useState(false);
  const hold = useMutation({
    mutationFn: (input: { reason: DriverSettlementHoldReason; note: string }) =>
      holdDriverSettlement({ settlementId: settlement.id, reason: input.reason, note: input.note }),
    onSuccess: () => {
      setHoldDialogOpen(false);
      onChanged();
    },
    onError: (error: Error) =>
      toast.error(error.message || translate("The settlement could not be held")),
  });

  const status = settlement.status as DriverSettlementStatus;
  const held = isHeld(settlement);
  const busy = runAction.isPending || hold.isPending;
  const heldTitle = held
    ? t("On hold: release the hold before approving, posting or paying it")
    : undefined;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {status === "Draft" && (
        <>
          <Button size="sm" disabled={busy} onClick={() => runAction.mutate("submit")}>
            <Send01Icon className="size-3.5" />
            {t("Submit for approval")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={busy}
            onClick={() => runAction.mutate("recalculate")}
          >
            <RefreshCcw02Icon className="size-3.5" />
            {t("Recalculate")}
          </Button>
        </>
      )}
      {status === "PendingApproval" && (
        <>
          <Button
            size="sm"
            disabled={busy || held}
            title={heldTitle}
            onClick={() => runAction.mutate("approve")}
          >
            <CheckDoubleIcon className="size-3.5" />
            {t("Approve")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={busy}
            onClick={() => setReasonAction("reject")}
          >
            <FlipBackwardIcon className="size-3.5" />
            {t("Reject")}
          </Button>
        </>
      )}
      {status === "Approved" && (
        <Button
          size="sm"
          disabled={busy || held}
          title={heldTitle}
          onClick={() => runAction.mutate("post")}
        >
          <CheckDoubleIcon className="size-3.5" />
          {t("Post to GL")}
        </Button>
      )}
      {status === "Posted" && (
        <Button size="sm" disabled={busy} onClick={() => setPayDialogOpen(true)}>
          <CurrencyDollarCircleIcon className="size-3.5" />
          {t("Mark paid")}
        </Button>
      )}
      {(status === "Draft" || status === "PendingApproval") && (
        <Button
          size="sm"
          variant="outline"
          disabled={busy}
          onClick={() => setAdjustDialogOpen(true)}
        >
          <PlusIcon className="size-3.5" />
          {t("Add adjustment")}
        </Button>
      )}
      {canHold && canBeHeld(settlement) && (
        <Button
          size="sm"
          variant="outline"
          disabled={busy}
          onClick={() => setHoldDialogOpen(true)}
          title={t("Keep this settlement out of the pay run until the hold is released")}
        >
          <PauseCircleIcon className="size-3.5" />
          {t("Hold")}
        </Button>
      )}
      {status !== "Paid" && status !== "Voided" && (
        <Button
          size="sm"
          variant="ghost"
          className="text-danger-foreground hover:text-danger-foreground"
          disabled={busy}
          onClick={() => setReasonAction("void")}
        >
          <XCloseIcon className="size-3.5" />
          {t("Void")}
        </Button>
      )}

      <ReasonDialog
        action={reasonAction}
        settlementId={settlement.id}
        onOpenChange={(open) => !open && setReasonAction(null)}
        onChanged={onChanged}
      />
      <ChoiceReasonDialog
        open={holdDialogOpen}
        onOpenChange={setHoldDialogOpen}
        title={t("Hold settlement {0}", settlement.settlementNumber)}
        description={holdDialogDescription(t, status)}
        choice={{
          label: t("Why"),
          placeholder: t("Pick a reason"),
          options: HOLD_REASONS.map((reason) => ({
            value: reason,
            label: HOLD_REASON_LABELS[reason],
          })),
          needsWords: [...HOLD_REASONS_NEEDING_A_NOTE],
        }}
        label={t("Note")}
        placeholder={t("What the hold waits on, such as the broker's corrected rate confirmation")}
        fieldDescription={t("Whoever releases the hold reads this. The driver does not.")}
        maxLength={MAX_HOLD_NOTE_LENGTH}
        optional
        confirmLabel={t("Hold settlement")}
        pending={hold.isPending}
        onConfirm={(note, reason) => {
          if (reason) hold.mutate({ reason: reason as DriverSettlementHoldReason, note });
        }}
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
        <span className="text-muted-foreground text-xs">
          {t("Voided: {0}", settlement.voidReason)}
        </span>
      )}
      <span className="sr-only">
        <Button variant="ghost" onClick={onClose}>
          {t("Close")}
        </Button>
      </span>
    </div>
  );
}

export function ReasonDialog({
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
  const t = useT();

  const [reason, setReason] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      action === "reject"
        ? rejectDriverSettlement({ settlementId, reason })
        : voidDriverSettlement({ settlementId, reason }),
    onSuccess: () => {
      setReason("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || translate("Action failed")),
  });

  return (
    <Dialog open={action != null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {action === "reject" ? t("Reject settlement") : t("Void settlement")}
          </DialogTitle>
          <DialogDescription>
            {action === "reject"
              ? t("The settlement will return to draft for corrections.")
              : t(
                  "Voiding releases pay events back to the accrual pool and reverses any GL postings.",
                )}
          </DialogDescription>
        </DialogHeader>
        <Textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder={t("Reason (required)")}
          rows={3}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            variant={action === "void" ? "destructive" : "default"}
            disabled={!reason.trim() || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {action === "reject" ? t("Reject") : t("Void")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function MarkPaidDialog({
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
  const t = useT();

  const [paymentMethod, setPaymentMethod] = useState("ACH");
  const [paymentReference, setPaymentReference] = useState("");

  const mutation = useMutation({
    mutationFn: () => markDriverSettlementPaid({ settlementId, paymentMethod, paymentReference }),
    onSuccess: () => {
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) =>
      toast.error(error.message || translate("Failed to mark settlement paid")),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Mark settlement paid")}</DialogTitle>
          <DialogDescription>
            {t("Record how the net pay was disbursed to the worker.")}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div>
            <p className="mb-1 text-xs font-medium">{t("Payment method")}</p>
            <p className="text-muted-foreground mb-1 text-xs">
              {t(
                "How the net pay was sent to the driver — recorded on the statement and audit trail.",
              )}
            </p>
            <div className="flex gap-2">
              {["ACH", "Check", "InstantPay", "Other"].map((method) => (
                <Button
                  key={method}
                  size="sm"
                  variant={paymentMethod === method ? "default" : "outline"}
                  onClick={() => setPaymentMethod(method)}
                >
                  {method === "InstantPay" ? t("Instant pay") : method}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <p className="mb-1 text-xs font-medium">{t("Reference")}</p>
            <Input
              value={paymentReference}
              onChange={(e) => setPaymentReference(e.target.value)}
              placeholder={t("ACH trace / check number (optional)")}
            />
            <p className="text-muted-foreground mt-1 text-xs">
              {t("The ACH trace or check number so the payment can be reconciled with the bank.")}
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button disabled={mutation.isPending} onClick={() => mutation.mutate()}>
            {t("Mark paid")}
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
  const t = useT();

  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");
  const [payCodeId, setPayCodeId] = useState<string>("");

  const parsedAmount = Number(amount);
  const valid = description.trim() !== "" && !Number.isNaN(parsedAmount) && parsedAmount !== 0;

  const mutation = useMutation({
    mutationFn: () =>
      addDriverSettlementAdjustment({
        settlementId,
        description: description.trim(),
        amountMinor: Math.round(parsedAmount * 100),
        payCodeId: payCodeId || undefined,
      }),
    onSuccess: () => {
      setDescription("");
      setAmount("");
      setPayCodeId("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || translate("Failed to add adjustment")),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Add manual adjustment")}</DialogTitle>
          <DialogDescription>
            {t("Positive amounts add pay (layover, breakdown, bonus); negative amounts deduct.")}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("Description (e.g. Layover pay - Detroit 6/12)")}
            />
            <p className="text-muted-foreground mt-1 text-xs">
              {t("Appears as the line item on the driver's statement — say what and when.")}
            </p>
          </div>
          <div>
            <Input
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder={t("Amount (e.g. 150.00 or -75.00)")}
              inputMode="decimal"
            />
            <p className="text-muted-foreground mt-1 text-xs">
              {t("Dollars, not cents; positive adds pay, negative deducts.")}
            </p>
          </div>
          <ControlledPayCodeAutocompleteField
            label={t("Pay code (optional)")}
            value={payCodeId}
            onValueChange={setPayCodeId}
            description={t(
              "Optional — tagging a pay code posts the adjustment to that code's GL account instead of the default expense account.",
            )}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button disabled={!valid || mutation.isPending} onClick={() => mutation.mutate()}>
            {t("Add adjustment")}
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
  const t = useT();

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
      onChanged();
    },
    onError: (error: Error) =>
      toast.error(error.message || translate("Failed to remove adjustment")),
  });

  const detachMutation = useMutation({
    mutationFn: (payEventId: string) =>
      detachPayEventFromSettlement({ settlementId: settlement.id, payEventId }),
    onSuccess: () => {
      onChanged();
    },
    onError: (error: Error) =>
      toast.error(error.message || translate("Failed to remove pay event")),
  });
  const canDetach = !readOnly && settlement.status === "Draft";

  if (grouped.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">{t("This settlement has no line items.")}</p>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {grouped.map(({ category, lines }) => {
        const subtotal = lines.reduce((sum, line) => sum + line.amountMinor, 0);
        return (
          <div key={category}>
            <div className="mb-1 flex items-baseline justify-between">
              <h4 className="text-muted-foreground text-xs font-semibold">
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
                        <span className="font-medium">{t(line.description)}</span>
                        {line.proNumber && (
                          <span className="text-muted-foreground ml-2 font-mono">
                            {line.proNumber}
                          </span>
                        )}
                      </td>
                      <td className="text-muted-foreground px-3 py-2 text-right tabular-nums">
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
                              aria-label={t("Remove adjustment")}
                            >
                              <XCloseIcon className="size-3" />
                            </Button>
                          )}
                          {canDetach && line.category === "Earning" && line.payEventId && (
                            <Button
                              size="icon"
                              variant="ghost"
                              className="size-6"
                              disabled={detachMutation.isPending}
                              onClick={() => detachMutation.mutate(line.payEventId as string)}
                              aria-label={t("Remove pay event from settlement")}
                              title={t(
                                "Remove this pay event — it returns to the unsettled pool for a later settlement",
                              )}
                            >
                              <XCloseIcon className="size-3" />
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
  const t = useT();

  const events = [
    { label: t("Created"), at: settlement.createdAt },
    { label: t("Submitted"), at: settlement.submittedAt },
    { label: t("Approved"), at: settlement.approvedAt },
    { label: t("Posted"), at: settlement.postedAt },
    {
      label: settlement.paymentMethod
        ? settlement.paymentReference
          ? t("Paid via {0} ({1})", settlement.paymentMethod, settlement.paymentReference)
          : t("Paid via {0}", settlement.paymentMethod)
        : t("Paid"),
      at: settlement.paidAt,
    },
    { label: t("Voided"), at: settlement.voidedAt },
    { label: t("Held"), at: settlement.heldAt },
    { label: t("Hold released"), at: settlement.holdReleasedAt },
  ].filter((event) => event.at);

  if (events.length <= 1) return null;

  return (
    <div className="border-t pt-3">
      <h4 className="text-muted-foreground mb-2 text-xs font-semibold">{t("History")}</h4>
      <ol className="flex flex-col gap-1">
        {events.map((event) => (
          <li key={event.label} className="flex justify-between text-xs">
            <span>{t(event.label)}</span>
            <span className="text-muted-foreground">{formatDate(event.at)}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}

function holdDialogDescription(t: TranslateFn, status: DriverSettlementStatus): string {
  switch (status) {
    case "Draft":
      return t(
        "It stays a draft and keeps its place in the batch, but approving, posting and paying it are refused until someone releases the hold.",
      );
    case "PendingApproval":
      return t(
        "It stays pending approval and keeps its place in the batch, but approving, posting and paying it are refused until someone releases the hold.",
      );
    default:
      return t(
        "It stays approved and keeps its place in the batch, but posting and paying it are refused until someone releases the hold.",
      );
  }
}
