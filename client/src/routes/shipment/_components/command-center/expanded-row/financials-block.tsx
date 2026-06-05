import { cn, formatCurrency } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";

function parseDecimal(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0;
  if (typeof value === "number") return value;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function FinancialsBlock({ shipment }: { shipment: Shipment }) {
  const freight = parseDecimal(shipment.freightChargeAmount as unknown as string);
  const other = parseDecimal(shipment.otherChargeAmount as unknown as string);
  const total = parseDecimal(shipment.totalChargeAmount as unknown as string);
  const accessorialsTotal = (shipment.additionalCharges ?? []).reduce(
    (sum, c) => sum + parseDecimal(c.amount as unknown as string) * (c.unit ?? 1),
    0,
  );

  const rows: { label: string; value: string; bold?: boolean; tone?: string }[] = [
    { label: "Linehaul", value: formatCurrency(freight) },
    { label: "Accessorials", value: formatCurrency(accessorialsTotal) },
    { label: "Other charges", value: formatCurrency(other - accessorialsTotal) },
    { label: "Total revenue", value: formatCurrency(total), bold: true },
  ];

  return (
    <dl className="grid grid-cols-1 gap-1 text-[11px]">
      {rows.map((row) => (
        <div
          key={row.label}
          className={cn(
            "flex items-center justify-between py-0.75",
            row.bold ? "mt-1 border-t border-border pt-2" : "",
          )}
        >
          <dt className="text-muted-foreground">{row.label}</dt>
          <dd
            className={cn("font-table tabular-nums", row.bold ? "font-semibold" : "font-medium")}
            style={row.tone ? { color: row.tone } : undefined}
          >
            {row.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

export default FinancialsBlock;
