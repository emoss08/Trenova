import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  attachPayEventsToSettlement,
  fetchWorkerEarningsSummary,
  fetchWorkerPayAdvances,
  fetchWorkerRecurringDeductions,
  fetchWorkerRecurringEarnings,
  fetchWorkerUnsettledPayEvents,
  holdDriverPayEvent,
  releaseDriverPayEvent,
  updateRecurringDeduction,
  updateRecurringEarning,
  type DriverPayEventRow,
  type DriverSettlementRow,
  type RecurringDeductionRow,
  type RecurringEarningRow,
} from "@/lib/graphql/driver-settlement";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ArrowLeftToLine, Pause, PauseCircle, Play, PlusCircle } from "lucide-react";
import { useState, type ReactNode } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  });
}

export function DriverContextRail({
  workerId,
  workerName,
  selectedSettlement,
  onChanged,
}: {
  workerId: string | null;
  workerName: string | null;
  selectedSettlement: DriverSettlementRow | null;
  onChanged: () => void;
}) {
  if (!workerId) {
    return (
      <div className="hidden items-center justify-center rounded-lg border bg-card p-6 text-center text-xs text-muted-foreground lg:flex">
        Driver context appears here once a settlement is selected.
      </div>
    );
  }

  return (
    <div className="hidden min-h-0 flex-col overflow-hidden rounded-lg border bg-card lg:flex">
      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="min-h-0"
        maskVariant="card"
        maskHeight={18}
      >
        <div className="flex flex-col gap-3 p-3">
          <div>
            <h3 className="text-sm font-semibold">{workerName ?? "Driver"}</h3>
            <p className="text-[11px] text-muted-foreground">
              Everything affecting this driver&apos;s pay — manage it without leaving the workspace.
            </p>
          </div>
          <UnsettledPaySection
            workerId={workerId}
            selectedSettlement={selectedSettlement}
            onChanged={onChanged}
          />
          <EarningsSection workerId={workerId} onChanged={onChanged} />
          <DeductionsSection workerId={workerId} onChanged={onChanged} />
          <AdvancesSection workerId={workerId} />
          <EscrowSection workerId={workerId} />
        </div>
      </ScrollArea>
    </div>
  );
}

function RailSection({
  title,
  hint,
  action,
  children,
}: {
  title: string;
  hint: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="border-t pt-3 first:border-t-0 first:pt-0">
      <div className="mb-1.5 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
            {title}
          </h4>
          <p className="text-[10px] text-muted-foreground">{hint}</p>
        </div>
        {action}
      </div>
      {children}
    </div>
  );
}

