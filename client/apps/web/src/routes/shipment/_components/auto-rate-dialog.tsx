import { BillView } from "@/components/tariff-bill/bill-view";
import { apiService } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { storedChargesBill } from "@trenova/shared/lib/tariff-lines";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ContractRate, Shipment } from "@trenova/shared/types/shipment";
import { readStoredTariffBill } from "@trenova/shared/types/tariff-form";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

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
  const t = useT();

  const { control, setValue } = useFormContext<Shipment>();
  const queryClient = useQueryClient();

  const [open, setOpen] = useState(false);
  const [applied, setApplied] = useState<{ rate: ContractRate; currency: string } | null>(null);

  const shipmentId = useWatch({ control, name: "id" });
  const rateLocked = Boolean(useWatch({ control, name: "rateLocked" }));

  const mutation = useMutation({
    mutationFn: () => apiService.shipmentService.autoRate(shipmentId ?? ""),
    onSuccess: ({ shipment, contractRate }) => {
      if (!contractRate.applied) {
        toast.error(t("No contract covers this lane"), {
          description: t("Nothing was changed. Write a rate agreement for it and try again."),
        });
        setOpen(false);
        return;
      }

      // Only the rating-owned fields come back into the open form, so unsaved
      // edits elsewhere survive the re-rate landing.
      setValue("formulaTemplateId", shipment.formulaTemplateId, { shouldDirty: false });
      setValue("tariffInputs", shipment.tariffInputs, { shouldDirty: false });
      setValue("freightLines", shipment.freightLines, { shouldDirty: false });
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

      // The bill is in the rating currency the shipment's rating detail records.
      setApplied({
        rate: contractRate,
        currency: readStoredTariffBill(shipment.ratingDetail?.bill)?.currency || "USD",
      });
      setOpen(false);
    },
    onError: () => {
      toast.error(t("The shipment could not be re-rated"), {
        description: t("Please try again or contact your system administrator."),
      });
    },
  });

  if (!shipmentId || rateLocked) {
    return null;
  }

  return (
    <>
      <Button type="button" variant="link" size="xxxs" onClick={() => setOpen(true)}>
        <span className="text-2xs">{t("Re-apply rate")}</span>
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        pending={mutation.isPending}
        onConfirm={() => mutation.mutate()}
      />

      {applied ? (
        <AppliedDialog
          rate={applied.rate}
          currency={applied.currency}
          onClose={() => setApplied(null)}
        />
      ) : null}
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
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AssistMark className="size-4" />
            {t("Re-rate from contract")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "The rate agreement covering this lane will replace the rating method, the tariff's input values and every charge the contract applies automatically. Anything you have set by hand on those fields is discarded.",
            )}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="button" onClick={onConfirm} disabled={pending}>
            {pending ? t("Re-rating…") : t("Re-rate")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** The account of what the contract just applied. */
function AppliedDialog({
  rate,
  currency,
  onClose,
}: {
  rate: ContractRate;
  currency: string;
  onClose: () => void;
}) {
  const t = useT();

  // An agreement's accessorial has no label of its own; it is named as the catalog names it.
  const contractBill = storedChargesBill(
    {
      ...rate,
      charges: rate.charges.map((charge) => ({
        ...charge,
        label: charge.label || charge.description || charge.code,
      })),
    },
    currency,
  );
  const previous = Number(rate.previousFreightAmount) || 0;
  const freight = Number(rate.freightAmount) || 0;
  const change = freight - previous;

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AssistMark className="size-4" />
            {t("Rate applied")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "{0}{1} priced this shipment.",
              rate.agreementName || t("A rate agreement"),
              rate.ruleLabel ? ` — ${rate.ruleLabel}` : "",
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 text-sm">
          <AppliedRow label={t("Tariff")} value={rate.formulaTemplateName || "—"} />
          {contractBill ? (
            <div className="rounded-md border p-2">
              <BillView bill={contractBill} currency={currency} />
            </div>
          ) : null}
          {previous > 0 && change !== 0 ? (
            <AppliedRow
              label={t("Freight change")}
              value={`${change > 0 ? "+" : ""}${formatCurrency(change, currency)}`}
            />
          ) : null}
        </div>

        <DialogFooter>
          <Button type="button" onClick={onClose}>
            {t("Done")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AppliedRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span className="tabular-nums">{value}</span>
    </div>
  );
}
