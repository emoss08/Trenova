import { useT } from "@trenova/shared/i18n/use-t";
import { getMarginTone, parseDecimal, resolveTargetMarginPct } from "@/lib/profitability";
import { getTotalMiles } from "@/lib/shipment-utils";
import { cn, formatCurrency, formatPercent, formatPerMile } from "@trenova/shared/lib/utils";
import { additionalChargeLineTotal, type Shipment } from "@trenova/shared/types/shipment";

const MARGIN_TEXT = {
  danger: "text-danger",
  warning: "text-warning",
  success: "text-success",
} as const;

function Figure({
  label,
  value,
  emphasis,
  className,
}: {
  label: string;
  value: string;
  emphasis?: string;
  className?: string;
}) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd
        className={cn(
          "flex items-baseline gap-1 font-mono text-sm whitespace-nowrap tabular-nums",
          className,
        )}
      >
        {value}
        {emphasis ? (
          <small className="text-muted-foreground text-xs font-normal">{emphasis}</small>
        ) : null}
      </dd>
    </div>
  );
}

/** What the load pays and what it costs, on one line. */
export function MoneyLine({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const revenue = parseDecimal(shipment.totalChargeAmount as unknown as string);
  const linehaul = parseDecimal(shipment.freightChargeAmount as unknown as string);
  const charges = shipment.additionalCharges ?? [];
  const isFuel = (charge: (typeof charges)[number]) =>
    !!charge.fuelSurchargeProgramId || !!charge.fuelSurchargeDetail;
  const fuel = charges
    .filter(isFuel)
    .reduce((sum, charge) => sum + (additionalChargeLineTotal(charge) ?? 0), 0);
  const accessorials = Math.max(0, revenue - linehaul - fuel);
  const miles = getTotalMiles(shipment);
  const estimate = shipment.profitabilityEstimate;
  const margin = estimate?.marginPercent != null ? parseDecimal(estimate.marginPercent) : null;
  const marginTone =
    margin != null
      ? getMarginTone(margin, resolveTargetMarginPct(estimate?.targetMarginPercent))
      : null;

  return (
    <dl className="flex flex-wrap gap-x-8 gap-y-3">
      <Figure
        label={t("Revenue")}
        value={formatCurrency(revenue)}
        emphasis={miles > 0 ? formatPerMile(revenue / miles) : undefined}
        className="font-semibold"
      />
      <Figure label={t("Linehaul")} value={formatCurrency(linehaul)} />
      <Figure label={t("Fuel")} value={formatCurrency(fuel)} />
      <Figure label={t("Accessorials")} value={formatCurrency(accessorials)} />
      <Figure
        label={t("Est. cost")}
        value={estimate ? formatCurrency(parseDecimal(estimate.estimatedCost)) : "—"}
      />
      <Figure
        label={t("Margin")}
        value={margin != null ? formatPercent(margin) : "—"}
        className={marginTone ? MARGIN_TEXT[marginTone] : undefined}
      />
    </dl>
  );
}
