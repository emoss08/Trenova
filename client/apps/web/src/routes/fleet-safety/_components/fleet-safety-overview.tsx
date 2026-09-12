import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
import { eventTotals, ratingSegments } from "@/lib/fleet-safety-console";
import type { FleetSafetySummary } from "@/lib/graphql/fleet-safety";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, GaugeIcon, SirenIcon, UsersIcon } from "lucide-react";
import { useMemo } from "react";

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

type FleetSafetyOverviewProps = {
  summary: FleetSafetySummary | undefined;
};

/**
 * The fleet in four numbers: how many drivers and how they are rated, the
 * average score, what happened in the window, and the orders that took
 * somebody off the road. Each is read from the same roll-up the panels below
 * draw, so the strip never disagrees with them.
 */
export function FleetSafetyOverview({ summary }: FleetSafetyOverviewProps) {
  const t = useT();

  const ratings = useMemo(() => ratingSegments(summary?.ratings ?? []), [summary]);
  const totals = useMemo(() => eventTotals(summary?.kinds ?? []), [summary]);
  const flagged = (summary?.atRisk ?? 0) + (summary?.watch ?? 0);

  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      <KpiCard span={2}>
        <KpiHeader
          icon={<UsersIcon className="size-[11px]" />}
          label={t("Drivers")}
          info={
            <InfoPopover title={t("Drivers")}>
              {t(
                "Active drivers in the counting window, with the bar showing how they are rated. Every rating stays on the bar, zero or not.",
              )}
            </InfoPopover>
          }
        />
        {summary ? (
          <NumberFlow value={summary.workers} className={VALUE_CLASS} aria-label={t("Drivers")} />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <CompositionBar
          size="sm"
          className="mt-auto"
          aria-label={t("Drivers by rating")}
          segments={ratings.map((segment) => ({
            key: segment.rating,
            label: segment.label,
            value: segment.workers,
          }))}
        />
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<GaugeIcon className="size-[11px]" />}
          label={t("Average score")}
          info={
            <InfoPopover title={t("Average score")}>
              {t(
                "Mean safety score across those drivers, out of 100. At risk and watch are the two ratings that need a look.",
              )}
            </InfoPopover>
          }
        />
        {summary ? (
          <div className="flex items-center gap-2.5">
            <RingGauge
              value={summary.averageScore / 100}
              size={40}
              strokeWidth={4}
              tone="brand"
              aria-label={t("Average safety score out of 100")}
            />
            <div className="flex items-baseline gap-1">
              <NumberFlow
                value={summary.averageScore}
                className={VALUE_CLASS}
                aria-label={t("Average score")}
              />
              <span className="text-muted-foreground font-mono text-[11px]">/ 100</span>
            </div>
          </div>
        ) : (
          <Skeleton className="h-10 w-24" />
        )}
        <KpiSub>
          {summary
            ? flagged > 0
              ? t("{0} at risk · {1} on watch", summary.atRisk, summary.watch)
              : t("Nobody at risk or on watch")
            : t("Across every active driver")}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<AlertTriangleIcon className="size-[11px]" />}
          label={t("Events in window")}
          info={
            <InfoPopover title={t("Events in window")}>
              {t(
                "Safety events dated inside the counting window: accidents, inspections, citations and the rest. Open and preventable are counted separately.",
              )}
            </InfoPopover>
          }
        />
        {summary ? (
          <NumberFlow
            value={summary.totalEvents}
            className={VALUE_CLASS}
            aria-label={t("Events in window")}
          />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>{summary ? describeEvents(summary.openEvents, totals.preventable) : ""}</KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<SirenIcon className="size-[11px]" />}
          label={t("Out of service")}
          info={
            <InfoPopover title={t("Out of service")}>
              {t("Roadside orders in the window that took a driver or vehicle off the road.")}
            </InfoPopover>
          }
        />
        {summary ? (
          <NumberFlow
            value={summary.outOfServiceOrders}
            className={cn(VALUE_CLASS, summary.outOfServiceOrders > 0 && "text-destructive")}
            aria-label={t("Out of service")}
          />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>
          {summary
            ? summary.outOfServiceOrders > 0
              ? t("{0} points carried across the fleet", summary.totalPoints)
              : t("No order has taken anybody off the road")
            : ""}
        </KpiSub>
      </KpiCard>
    </div>
  );
}

function describeEvents(open: number, preventable: number): string {
  const parts: string[] = [];
  if (open > 0) parts.push(`${open} open`);
  if (preventable > 0) parts.push(`${preventable} preventable`);
  return parts.length > 0 ? parts.join(" · ") : "Nothing open, nothing preventable";
}
