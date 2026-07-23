import type { ShipmentProfitabilityQuery } from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { getMarginTone, parseDecimal, resolveTargetMarginPct } from "@/lib/profitability";
import { queries } from "@/lib/queries";
import { formatCurrency, formatPercent, formatPerMile } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { useState, type ReactNode } from "react";
import { toneVar } from "../analytics/kpi/tone";

const sourceBadges: Record<string, { label: string; className: string }> = {
  Benchmark: { label: "Benchmark", className: "text-2xs" },
  Override: { label: "Override", className: "text-2xs" },
  GLActual: { label: "GL Actual", className: "text-2xs" },
  LiveIndex: { label: "Live Fuel", className: "text-2xs" },
};

function DetailRow({
  label,
  value,
  valueStyle,
}: {
  label: string;
  value: string;
  valueStyle?: React.CSSProperties;
}) {
  return (
    <div className="flex items-center justify-between gap-4 text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums" style={valueStyle}>
        {value}
      </span>
    </div>
  );
}

export function ProfitabilityBreakdownPopover({
  shipmentId,
  trigger,
  align = "end",
}: {
  shipmentId: string;
  trigger: ReactNode;
  align?: "start" | "center" | "end";
}) {
  const [open, setOpen] = useState(false);

  const { data, isLoading } = useQuery({
    ...queries.shipment.profitability(shipmentId),
    enabled: open,
    staleTime: 60_000,
  });

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<span>{trigger}</span>} />
      <PopoverContent align={align} className="w-80 space-y-3">
        {isLoading || !data ? (
          <BreakdownSkeleton />
        ) : (
          <BreakdownContent data={data} />
        )}
      </PopoverContent>
    </Popover>
  );
}

function BreakdownSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-5 w-40" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-2/3" />
    </div>
  );
}

type ProfitabilityData = ShipmentProfitabilityQuery["shipmentProfitability"];

function BreakdownContent({ data }: { data: ProfitabilityData }) {
  const revenue = parseDecimal(data.revenue);
  const profit = parseDecimal(data.profit);
  const marginPct = data.marginPercent !== null ? parseDecimal(data.marginPercent) : null;
  const targetPct = resolveTargetMarginPct(data.profile.targetMarginPercent);
  const profitTone = getMarginTone(marginPct ?? (profit < 0 ? -1 : targetPct), targetPct);
  const fuel = data.profile.fuel;

  return (
    <>
      <div>
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium">Cost Estimate</p>
          <Badge variant="secondary" className="text-2xs">
            {formatPerMile(parseDecimal(data.profile.totalCpm))}
          </Badge>
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {data.totalMiles.toFixed(0)} mi total · {data.loadedMiles.toFixed(0)} loaded ·{" "}
          {data.deadheadMiles.toFixed(0)} empty
        </p>
      </div>

      {data.missingDistance && (
        <div className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-xs text-amber-700 dark:text-amber-400">
          <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
          <span>
            Some moves are missing distance — the estimate only covers moves with a calculated
            distance.
          </span>
        </div>
      )}

      <div className="space-y-1.5">
        {data.breakdown.map((line) => {
          const badge = sourceBadges[line.effectiveSource] ?? sourceBadges.Benchmark;
          return (
            <div key={`${line.category}-${line.name}`} className="flex items-center justify-between gap-2 text-xs">
              <span className="flex min-w-0 items-center gap-1.5">
                <span className="truncate text-muted-foreground">{line.name}</span>
                <Badge variant="outline" className={badge.className}>
                  {badge.label}
                </Badge>
              </span>
              <span className="shrink-0 font-medium tabular-nums">
                {formatCurrency(parseDecimal(line.amount))}
                <span className="ml-1 text-muted-foreground">
                  ({formatPerMile(parseDecimal(line.ratePerMile))})
                </span>
              </span>
            </div>
          );
        })}
      </div>

      <Separator />

      <div className="space-y-1.5">
        <DetailRow label="Estimated cost" value={formatCurrency(parseDecimal(data.estimatedCost))} />
        <DetailRow label="Revenue" value={formatCurrency(revenue)} />
        <DetailRow
          label="Profit"
          value={formatCurrency(profit)}
          valueStyle={{ color: toneVar(profitTone) }}
        />
        {marginPct !== null && (
          <DetailRow
            label="Margin"
            value={formatPercent(marginPct)}
            valueStyle={{ color: toneVar(getMarginTone(marginPct, targetPct)) }}
          />
        )}
        {data.breakEvenRpm !== null && data.breakEvenRpm !== undefined && (
          <DetailRow
            label="Break-even RPM"
            value={formatPerMile(parseDecimal(data.breakEvenRpm))}
          />
        )}
        {data.revenuePerLoadedMile !== null && data.revenuePerLoadedMile !== undefined && (
          <DetailRow
            label="Actual RPM"
            value={formatPerMile(parseDecimal(data.revenuePerLoadedMile))}
          />
        )}
        {fuel && fuel.source === "LiveIndex" && fuel.pricePerGallon && (
          <DetailRow
            label={`Diesel (${fuel.priceDate})`}
            value={`${formatCurrency(parseDecimal(fuel.pricePerGallon))}/gal ÷ ${parseDecimal(fuel.milesPerGallon)} MPG`}
          />
        )}
      </div>

      <p className="text-2xs text-muted-foreground">
        Estimated from your cost profile as of {data.profile.asOfDate}. Rates come from industry
        benchmarks unless overridden, mapped to GL actuals, or resolved from a live fuel index.
      </p>
    </>
  );
}