function UnsettledPaySection({
  workerId,
  selectedSettlement,
  onChanged,
}: {
  workerId: string;
  selectedSettlement: DriverSettlementRow | null;
  onChanged: () => void;
}) {
  const [holdTarget, setHoldTarget] = useState<DriverPayEventRow | null>(null);
  const { data: events, isLoading } = useQuery({
    queryKey: ["worker-unsettled-events", workerId],
    queryFn: () => fetchWorkerUnsettledPayEvents(workerId),
  });

  const canAttach = selectedSettlement?.status === "Draft";

  const attachMutation = useMutation({
    mutationFn: (payEventId: string) =>
      attachPayEventsToSettlement({
        settlementId: selectedSettlement?.id as string,
        payEventIds: [payEventId],
      }),
    onSuccess: () => {
      toast.success("Pay event added to the settlement");
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to add pay event"),
  });

  const releaseMutation = useMutation({
    mutationFn: (payEventId: string) => releaseDriverPayEvent(payEventId),
    onSuccess: () => {
      toast.success("Hold released — the event will settle normally");
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to release hold"),
  });

  if (isLoading) {
    return (
      <RailSection title="Unsettled Pay" hint="Accrued pay not yet on a settlement.">
        <Skeleton className="h-16 w-full" />
      </RailSection>
    );
  }

  const list = events ?? [];

  return (
    <RailSection
      title="Unsettled Pay"
      hint="Accrued pay not yet on a settlement — attach it, or hold it for a later period."
    >
      {list.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">
          Nothing waiting. New pay accrues automatically as moves complete.
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {list.map((event) => (
            <li
              key={event.id}
              className={cn(
                "rounded-md border p-2",
                event.onHold &&
                  "border-blue-200 bg-blue-50/40 dark:border-blue-900 dark:bg-blue-950/20",
              )}
            >
              <div className="flex items-center gap-1.5">
                <span className="font-mono text-[10px] text-muted-foreground">
                  {event.proNumber || "No PRO"}
                </span>
                <span className="text-[10px] text-muted-foreground">
                  {formatDate(event.eventDate)}
                </span>
                <span className="ml-auto text-xs font-semibold">
                  <AmountDisplay value={event.grossAmountMinor} currency={event.currencyCode} />
                </span>
              </div>
              {event.onHold && (
                <p className="mt-1 text-[10px] text-blue-700 dark:text-blue-300">
                  On hold: {event.holdReason}
                </p>
              )}
              <div className="mt-1.5 flex gap-1">
                {canAttach && !event.onHold && (
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-6 px-2 text-[10px]"
                    disabled={attachMutation.isPending}
                    onClick={() => attachMutation.mutate(event.id)}
                    title="Add this pay event to the selected draft settlement"
                  >
                    <ArrowLeftToLine className="size-3" />
                    Add to settlement
                  </Button>
                )}
                {event.onHold ? (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-6 px-2 text-[10px]"
                    disabled={releaseMutation.isPending}
                    onClick={() => releaseMutation.mutate(event.id)}
                  >
                    <Play className="size-3" />
                    Release hold
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-6 px-2 text-[10px] text-muted-foreground"
                    onClick={() => setHoldTarget(event)}
                    title="Defer this pay to a later settlement — it will skip generation until released"
                  >
                    <Pause className="size-3" />
                    Hold
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
      <HoldDialog
        event={holdTarget}
        onOpenChange={(open) => !open && setHoldTarget(null)}
        onChanged={onChanged}
      />
    </RailSection>
  );
}

function HoldDialog({
  event,
  onOpenChange,
  onChanged,
}: {
  event: DriverPayEventRow | null;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const [reason, setReason] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      holdDriverPayEvent({ payEventId: event?.id as string, reason: reason.trim() }),
    onSuccess: () => {
      toast.success("Pay event held — it will skip settlements until released");
      setReason("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to hold pay event"),
  });

  return (
    <Dialog open={event != null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Hold pay event</DialogTitle>
          <DialogDescription>
            Held pay stays accrued but is skipped by settlement generation and auto-attach until you
            release it — use it for disputed loads or pay you want on a later statement.
          </DialogDescription>
        </DialogHeader>
        <Textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Reason (required) — e.g. Disputed detention, awaiting customer confirmation"
          rows={3}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={!reason.trim() || mutation.isPending} onClick={() => mutation.mutate()}>
            <PauseCircle className="size-4" />
            Hold Pay
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function EarningsSection({ workerId, onChanged }: { workerId: string; onChanged: () => void }) {
  const { data: earnings, isLoading } = useQuery({
    queryKey: ["worker-recurring-earnings", workerId],
    queryFn: () => fetchWorkerRecurringEarnings(workerId),
  });

  const toggleMutation = useMutation({
    mutationFn: (earning: RecurringEarningRow) =>
      updateRecurringEarning({
        id: earning.id,
        version: earning.version,
        workerId: earning.workerId,
        payCodeId: earning.payCodeId,
        status: earning.status === "Paused" ? "Active" : "Paused",
        frequency: earning.frequency,
        description: earning.description,
        amountMinor: earning.amountMinor,
        totalCapMinor: earning.totalCapMinor ?? undefined,
        startDate: earning.startDate,
        endDate: earning.endDate ?? undefined,
      }),
    onSuccess: (_updated, earning) => {
      toast.success(
        earning.status === "Paused"
          ? "Earning resumed"
          : "Earning paused — upcoming settlements will skip it",
      );
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to update earning"),
  });

  if (isLoading) {
    return (
      <RailSection title="Recurring Earnings" hint="Added each settlement.">
        <Skeleton className="h-12 w-full" />
      </RailSection>
    );
  }

  const list = (earnings ?? []).filter((earning) => earning.status !== "Completed");

  return (
    <RailSection
      title="Recurring Earnings"
      hint="Added automatically each settlement — pause here to skip a period."
      action={
        <Link
          to="/payroll/earnings"
          title="Create or edit earnings on the full page"
          className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-6 px-1.5 text-[10px]")}
        >
          <PlusCircle className="size-3" />
          Manage
        </Link>
      }
    >
      {list.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">No active earnings for this driver.</p>
      ) : (
        <ul className="flex flex-col gap-1">
          {list.map((earning) => (
            <li key={earning.id} className="flex items-center gap-2 rounded-md border p-2">
              <div className="min-w-0 flex-1">
                <p className="truncate text-[11px] font-medium">{earning.description}</p>
                <p className="text-[10px] text-muted-foreground">
                  <AmountDisplay value={earning.amountMinor} currency="USD" /> ·{" "}
                  {earning.frequency === "Monthly" ? "monthly" : "every settlement"}
                  {earning.status === "Paused" && " · paused"}
                </p>
              </div>
              <Button
                size="icon"
                variant="ghost"
                className="size-6"
                disabled={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate(earning)}
                aria-label={earning.status === "Paused" ? "Resume earning" : "Pause earning"}
                title={
                  earning.status === "Paused"
                    ? "Resume — future settlements include it again"
                    : "Pause — future settlements skip it, history is kept"
                }
              >
                {earning.status === "Paused" ? (
                  <Play className="size-3" />
                ) : (
                  <Pause className="size-3" />
                )}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </RailSection>
  );
}

function DeductionsSection({ workerId, onChanged }: { workerId: string; onChanged: () => void }) {
  const { data: deductions, isLoading } = useQuery({
    queryKey: ["worker-recurring-deductions", workerId],
    queryFn: () => fetchWorkerRecurringDeductions(workerId),
  });

  const toggleMutation = useMutation({
    mutationFn: (deduction: RecurringDeductionRow) =>
      updateRecurringDeduction({
        id: deduction.id,
        version: deduction.version,
        workerId: deduction.workerId,
        payCodeId: deduction.payCodeId,
        status: deduction.status === "Paused" ? "Active" : "Paused",
        frequency: deduction.frequency,
        description: deduction.description,
        amountMinor: deduction.amountMinor,
        totalCapMinor: deduction.totalCapMinor ?? undefined,
        escrowAccountId: deduction.escrowAccountId ?? undefined,
        startDate: deduction.startDate,
        endDate: deduction.endDate ?? undefined,
      }),
    onSuccess: (_updated, deduction) => {
      toast.success(
        deduction.status === "Paused"
          ? "Deduction resumed"
          : "Deduction paused — upcoming settlements will skip it",
      );
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to update deduction"),
  });

  if (isLoading) {
    return (
      <RailSection title="Recurring Deductions" hint="Withheld each settlement.">
        <Skeleton className="h-12 w-full" />
      </RailSection>
    );
  }

  const list = (deductions ?? []).filter((deduction) => deduction.status !== "Completed");

  return (
    <RailSection
      title="Recurring Deductions"
      hint="Withheld automatically each settlement — pause here to skip a period."
      action={
        <Link
          to="/payroll/deductions"
          title="Create or edit deductions on the full page"
          className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-6 px-1.5 text-[10px]")}
        >
          <PlusCircle className="size-3" />
          Manage
        </Link>
      }
    >
      {list.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">No active deductions for this driver.</p>
      ) : (
        <ul className="flex flex-col gap-1">
          {list.map((deduction) => (
            <li key={deduction.id} className="flex items-center gap-2 rounded-md border p-2">
              <div className="min-w-0 flex-1">
                <p className="truncate text-[11px] font-medium">{deduction.description}</p>
                <p className="text-[10px] text-muted-foreground">
                  <AmountDisplay value={deduction.amountMinor} currency="USD" /> ·{" "}
                  {deduction.frequency === "Monthly" ? "monthly" : "every settlement"}
                  {deduction.status === "Paused" && " · paused"}
                </p>
              </div>
              <Button
                size="icon"
                variant="ghost"
                className="size-6"
                disabled={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate(deduction)}
                aria-label={deduction.status === "Paused" ? "Resume deduction" : "Pause deduction"}
                title={
                  deduction.status === "Paused"
                    ? "Resume — future settlements withhold it again"
                    : "Pause — future settlements skip it, history is kept"
                }
              >
                {deduction.status === "Paused" ? (
                  <Play className="size-3" />
                ) : (
                  <Pause className="size-3" />
                )}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </RailSection>
  );
}

function AdvancesSection({ workerId }: { workerId: string }) {
  const { data: advances, isLoading } = useQuery({
    queryKey: ["worker-pay-advances", workerId],
    queryFn: () => fetchWorkerPayAdvances(workerId),
  });

  if (isLoading) {
    return (
      <RailSection title="Advances" hint="Recovered automatically from settlements.">
        <Skeleton className="h-10 w-full" />
      </RailSection>
    );
  }

  const outstanding = (advances ?? []).filter(
    (advance) => advance.status === "Outstanding" || advance.status === "PartiallyRecovered",
  );

  return (
    <RailSection
      title="Advances"
      hint="Outstanding balances are recovered automatically on each settlement."
      action={
        <Link
          to="/payroll/advances"
          title="Issue or write off advances on the full page"
          className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-6 px-1.5 text-[10px]")}
        >
          <PlusCircle className="size-3" />
          Manage
        </Link>
      }
    >
      {outstanding.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">No outstanding advances.</p>
      ) : (
        <ul className="flex flex-col gap-1">
          {outstanding.map((advance) => (
            <li key={advance.id} className="flex items-center gap-2 rounded-md border p-2">
              <div className="min-w-0 flex-1">
                <p className="truncate text-[11px] font-medium">
                  {advance.reference || advance.source}
                </p>
                <p className="text-[10px] text-muted-foreground">
                  issued {formatDate(advance.issuedDate)}
                </p>
              </div>
              <span className="text-[11px] font-semibold">
                <AmountDisplay
                  value={advance.amountMinor - advance.recoveredMinor}
                  currency="USD"
                />
              </span>
            </li>
          ))}
        </ul>
      )}
    </RailSection>
  );
}

function EscrowSection({ workerId }: { workerId: string }) {
  const { data: summary, isLoading } = useQuery({
    queryKey: ["worker-earnings-summary", workerId],
    queryFn: () => fetchWorkerEarningsSummary(workerId),
  });

  if (isLoading) {
    return (
      <RailSection title="Escrow" hint="Maintenance reserve balance.">
        <Skeleton className="h-8 w-full" />
      </RailSection>
    );
  }

  return (
    <RailSection
      title="Escrow"
      hint="Reserve funded through settlement contributions; interest accrues per 49 CFR 376.12(k)."
      action={
        <Link
          to="/payroll/escrow-accounts"
          title="Open the escrow ledger"
          className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-6 px-1.5 text-[10px]")}
        >
          View ledger
        </Link>
      }
    >
      <p className="text-sm font-semibold">
        <AmountDisplay value={summary?.escrowBalanceMinor ?? 0} currency="USD" />
      </p>
    </RailSection>
  );
}
