import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader } from "@/components/kpi/kpi-card";
import { KPI_VALUE_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { eventTotals, ratingSegments } from "@/lib/fleet-safety-console";
import type { FleetSafetySummary } from "@/lib/graphql/fleet-safety";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useMemo } from "react";

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
    <KpiStrip>
      <KpiCard span={2}>
        <KpiHeader
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
          <NumberFlow
            value={summary.workers}
            className={KPI_VALUE_CLASS}
            aria-label={t("Drivers")}
          />
        ) : (
          <Skeleton className="h-6 w-10" />
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

      <KpiStripItem
        label={t("Average score")}
        info={
          <InfoPopover title={t("Average score")}>
            {t(
              "Mean safety score across those drivers, out of 100. At risk and watch are the two ratings that need a look.",
            )}
          </InfoPopover>
        }
        value={
          summary ? (
            <span className="flex items-center gap-2">
              <RingGauge
                value={summary.averageScore / 100}
                size={24}
                strokeWidth={3}
                tone="brand"
                aria-label={t("Average safety score out of 100")}
              />
              <span className="flex items-baseline gap-1">
                <NumberFlow value={summary.averageScore} aria-label={t("Average score")} />
                <span className="text-muted-foreground text-xs font-normal">/ 100</span>
              </span>
            </span>
          ) : (
            <Skeleton className="h-6 w-24" />
          )
        }
        sub={
          summary
            ? flagged > 0
              ? t("{0} at risk · {1} on watch", summary.atRisk, summary.watch)
              : t("Nobody at risk or on watch")
            : t("Across every active driver")
        }
      />

      <KpiStripItem
        label={t("Events in window")}
        info={
          <InfoPopover title={t("Events in window")}>
            {t(
              "Safety events dated inside the counting window: accidents, inspections, citations and the rest. Open and preventable are counted separately.",
            )}
          </InfoPopover>
        }
        value={
          summary ? (
            <NumberFlow value={summary.totalEvents} aria-label={t("Events in window")} />
          ) : (
            <Skeleton className="h-6 w-10" />
          )
        }
        sub={summary ? describeEvents(summary.openEvents, totals.preventable) : ""}
      />

      <KpiStripItem
        label={t("Out of service")}
        tone={summary && summary.outOfServiceOrders > 0 ? "danger" : undefined}
        info={
          <InfoPopover title={t("Out of service")}>
            {t("Roadside orders in the window that took a driver or vehicle off the road.")}
          </InfoPopover>
        }
        value={
          summary ? (
            <NumberFlow value={summary.outOfServiceOrders} aria-label={t("Out of service")} />
          ) : (
            <Skeleton className="h-6 w-10" />
          )
        }
        sub={
          summary
            ? summary.outOfServiceOrders > 0
              ? t("{0} points carried across the fleet", summary.totalPoints)
              : t("No order has taken anybody off the road")
            : ""
        }
      />
    </KpiStrip>
  );
}

function describeEvents(open: number, preventable: number): string {
  const parts: string[] = [];
  if (open > 0) parts.push(`${open} open`);
  if (preventable > 0) parts.push(`${preventable} preventable`);
  return parts.length > 0 ? parts.join(" · ") : "Nothing open, nothing preventable";
}
