"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChargeSplitEditor } from "@/components/billing/charge-split-editor";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ChargeAllocation, Shipment } from "@trenova/shared/types/shipment";
import { useRef } from "react";
import { type Control, type FieldValues, useFormContext, useWatch } from "react-hook-form";

/**
 * Divides the freight charge among payers. Cancelling restores the split that
 * was there when the dialog opened, so a half-typed change never lands on the
 * form.
 */
export function FreightSplitDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const { control, getValues, setValue, trigger } = useFormContext<Shipment>();
  const freightChargeAmount = useWatch({ control, name: "freightChargeAmount" });
  const customerId = useWatch({ control, name: "customerId" });
  const billToCustomerId = useWatch({ control, name: "billToCustomerId" });
  const customer = useWatch({ control, name: "customer" });
  const billToCustomer = useWatch({ control, name: "billToCustomer" });
  const snapshot = useRef<ChargeAllocation[]>([]);

  const payerId = billToCustomerId || customerId;
  const payerLabel = billToCustomerId ? billToCustomer?.name : customer?.name;

  const handleOpenChange = (next: boolean) => {
    if (next) {
      snapshot.current = structuredClone(getValues("freightAllocations") ?? []);
    }
    onOpenChange(next);
  };

  const handleCancel = () => {
    setValue("freightAllocations", snapshot.current, { shouldDirty: true, shouldValidate: true });
    onOpenChange(false);
  };

  const handleSave = async () => {
    const valid = await trigger("freightAllocations");
    if (!valid) return;
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Split freight charge")}</DialogTitle>
          <DialogDescription>
            {t(
              "Divide the {0} freight charge between the customers who pay for it. Each payer receives an invoice for their share.",
              formatCurrency(Number(freightChargeAmount ?? 0)),
            )}
          </DialogDescription>
        </DialogHeader>
        <ChargeSplitEditor
          control={control as unknown as Control<FieldValues>}
          name="freightAllocations"
          chargeAmount={Number(freightChargeAmount ?? 0)}
          defaultPayer={payerId ? { id: payerId, label: payerLabel ?? t("customer") } : null}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleCancel}>
            {t("Cancel")}
          </Button>
          <Button type="button" onClick={() => void handleSave()}>
            {t("Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
