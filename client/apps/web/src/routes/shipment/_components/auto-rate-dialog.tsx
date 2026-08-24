import { apiService } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Separator } from "@trenova/shared/components/ui/separator";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ContractRate, Shipment } from "@trenova/shared/types/shipment";
import { SparklesIcon } from "lucide-react";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";

/**
 * Re-rates a shipment from its contract, on request.
 *
 * A contract prices a shipment once, when it is created, and everything it
 * produced becomes an ordinary editable field. This is the way back: it
 * discards whatever is there and asks the agreement again. Because it
 * overwrites work somebody may have done by hand, it says what it is about to
 * replace before it does it, and what it applied afterwards.
 */
export function AutoRateDialog() {
  const { getValues, setValue } = useFormContext<Shipment>();
  const queryClient = useQueryClient();

  const [open, setOpen] = useState(false);
  const [applied, setApplied] = useState<ContractRate | null>(null);

  const shipmentId = getValues("id");
  const rateLocked = Boolean(getValues("rateLocked"));

  const mutation = useMutation({
    mutationFn: () => apiService.shipmentService.autoRate(shipmentId ?? ""),
    onSuccess: ({ shipment, contractRate }) => {
      if (!contractRate.applied) {
        toast.error("No contract covers this lane", {
          description: "Nothing was changed. Write a rate agreement for it and try again.",
        });
        setOpen(false);
        return;
      }

      // Only the rating-owned fields come back into the open form, so unsaved
      // edits elsewhere survive the re-rate landing.
      setValue("formulaTemplateId", shipment.formulaTemplateId, { shouldDirty: false });
      setValue("baseRate", shipment.baseRate, { shouldDirty: false });
      setValue("additionalCharges", shipment.additionalCharges, { shouldDirty: false });
      setValue("freightChargeAmount", shipment.freightChargeAmount, { shouldDirty: false });
      setValue("otherChargeAmount", shipment.otherChargeAmount, { shouldDirty: false });
      setValue("totalChargeAmount", shipment.totalChargeAmount, { shouldDirty: false });
      setValue("ratingDetail", shipment.ratingDetail, { shouldDirty: false });
      setValue("autoRated", shipment.autoRated, { shouldDirty: false });
      setValue("autoRatedAt", shipment.autoRatedAt, { shouldDirty: false });
      setValue("rateAgreementId", shipment.rateAgreementId, { shouldDirty: false });
      setValue("rateAgreementRuleId", shipment.rateAgreementRuleId, { shouldDirty: false });
      setValue("rateQuoteId", shipment.rateQuoteId, { shouldDirty: false });
      setValue("rateOverrideAmount", shipment.rateOverrideAmount, { shouldDirty: false });
      setValue("rateOverrideReason", shipment.rateOverrideReason, { shouldDirty: false });
      setValue("version", shipment.version, { shouldDirty: false });

      void queryClient.invalidateQueries({ queryKey: ["shipment"] });
      void queryClient.invalidateQueries({ queryKey: ["rateQuote"] });

      setApplied(contractRate);
      setOpen(false);
    },
    onError: () => {
      toast.error("The shipment could not be re-rated", {
        description: "Please try again or contact your system administrator.",
      });
    },
  });

  if (!shipmentId || rateLocked) {
    return null;
  }

  return (
    <>
      <Button type="button" size="xxxs" onClick={() => setOpen(true)}>
        <span className="text-2xs">Re-Apply Rate</span>
      </Button>

      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        pending={mutation.isPending}
        onConfirm={() => mutation.mutate()}
      />

      <AppliedDialog rate={applied} onClose={() => setApplied(null)} />
    </>
  );
}

function ConfirmDialog({
  open,
  onOpenChange,
  pending,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pending: boolean;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-110">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <SparklesIcon className="size-4" />
            Re-rate from contract
          </DialogTitle>
          <DialogDescription>
            The rate agreement covering this lane will replace the rating method, the base rate and
            every charge the contract applies automatically. Anything you have set by hand on those
            fields is discarded.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={onConfirm} disabled={pending}>
            {pending ? "Re-rating…" : "Re-rate"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** The account of what the contract just applied. */
function AppliedDialog({ rate, onClose }: { rate: ContractRate | null; onClose: () => void }) {
  if (!rate) {
    return null;
  }

  const previous = Number(rate.previousLinehaulAmount) || 0;
  const linehaul = Number(rate.linehaulAmount) || 0;
  const change = linehaul - previous;

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-115">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <SparklesIcon className="size-4" />
            Rate applied
          </DialogTitle>
          <DialogDescription>
            {rate.agreementName || "A rate agreement"}
            {rate.ruleLabel ? ` — ${rate.ruleLabel}` : ""} priced this shipment.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 text-sm">
          <AppliedRow label="Rating method" value={rate.formulaTemplateName || "—"} />
          {rate.baseRate ? (
            <AppliedRow label="Base rate" value={formatCurrency(Number(rate.baseRate))} />
          ) : null}
          <AppliedRow label="Freight charges" value={formatCurrency(linehaul)} />
          {previous > 0 && change !== 0 ? (
            <AppliedRow
              label="Change"
              value={`${change > 0 ? "+" : ""}${formatCurrency(change)}`}
            />
          ) : null}

          {rate.accessorials.length > 0 && (
            <>
              <Separator className="my-2" />
              <p className="text-2xs text-muted-foreground">
                Charges the contract applies automatically
              </p>
              {rate.accessorials.map((accessorial) => (
                <div
                  key={accessorial.accessorialChargeId}
                  className="flex items-center justify-between gap-3"
                >
                  <span className="text-muted-foreground flex items-center gap-2">
                    {accessorial.description || accessorial.accessorialChargeId}
                    <Badge variant="outline" className="text-[10px]">
                      {accessorial.method}
                    </Badge>
                  </span>
                  <span className="tabular-nums">
                    {formatCurrency(Number(accessorial.amount) || 0)}
                  </span>
                </div>
              ))}
            </>
          )}

          <Separator className="my-2" />
          <AppliedRow
            label="Contract total"
            value={formatCurrency(Number(rate.totalChargeAmount) || 0)}
            bold
          />
        </div>

        <DialogFooter>
          <Button type="button" onClick={onClose}>
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AppliedRow({ label, value, bold }: { label: string; value: string; bold?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span className={bold ? "font-semibold tabular-nums" : "tabular-nums"}>{value}</span>
    </div>
  );
}
