import { apiService } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import {
  NumberFieldGroup,
  NumberFieldInput,
  NumberField as NumberFieldRoot,
} from "@trenova/shared/components/ui/number-field";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import type { Shipment } from "@trenova/shared/types/shipment";
import { PencilRulerIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";

/**
 * Set or clear a hand-set rate.
 *
 * The override fields are system-owned everywhere else — a plain save can
 * neither set nor clear one — so this dialog is the whole write path. The
 * shipment is re-rated on the spot, and the quote records what the contract
 * would have charged instead, which is the rate leakage report.
 */
export function RateOverrideDialog() {
  const { getValues, setValue } = useFormContext<Shipment>();
  const queryClient = useQueryClient();

  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState<number | undefined>();
  const [reason, setReason] = useState("");
  const [locked, setLocked] = useState(false);
  const [amountMissing, setAmountMissing] = useState(false);

  const shipmentId = getValues("id");
  const hasOverride = Boolean(getValues("rateOverrideAmount"));

  useEffect(() => {
    if (open) {
      const current = getValues("rateOverrideAmount");
      setAmount(current ? Number(current) : undefined);
      setReason(getValues("rateOverrideReason") ?? "");
      setLocked(Boolean(getValues("rateLocked")));
      setAmountMissing(false);
    }
  }, [open, getValues]);

  const mutation = useMutation({
    mutationFn: (payload: {
      amount?: number;
      reason?: string;
      rateLocked?: boolean;
      clear?: boolean;
    }) => apiService.shipmentService.setRateOverride(shipmentId ?? "", payload),
    onSuccess: (updated, variables) => {
      // Only the rating-owned fields come back into the open form, so unsaved
      // edits elsewhere survive the override landing.
      setValue("rateOverrideAmount", updated.rateOverrideAmount, { shouldDirty: false });
      setValue("rateOverrideReason", updated.rateOverrideReason, { shouldDirty: false });
      setValue("rateLocked", updated.rateLocked, { shouldDirty: false });
      setValue("freightChargeAmount", updated.freightChargeAmount, { shouldDirty: false });
      setValue("totalChargeAmount", updated.totalChargeAmount, { shouldDirty: false });
      setValue("ratingDetail", updated.ratingDetail, { shouldDirty: false });
      setValue("version", updated.version, { shouldDirty: false });

      void queryClient.invalidateQueries({ queryKey: ["shipment"] });
      void queryClient.invalidateQueries({ queryKey: ["rateQuote"] });

      toast.success(variables.clear ? "Rate override removed" : "Rate override applied");
      setOpen(false);
    },
    onError: () => {
      toast.error("The rate override could not be saved", {
        description: "Please try again or contact your system administrator.",
      });
    },
  });

  if (!shipmentId) {
    return null;
  }

  const apply = () => {
    if (amount === undefined || Number.isNaN(amount)) {
      setAmountMissing(true);
      return;
    }

    mutation.mutate({ amount, reason: reason.trim(), rateLocked: locked });
  };

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-6 gap-1 px-1.5"
        onClick={() => setOpen(true)}
      >
        <PencilRulerIcon className="size-3" />
        <span className="text-2xs">{hasOverride ? "Overridden" : "Override"}</span>
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <PencilRulerIcon className="size-4" />
              Override Rate
            </DialogTitle>
            <DialogDescription>
              The hand-set amount replaces the contract&apos;s linehaul and survives every re-rate.
              The quote keeps recording what the contract would have charged, so the difference
              stays visible.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 py-2">
            <div className="space-y-1.5">
              <label htmlFor="override-amount" className="text-xs font-medium">
                Linehaul Amount
              </label>
              <NumberFieldRoot
                value={amount}
                onValueChange={(value) => {
                  setAmount(value ?? undefined);
                  setAmountMissing(false);
                }}
                min={0}
                size="sm"
              >
                <NumberFieldGroup>
                  <NumberFieldInput id="override-amount" className="text-right" />
                </NumberFieldGroup>
              </NumberFieldRoot>
              {amountMissing && (
                <p className="text-2xs text-destructive">An override amount is required</p>
              )}
            </div>

            <div className="space-y-1.5">
              <label htmlFor="override-reason" className="text-xs font-medium">
                Reason
              </label>
              <Textarea
                id="override-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="e.g. Negotiated spot rate with the customer"
                minRows={2}
                maxRows={4}
              />
              <p className="text-2xs text-muted-foreground">
                Kept on the quote for the audit trail. Your organization may require one.
              </p>
            </div>

            <div className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2">
              <div>
                <p className="text-xs font-medium">Lock the Rate</p>
                <p className="mt-0.5 text-2xs text-muted-foreground">
                  Freezes every charge against re-rating — for a shipment already invoiced.
                </p>
              </div>
              <Switch checked={locked} onCheckedChange={setLocked} />
            </div>
          </div>

          <DialogFooter>
            {hasOverride && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="mr-auto text-destructive"
                disabled={mutation.isPending}
                onClick={() => mutation.mutate({ clear: true })}
              >
                Remove Override
              </Button>
            )}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setOpen(false)}
              disabled={mutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={apply}
              isLoading={mutation.isPending}
              loadingText="Applying..."
            >
              Apply Override
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
