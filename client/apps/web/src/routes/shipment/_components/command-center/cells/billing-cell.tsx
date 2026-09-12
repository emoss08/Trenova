import { ColorOptionValue } from "@/components/fields/select-components";
import { shipmentBillingStatusChoices } from "@/lib/choices";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Shipment } from "@trenova/shared/types/shipment";

export function BillingCell({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const choice = shipmentBillingStatusChoices.find(
    (option) => option.value === shipment.billingTransferStatus,
  );

  if (!choice) {
    return <span className="font-table text-muted-foreground text-[11.5px]">—</span>;
  }

  const label = t(choice.label);

  return (
    <div title={t("Billing queue: {0}", label)}>
      <ColorOptionValue
        value={label}
        color={choice.color}
        className="h-auto"
        textClassName="font-table text-[11.5px]"
      />
    </div>
  );
}
