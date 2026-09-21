import { useT } from "@trenova/shared/i18n/use-t";
import { toneVar } from "@/components/kpi/tone";
import { getMarginTone, parseDecimal, resolveTargetMarginPct } from "@/lib/profitability";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import type { ShipmentProfitabilityQuery } from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatCurrency, formatPercent, formatPerMile } from "@trenova/shared/lib/utils";
import { AlertTriangle } from "lucide-react";
import { useState, type ReactNode } from "react";

const sourceBadges: Record<string, { label: string; className: string }> = {
  Benchmark: { label: "Benchmark", className: "text-2xs" },
  Override: { label: "Override", className: "text-2xs" },
  GLActual: { label: "GL Actual", className: "text-2xs" },
  LiveIndex: { label: "Live Fuel", className: "text-2xs" },
};

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
      <PopoverTrigger render={<button>{trigger}</button>} />
      <PopoverContent align={align} className="w-80 space-y-3">
        {isLoading || !data ? <BreakdownSkeleton /> : <BreakdownContent data={data} />}
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
  const t = useT();

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
          <p className="text-sm font-medium">{t("Cost Estimate")}</p>
          <Badge variant="neutral" className="text-2xs">
            {formatPerMile(parseDecimal(data.profile.totalCpm))}
          </Badge>
        </div>
        <p className="text-muted-foreground mt-0.5 text-xs">
          {t(
            "{0} mi total · {1} loaded · {2} empty",
            data.totalMiles.toFixed(0),
            data.loadedMiles.toFixed(0),
            data.deadheadMiles.toFixed(0),
          )}
        </p>
      </div>

      {data.missingDistance && (
        <Alert variant="warning" size="sm">
          <AlertTriangle />
          <AlertDescription>
            {t(
              "Some moves are missing distance — the estimate only covers moves with a calculated distance.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <DescriptionList layout="split">
        {data.breakdown.map((line) => {
          const badge = sourceBadges[line.effectiveSource] ?? sourceBadges.Benchmark;
          return (
            <DescriptionItem
              key={`${line.category}-${line.name}`}
              numeric
              label={
                <span className="flex min-w-0 items-center gap-1.5">
                  <span className="truncate">{line.name}</span>
                  <Badge variant="neutral" appearance="outline" className={badge.className}>
                    {t(badge.label)}
                  </Badge>
                </span>
              }
            >
              {formatCurrency(parseDecimal(line.amount))}
              <span className="text-foreground-subtle ml-1">
                ({formatPerMile(parseDecimal(line.ratePerMile))})
              </span>
            </DescriptionItem>
          );
        })}
      </DescriptionList>

      <DescriptionList layout="split">
        <DescriptionItem label={t("Estimated cost")} numeric>
          {formatCurrency(parseDecimal(data.estimatedCost))}
        </DescriptionItem>
        <DescriptionItem label={t("Revenue")} numeric>
          {formatCurrency(revenue)}
        </DescriptionItem>
        <DescriptionItem label={t("Profit")} numeric>
          <span style={{ color: toneVar(profitTone) }}>{formatCurrency(profit)}</span>
        </DescriptionItem>
        {marginPct !== null && (
          <DescriptionItem label={t("Margin")} numeric>
            <span style={{ color: toneVar(getMarginTone(marginPct, targetPct)) }}>
              {formatPercent(marginPct)}
            </span>
          </DescriptionItem>
        )}
        {data.breakEvenRpm !== null && data.breakEvenRpm !== undefined && (
          <DescriptionItem label={t("Break-even RPM")} numeric>
            {formatPerMile(parseDecimal(data.breakEvenRpm))}
          </DescriptionItem>
        )}
        {data.revenuePerLoadedMile !== null && data.revenuePerLoadedMile !== undefined && (
          <DescriptionItem label={t("Actual RPM")} numeric>
            {formatPerMile(parseDecimal(data.revenuePerLoadedMile))}
          </DescriptionItem>
        )}
        {fuel && fuel.source === "LiveIndex" && fuel.pricePerGallon && (
          <DescriptionItem label={`Diesel (${fuel.priceDate})`} numeric>
            {`${formatCurrency(parseDecimal(fuel.pricePerGallon))}/gal ÷ ${parseDecimal(fuel.milesPerGallon)} MPG`}
          </DescriptionItem>
        )}
      </DescriptionList>

      <p className="text-2xs text-muted-foreground">
        {t(
          "Estimated from your cost profile as of {0}. Rates come from industry benchmarks unless overridden, mapped to GL actuals, or resolved from a live fuel index.",
          data.profile.asOfDate,
        )}
      </p>
    </>
  );
}
