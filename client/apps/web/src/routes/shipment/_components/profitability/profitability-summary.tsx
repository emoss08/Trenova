import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { getMarginTone, parseDecimal, resolveTargetMarginPct } from "@/lib/profitability";
import { queries } from "@/lib/queries";
import { formatCurrency, formatPerMile } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { toneVar } from "../analytics/kpi/tone";
import { MarginPill } from "./margin-pill";
import { ProfitabilityBreakdownPopover } from "./profitability-breakdown-popover";

function StatCell({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="text-2xs text-muted-foreground">{label}</span>
      <span className="truncate text-sm font-medium tabular-nums">{children}</span>
    </div>
  );
}

export function ProfitabilitySummary({ shipmentId }: { shipmentId: string }) {
  const { data, isLoading } = useQuery({
    ...queries.shipment.profitability(shipmentId),
    staleTime: 60_000,
  });

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-3 rounded-md border border-border bg-muted/40 p-2 sm:grid-cols-4">
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
      </div>
    );
  }

  if (!data || data.totalMiles <= 0) {
    return null;
  }

  const profit = parseDecimal(data.profit);
  const marginPct = data.marginPercent !== null ? parseDecimal(data.marginPercent) : null;
  const targetPct = resolveTargetMarginPct(data.profile.targetMarginPercent);
  const profitTone = getMarginTone(marginPct ?? (profit < 0 ? -1 : targetPct), targetPct);

  return (
    <div className="rounded-md border border-border bg-muted/40 p-2">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          Profitability estimate
        </span>
        <ProfitabilityBreakdownPopover
          shipmentId={shipmentId}
          trigger={
            <button
              type="button"
              className="cursor-pointer text-2xs font-medium text-primary hover:underline"
            >
              View breakdown
            </button>
          }
        />
      </div>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatCell label={`Est. cost (${formatPerMile(parseDecimal(data.profile.totalCpm))})`}>
          {formatCurrency(parseDecimal(data.estimatedCost))}
        </StatCell>
        <StatCell label="Est. profit">
          <span style={{ color: toneVar(profitTone) }}>{formatCurrency(profit)}</span>
        </StatCell>
        <StatCell label="Margin">
          {marginPct !== null ? (
            <MarginPill
              marginPct={marginPct}
              targetMarginPercent={data.profile.targetMarginPercent}
            />
          ) : (
            "—"
          )}
        </StatCell>
        <StatCell label="RPM vs break-even">
          {data.revenuePerLoadedMile !== null &&
            data.revenuePerLoadedMile !== undefined &&
            data.breakEvenRpm !== null &&
            data.breakEvenRpm !== undefined ? (
            <span
              style={{
                color: toneVar(
                  parseDecimal(data.revenuePerLoadedMile) >= parseDecimal(data.breakEvenRpm)
                    ? "success"
                    : "danger",
                ),
              }}
            >
              {`${formatCurrency(parseDecimal(data.revenuePerLoadedMile))} vs ${formatCurrency(parseDecimal(data.breakEvenRpm))}`}
            </span>
          ) : (
            "—"
          )}
        </StatCell>
      </div>
    </div>
  );
}
