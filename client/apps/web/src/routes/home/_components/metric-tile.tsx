import { KpiGoalBar } from "@/components/kpi/kpi-goal-bar";
import { KpiHero } from "@/components/kpi/kpi-hero";
import { KpiRing } from "@/components/kpi/kpi-ring";
import { KpiStat } from "@/components/kpi/kpi-stat";
import { Link } from "react-router";
import type { MetricValue } from "./metric-values";

/**
 * The KPI cards bring their own frame and fixed height, which the widget tile
 * already supplies. Stripping both lets a metric fill whatever the author sized
 * its tile to instead of leaving a border inside a border.
 */
const FILL = "h-full !border-0 bg-transparent";

const EMPTY_MILE_SCALE_MAX = 20;

export function MetricTile({ metric }: { metric: MetricValue }) {
  const shared = {
    label: metric.label,
    value: metric.display,
    unit: metric.unit,
    delta: metric.delta,
    deltaLabel: metric.deltaLabel,
    deltaTone: metric.deltaTone,
    sub: metric.sub,
    className: FILL,
  } as const;

  const body = (() => {
    switch (metric.shape) {
      case "hero":
        return <KpiHero {...shared} sparkData={metric.sparkline} sparkColor="var(--brand)" />;
      case "ring":
        return <KpiRing {...shared} ringValue={metric.raw} target={metric.target} />;
      case "goal":
        return (
          <KpiGoalBar
            {...shared}
            actual={metric.raw}
            target={metric.target ?? 0}
            max={EMPTY_MILE_SCALE_MAX}
          />
        );
      case "stat":
        return <KpiStat {...shared} tone={metric.deltaTone ?? "brand"} />;
    }
  })();

  // Every number on the home screen is a door: the whole tile is the link, not
  // a footer the eye has to find.
  return (
    <Link to={metric.href} className="flex min-h-0 flex-1 flex-col" aria-label={metric.label}>
      {body}
    </Link>
  );
}
