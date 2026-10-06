import { useT } from "@trenova/shared/i18n/use-t";
import { getMarginTone, parseDecimal, resolveTargetMarginPct } from "@/lib/profitability";
import { formatCurrency, formatPercent } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { toneVar } from "@/components/kpi/tone";
import { ProfitabilityBreakdownPopover } from "../../profitability/profitability-breakdown-popover";

export function MarginCell({ shipment }: { shipment: Shipment }) {
  const t = useT();

  const estimate = shipment.profitabilityEstimate;

  if (!estimate || estimate.totalMiles <= 0) {
    return (
      <div className="flex flex-col items-end gap-0.5 text-right">
        <span className="font-table text-muted-foreground text-xs tabular-nums">—</span>
      </div>
    );
  }

  const marginPct =
    estimate.marginPercent !== null && estimate.marginPercent !== undefined
      ? parseDecimal(estimate.marginPercent)
      : null;
  const profit = parseDecimal(estimate.profit);
  const tone = getMarginTone(
    marginPct ?? (profit < 0 ? -1 : 0),
    resolveTargetMarginPct(estimate.targetMarginPercent),
  );

  return (
    <ProfitabilityBreakdownPopover
      shipmentId={shipment.id as string}
      trigger={
        <div
          className="flex cursor-pointer flex-col items-end gap-0.5 text-right"
          onClick={(event) => event.stopPropagation()}
        >
          <span
            className="font-table text-xs font-semibold tabular-nums"
            style={{ color: toneVar(tone) }}
          >
            {marginPct !== null ? formatPercent(marginPct) : "—"}
          </span>
          <span className="font-table text-muted-foreground text-3xs tabular-nums">
            {t("CPM {0}", formatCurrency(parseDecimal(estimate.costPerMile)))}
          </span>
        </div>
      }
    />
  );
}
