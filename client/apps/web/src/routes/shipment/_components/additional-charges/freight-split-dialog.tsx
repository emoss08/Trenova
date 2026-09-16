"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChargeSplitDialog } from "@/components/billing/charge-split-dialog";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useFormContext, useWatch } from "react-hook-form";
import { useShipmentDefaultPayer } from "../use-shipment-default-payer";

/** Divides the freight charge among payers. */
export function FreightSplitDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const { control } = useFormContext<Shipment>();
  const freightChargeAmount = useWatch({ control, name: "freightChargeAmount" });
  const defaultPayer = useShipmentDefaultPayer();

  return (
    <ChargeSplitDialog
      open={open}
      onOpenChange={onOpenChange}
      name="freightAllocations"
      chargeAmount={Number(freightChargeAmount ?? 0)}
      defaultPayer={defaultPayer}
      title={t("Split freight charge")}
      description={t(
        "Divide the {0} freight charge between the customers who pay for it. Each payer receives an invoice for their share.",
        formatCurrency(Number(freightChargeAmount ?? 0)),
      )}
    />
  );
}
