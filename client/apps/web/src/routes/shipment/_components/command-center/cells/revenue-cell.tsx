import { parseDecimal } from "@/lib/profitability";
import { getTotalMiles } from "@/lib/shipment-utils";
import { formatCurrency } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";

export function RevenueCell({ shipment }: { shipment: Shipment }) {
  const total = parseDecimal(shipment.totalChargeAmount as unknown as string | number);
  const miles = getTotalMiles(shipment);
  const rpm = miles > 0 ? total / miles : null;

  return (
    <div className="flex flex-col items-end gap-0.5 text-right">
      <span className="font-table text-[11.5px] font-semibold tabular-nums">
        {formatCurrency(total)}
      </span>
      <span className="font-table text-[9.5px] text-muted-foreground tabular-nums">
        {rpm !== null ? `RPM ${formatCurrency(rpm)}` : "RPM —"}
      </span>
    </div>
  );
}
