import { useT } from "@trenova/shared/i18n/use-t";
import { parseDecimal } from "@/lib/profitability";
import { getTotalMiles } from "@/lib/shipment-utils";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";

export function RevenueCell({ shipment }: { shipment: Shipment }) {
  const t = useT();

  const total = parseDecimal(shipment.totalChargeAmount as unknown as string | number);
  const miles = getTotalMiles(shipment);
  const rpm = miles > 0 ? total / miles : null;

  return (
    <div className="flex flex-col items-end gap-0.5 text-right">
      <span className="font-table text-[11.5px] font-semibold tabular-nums">
        {formatCurrency(total)}
      </span>
      <span className="font-table text-muted-foreground text-[9.5px] tabular-nums">
        {rpm !== null ? t("RPM {0}", formatCurrency(rpm)) : t("RPM —")}
      </span>
    </div>
  );
}
