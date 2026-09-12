import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
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
  const t = useT();

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
    queryFn: ({ signal }) => fetchUnsettledPayEvents(workerId, { signal }),
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
            {t("Net")} <AmountDisplay value={settlement.netPayMinor} currency={settlement.currencyCode} />{t("via {0} · posted to the GL and visible to the driver in Dash.", settlement.paymentMethod)}
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
          <DialogTitle>{t("Pay driver now")}</DialogTitle>
          <DialogDescription>
            {t("Builds an off-cycle settlement from the selected loads and approves, posts, and marks it paid in one pass — the driver sees it in Dash immediately.")}
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
                label={t("Driver")}
                rules={{ required: true }}
                placeholder={t("Select a driver")}
              />
            </FormProvider>
          )}

          {workerId.length === 0 ? null : events.isPending ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (events.data?.length ?? 0) === 0 ? (
            <p className="text-muted-foreground rounded-md border border-dashed p-4 text-center text-xs">
              {t("This driver has no payable accrued events. Pay accrues once a load reaches the pay trigger milestone; held events must be released first.")}
            </p>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium">
                  {t("Loads to pay ({0}/{1})", selectedEvents.length, events.data?.length)}
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
                  {allSelected ? t("Clear all") : t("Select all")}
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

              <div className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2">
                <span className="text-xs font-medium">{t("Gross selected")}</span>
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
                  {t("Apply recurring deductions, escrow, and advance recovery")}
                  <span className="text-muted-foreground mt-0.5 block text-[11px]">
                    {t("Off by default so this payout doesn't double-dip items the regular period settlement will take.")}
                  </span>
                </Label>
              </div>

              <div>
                <p className="mb-1 text-xs font-medium">{t("Payment method")}</p>
                <div className="flex flex-wrap gap-2">
                  {paymentMethods.map((method) => (
                    <Button
                      key={method}
                      size="sm"
                      variant={paymentMethod === method ? "default" : "outline"}
                      onClick={() => setPaymentMethod(method)}
                    >
                      {method === "InstantPay" ? t("Instant Pay") : method}
                    </Button>
                  ))}
                </div>
              </div>
              <div>
                <p className="mb-1 text-xs font-medium">{t("Payment reference")}</p>
                <Input
                  value={paymentReference}
                  onChange={(event) => setPaymentReference(event.target.value)}
                  placeholder={t("ACH trace / check number (optional)")}
                />
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            disabled={!canPay}
            onClick={() => payMutation.mutate()}
            title={t("Generates, approves, posts, and marks the settlement paid in one pass")}
          >
            <Zap className="size-3.5" />
            {payMutation.isPending ? t("Paying...") : t("Pay now")}
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
  const t = useT();

  return (
    <li className="flex items-center gap-2.5 rounded-md border p-2">
      <Checkbox
        checked={checked}
        onCheckedChange={onToggle}
        aria-label={`Pay ${event.proNumber}`}
      />
      <div className="min-w-0 flex-1">
        <p className="truncate font-mono text-xs font-medium">{event.proNumber || t("No pro #")}</p>
        <p className="text-muted-foreground text-[11px]">
          {formatUnixDate(event.eventDate)}
          {Number(event.totalMiles) > 0 ? t("· {0} mi", Number(event.totalMiles).toFixed(0)) : ""}
        </p>
      </div>
      <span className="shrink-0 text-xs font-semibold tabular-nums">
        <AmountDisplay value={event.grossAmountMinor} currency={event.currencyCode} />
      </span>
    </li>
  );
}
