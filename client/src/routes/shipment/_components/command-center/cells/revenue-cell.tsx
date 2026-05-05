import { formatCurrency } from "@/lib/utils";
import { getTotalMiles } from "@/lib/shipment-utils";
import type { Shipment } from "@/types/shipment";

function parseDecimal(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0;
  if (typeof value === "number") return value;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

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
