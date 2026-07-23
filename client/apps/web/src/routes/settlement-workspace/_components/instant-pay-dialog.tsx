import { AmountDisplay } from "@/components/accounting/amount-display";
import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { formatUnixDate } from "@/lib/date";
import {
  fetchUnsettledPayEvents,
  payWorkerNow,
  type UnsettledPayEvent,
} from "@/lib/graphql/driver-settlement";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Zap } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

const paymentMethods = ["ACH", "Check", "InstantPay", "Cash", "Other"];

type InstantPayWorker = {
  workerId: string;
  workerName: string;
};

type InstantPayForm = {
  workerId: string;
};

export function InstantPayDialog({
  open,
  onOpenChange,
  worker,
  onPaid,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  worker?: InstantPayWorker | null;
  onPaid: () => void;
}) {
  const form = useForm<InstantPayForm>({
    defaultValues: { workerId: worker?.workerId ?? "" },
  });
  const { control, reset } = form;
  const selectedWorkerId = useWatch({ control, name: "workerId" });
  const workerId = worker?.workerId ?? selectedWorkerId;

  const [checkedIds, setCheckedIds] = useState<Set<string>>(new Set());
  const [applyRecurring, setApplyRecurring] = useState(false);
  const [paymentMethod, setPaymentMethod] = useState("ACH");
  const [paymentReference, setPaymentReference] = useState("");

  useEffect(() => {
    if (open) {
      reset({ workerId: worker?.workerId ?? "" });
      setCheckedIds(new Set());
      setApplyRecurring(false);
      setPaymentMethod("ACH");
      setPaymentReference("");
    }
  }, [open, worker?.workerId, reset]);

  const events = useQuery({
    queryKey: ["unsettled-pay-events", workerId],
    queryFn: () => fetchUnsettledPayEvents(workerId),
    enabled: open && workerId.length > 0,
  });

  useEffect(() => {
    if (events.data) {
      setCheckedIds(new Set(events.data.map((event) => event.id)));
    }
  }, [events.data]);

  const selectedEvents = useMemo(
    () => (events.data ?? []).filter((event) => checkedIds.has(event.id)),
    [events.data, checkedIds],
  );
  const selectedGross = selectedEvents.reduce((sum, event) => sum + event.grossAmountMinor, 0);
  const allSelected =
    (events.data?.length ?? 0) > 0 && selectedEvents.length === events.data?.length;

  const toggleEvent = (eventId: string) => {
    setCheckedIds((current) => {
      const next = new Set(current);
      if (next.has(eventId)) {
        next.delete(eventId);
      } else {
        next.add(eventId);
      }
      return next;
    });
  };

  const payMutation = useMutation({
    mutationFn: () =>
      payWorkerNow({
        workerId,
        payEventIds: allSelected ? undefined : [...checkedIds],
        applyRecurring,
        paymentMethod,
        paymentReference: paymentReference.trim() || undefined,
      }),
    onSuccess: (settlement) => {
      toast.success(`${settlement.settlementNumber} paid`, {
        description: (
          <span>
            Net <AmountDisplay value={settlement.netPayMinor} currency={settlement.currencyCode} />{" "}
            via {settlement.paymentMethod} · posted to the GL and visible to the driver in Dash.
          </span>
        ),
      });
      onPaid();
      onOpenChange(false);
    },
    onError: (error: Error) => toast.error(error.message || "Instant payout failed"),
  });

  const canPay = workerId.length > 0 && selectedEvents.length > 0 && !payMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Pay driver now</DialogTitle>
          <DialogDescription>
            Builds an off-cycle settlement from the selected loads and approves, posts, and marks it
            paid in one pass — the driver sees it in Dash immediately.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          {worker ? (
            <p className="text-sm font-medium">{worker.workerName}</p>
          ) : (
            <FormProvider {...form}>
              <WorkerAutocompleteField<InstantPayForm>
                control={control}
                name="workerId"
                label="Driver"
                rules={{ required: true }}
                placeholder="Select a driver"
              />
            </FormProvider>
          )}

          {workerId.length === 0 ? null : events.isPending ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (events.data?.length ?? 0) === 0 ? (
            <p className="rounded-md border border-dashed p-4 text-center text-xs text-muted-foreground">
              This driver has no payable accrued events. Pay accrues once a load reaches the pay
              trigger milestone; held events must be released first.
            </p>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium">
                  Loads to pay ({selectedEvents.length}/{events.data?.length})
                </p>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 px-2 text-[11px]"
                  onClick={() =>
                    setCheckedIds(
                      allSelected
                        ? new Set()
                        : new Set((events.data ?? []).map((event) => event.id)),
                    )
                  }
                >
                  {allSelected ? "Clear all" : "Select all"}
                </Button>
              </div>
              <ScrollArea className="max-h-56 min-h-0" viewportClassName="min-h-0" maskHeight={18}>
                <ul className="flex flex-col gap-1.5 pr-2">
                  {(events.data ?? []).map((event) => (
                    <EventRow
                      key={event.id}
                      event={event}
                      checked={checkedIds.has(event.id)}
                      onToggle={() => toggleEvent(event.id)}
                    />
                  ))}
                </ul>
              </ScrollArea>

              <div className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2">
                <span className="text-xs font-medium">Gross selected</span>
                <span className="text-sm font-semibold tabular-nums">
                  <AmountDisplay value={selectedGross} currency="USD" />
                </span>
              </div>

              <div className="flex items-start gap-2">
                <Checkbox
                  id="instant-pay-recurring"
                  checked={applyRecurring}
                  onCheckedChange={(checked) => setApplyRecurring(checked === true)}
                />
                <Label htmlFor="instant-pay-recurring" className="text-xs font-normal">
                  Apply recurring deductions, escrow, and advance recovery
                  <span className="mt-0.5 block text-[11px] text-muted-foreground">
                    Off by default so this payout doesn&apos;t double-dip items the regular period
                    settlement will take.
                  </span>
                </Label>
              </div>

              <div>
                <p className="mb-1 text-xs font-medium">Payment method</p>
                <div className="flex flex-wrap gap-2">
                  {paymentMethods.map((method) => (
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
                <p className="mb-1 text-xs font-medium">Payment reference</p>
                <Input
                  value={paymentReference}
                  onChange={(event) => setPaymentReference(event.target.value)}
                  placeholder="ACH trace / check number (optional)"
                />
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={!canPay}
            onClick={() => payMutation.mutate()}
            title="Generates, approves, posts, and marks the settlement paid in one pass"
          >
            <Zap className="size-3.5" />
            {payMutation.isPending ? "Paying..." : "Pay now"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function EventRow({
  event,
  checked,
  onToggle,
}: {
  event: UnsettledPayEvent;
  checked: boolean;
  onToggle: () => void;
}) {
  return (
    <li className="flex items-center gap-2.5 rounded-md border p-2">
      <Checkbox
        checked={checked}
        onCheckedChange={onToggle}
        aria-label={`Pay ${event.proNumber}`}
      />
      <div className="min-w-0 flex-1">
        <p className="truncate font-mono text-xs font-medium">{event.proNumber || "No pro #"}</p>
        <p className="text-[11px] text-muted-foreground">
          {formatUnixDate(event.eventDate)}
          {Number(event.totalMiles) > 0 ? ` · ${Number(event.totalMiles).toFixed(0)} mi` : ""}
        </p>
      </div>
      <span className="shrink-0 text-xs font-semibold tabular-nums">
        <AmountDisplay value={event.grossAmountMinor} currency={event.currencyCode} />
      </span>
    </li>
  );
}
