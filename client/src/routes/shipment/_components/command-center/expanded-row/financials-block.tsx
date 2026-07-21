import { getMarginTone, parseDecimal, resolveTargetMarginPct } from "@/lib/profitability";
import { cn, formatCurrency, formatPercent, formatPerMile } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";
import { toneVar } from "../../analytics/kpi/tone";
import { ProfitabilityBreakdownPopover } from "../../profitability/profitability-breakdown-popover";

type FinancialRow = { label: string; value: string; bold?: boolean; tone?: string };

function RowList({ rows }: { rows: FinancialRow[] }) {
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

export function FinancialsBlock({ shipment }: { shipment: Shipment }) {
  const freight = parseDecimal(shipment.freightChargeAmount as unknown as string);
  const other = parseDecimal(shipment.otherChargeAmount as unknown as string);
  const total = parseDecimal(shipment.totalChargeAmount as unknown as string);
  const accessorialsTotal = (shipment.additionalCharges ?? []).reduce(
    (sum, c) => sum + parseDecimal(c.amount as unknown as string) * (c.unit ?? 1),
    0,
  );

  const revenueRows: FinancialRow[] = [
    { label: "Linehaul", value: formatCurrency(freight) },
    { label: "Accessorials", value: formatCurrency(accessorialsTotal) },
    { label: "Other charges", value: formatCurrency(other - accessorialsTotal) },
    { label: "Total revenue", value: formatCurrency(total), bold: true },
  ];

  const estimate = shipment.profitabilityEstimate;
  const hasEstimate = Boolean(estimate && estimate.totalMiles > 0);

  let costRows: FinancialRow[] = [];
  if (estimate && hasEstimate) {
    const profit = parseDecimal(estimate.profit);
    const marginPct =
      estimate.marginPercent !== null && estimate.marginPercent !== undefined
        ? parseDecimal(estimate.marginPercent)
        : null;
    const targetPct = resolveTargetMarginPct(estimate.targetMarginPercent);
    const tone = toneVar(getMarginTone(marginPct ?? (profit < 0 ? -1 : 0), targetPct));

    const rpm = estimate.totalMiles > 0 ? total / estimate.totalMiles : null;

    costRows = [
      {
        label: "Est. cost",
        value: formatCurrency(parseDecimal(estimate.estimatedCost)),
      },
      { label: "Est. profit", value: formatCurrency(profit), tone },
      ...(marginPct !== null
        ? [{ label: "Margin", value: formatPercent(marginPct), tone }]
        : []),
      ...(rpm !== null
        ? [
            {
              label: "RPM vs CPM",
              value: `${formatCurrency(rpm)} vs ${formatCurrency(parseDecimal(estimate.costPerMile))}`,
              tone: toneVar(
                rpm >= parseDecimal(estimate.costPerMile) ? "success" : "danger",
              ),
            },
          ]
        : []),
      ...(estimate.breakEvenRpm !== null && estimate.breakEvenRpm !== undefined
        ? [
            {
              label: "Break-even RPM",
              value: formatPerMile(parseDecimal(estimate.breakEvenRpm)),
            },
          ]
        : []),
    ];
  }

  return (
    <div className="flex flex-col gap-2">
      <RowList rows={revenueRows} />
      {estimate && hasEstimate && (
        <div className="border-t border-border pt-2">
          <div className="mb-1 flex items-center justify-between">
            <span className="text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
              Cost estimate
            </span>
            <ProfitabilityBreakdownPopover
              shipmentId={shipment.id as string}
              align="start"
              trigger={
                <button
                  type="button"
                  className="cursor-pointer text-[10px] font-medium text-primary hover:underline"
                >
                  View breakdown
                </button>
              }
            />
          </div>
          <RowList rows={costRows} />
        </div>
      )}
    </div>
  );
}

export default FinancialsBlock;
