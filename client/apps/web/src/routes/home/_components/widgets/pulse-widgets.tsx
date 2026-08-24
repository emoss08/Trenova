import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@trenova/shared/components/ui/chart";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts";
import { Link } from "react-router";
import { MetricTile } from "../metric-tile";
import { resolveMetric } from "../metric-values";
import { WidgetEmpty, WidgetNeedsSetup, WidgetShell, WidgetSkeleton } from "../widget-shell";
import type { WidgetProps } from "../widget-registry";

const SHIPMENTS_HREF = "/shipment-management/shipments";
const RECEIVABLES_HREF = "/accounting/ar/aging";

export function KPIWidget({ widget, data }: WidgetProps) {
  const metric = widget.config.metric ? resolveMetric(widget.config.metric, data) : null;

  if (!widget.config.metric) {
    return (
      <WidgetShell title={widget.title || "Metric"} scroll={false}>
        <WidgetNeedsSetup message="Choose the metric this tile shows." />
      </WidgetShell>
    );
  }

  if (!metric) {
    return (
      <WidgetShell title={widget.title || "Metric"} scroll={false}>
        <WidgetSkeleton rows={2} />
      </WidgetShell>
    );
  }

  // A single metric is its own frame: the KPI card already carries a label and
  // a value, so wrapping it in widget chrome would state the name twice.
  return <MetricTile metric={metric} />;
}

export function KPIRowWidget({ widget, data }: WidgetProps) {
  const configured = widget.config.metrics ?? [];
  const metrics = configured
    .map((key) => resolveMetric(key, data))
    .filter((metric): metric is NonNullable<typeof metric> => metric != null);

  if (metrics.length === 0) {
    const loading =
      configured.length > 0 && (data.shipmentAnalyticsLoading || data.receivablesLoading);

    return (
      <WidgetShell title={widget.title || "Metrics"} scroll={false}>
        {loading ? (
          <WidgetSkeleton rows={2} />
        ) : (
          <WidgetNeedsSetup message="Choose the metrics this strip shows." />
        )}
      </WidgetShell>
    );
  }

  return (
    <div
      className="grid h-full min-h-0 gap-2 p-0.5"
      style={{ gridTemplateColumns: `repeat(${metrics.length}, minmax(0, 1fr))` }}
    >
      {metrics.map((metric) => (
        <MetricTile key={metric.key} metric={metric} />
      ))}
    </div>
  );
}

export function ARSnapshotWidget({ widget, data }: WidgetProps) {
  const overview = data.receivables?.overview;

  return (
    <WidgetShell
      title={widget.title || "Receivables"}
      href={RECEIVABLES_HREF}
      hrefLabel="Open aging"
      scroll={false}
    >
      {data.receivablesLoading ? (
        <WidgetSkeleton rows={2} />
      ) : !data.receivables ? (
        <WidgetEmpty>Receivables data is unavailable.</WidgetEmpty>
      ) : (
        <div className="grid flex-1 grid-cols-2 gap-3 sm:grid-cols-4">
          <Figure label="Open" value={formatCurrency((overview?.totalOpenMinor ?? 0) / 100)} />
          <Figure
            label="Overdue"
            value={formatCurrency((overview?.overdueMinor ?? 0) / 100)}
            tone="danger"
          />
          <Figure
            label="Unapplied"
            value={formatCurrency((overview?.unappliedCashMinor ?? 0) / 100)}
          />
          <Figure label="DSO" value={`${(data.receivables.currentDsoDays ?? 0).toFixed(1)}d`} />
        </div>
      )}
    </WidgetShell>
  );
}

function Figure({ label, value, tone }: { label: string; value: string; tone?: "danger" }) {
  return (
    <div className="flex flex-col justify-center gap-1">
      <span className="cc-label text-muted-foreground">{label}</span>
      <span
        className={
          tone === "danger"
            ? "text-destructive font-mono text-base leading-none font-semibold tabular-nums"
            : "font-mono text-base leading-none font-semibold tabular-nums"
        }
      >
        {value}
      </span>
    </div>
  );
}

const REVENUE_CHART_CONFIG = {
  value: { label: "Revenue", color: "var(--brand)" },
} as const;

export function RevenueTrendWidget({ widget, data }: WidgetProps) {
  const points = data.shipmentAnalytics.revenueToday.sparkline;
  // Roughly six labels regardless of how many points the window holds — a
  // label per hour reads as noise at this size.
  const tickInterval = Math.max(Math.ceil(points.length / 6) - 1, 0);

  return (
    <WidgetShell
      title={widget.title || "Revenue & Volume"}
      href={SHIPMENTS_HREF}
      hrefLabel="Open shipments"
      scroll={false}
    >
      {data.shipmentAnalyticsLoading ? (
        <WidgetSkeleton rows={4} />
      ) : points.length === 0 ? (
        <WidgetEmpty>No revenue recorded in this window.</WidgetEmpty>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col gap-2">
          <div className="flex items-baseline gap-2">
            <span className="font-mono text-xl leading-none font-semibold tracking-tight tabular-nums">
              {formatCurrency(data.shipmentAnalytics.revenueToday.total)}
            </span>
            <span className="text-2xs text-muted-foreground">today</span>
          </div>
          <ChartContainer config={REVENUE_CHART_CONFIG} className="min-h-0 w-full flex-1">
            <AreaChart data={points} margin={{ top: 4, right: 12, bottom: 0, left: 12 }}>
              <defs>
                <linearGradient id="home-revenue-fill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--color-value)" stopOpacity={0.22} />
                  <stop offset="100%" stopColor="var(--color-value)" stopOpacity={0.02} />
                </linearGradient>
              </defs>
              <CartesianGrid
                vertical={false}
                stroke="var(--border)"
                strokeOpacity={0.5}
                strokeDasharray="3 3"
              />
              <XAxis
                dataKey="hour"
                tickLine={false}
                axisLine={false}
                tickMargin={6}
                interval={tickInterval}
                tick={{ fontSize: 10 }}
              />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Area
                dataKey="value"
                type="monotone"
                stroke="var(--color-value)"
                fill="url(#home-revenue-fill)"
                strokeWidth={1.5}
              />
            </AreaChart>
          </ChartContainer>
        </div>
      )}
    </WidgetShell>
  );
}

export function OnTimeGoalWidget({ widget, data }: WidgetProps) {
  const onTime = data.shipmentAnalytics.onTimePercent;

  return (
    <WidgetShell
      title={widget.title || "On-Time Goal"}
      href={SHIPMENTS_HREF}
      hrefLabel="Open shipments"
      scroll={false}
    >
      {data.shipmentAnalyticsLoading ? (
        <WidgetSkeleton rows={2} />
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-2">
          <RingGauge
            value={onTime.percent / 100}
            tone={onTime.percent >= onTime.target ? "success" : "warning"}
            size={72}
            aria-label={`On-time ${onTime.percent.toFixed(1)} percent`}
          >
            <span className="font-mono text-sm font-semibold tabular-nums">
              {onTime.percent.toFixed(1)}%
            </span>
          </RingGauge>
          <span className="text-2xs text-muted-foreground">
            Target {onTime.target}% · 7-day {onTime.sevenDayPercent.toFixed(1)}%
          </span>
        </div>
      )}
    </WidgetShell>
  );
}

export function FleetStatusWidget({ widget, data }: WidgetProps) {
  const breakdown = data.shipmentAnalytics.activeShipments.breakdown;
  const rows = [
    { label: "In transit", value: breakdown.inTransit, color: "var(--brand)" },
    { label: "Loading", value: breakdown.loading, color: "var(--info)" },
    { label: "At risk", value: breakdown.atRisk, color: "var(--destructive)" },
    { label: "Delivered", value: breakdown.done, color: "var(--success)" },
  ];
  const total = rows.reduce((sum, row) => sum + row.value, 0);

  return (
    <WidgetShell title={widget.title || "Fleet Status"} href="/equipment/tractors">
      {data.shipmentAnalyticsLoading ? (
        <WidgetSkeleton rows={4} />
      ) : total === 0 ? (
        <WidgetEmpty>Nothing moving right now.</WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-2">
          <div className="bg-muted flex h-1.5 overflow-hidden rounded-full">
            {rows
              .filter((row) => row.value > 0)
              .map((row) => (
                <span
                  key={row.label}
                  style={{ width: `${(row.value / total) * 100}%`, background: row.color }}
                />
              ))}
          </div>
          {rows.map((row) => (
            <div key={row.label} className="flex items-center gap-2">
              <span
                className="size-1.5 shrink-0 rounded-full"
                style={{ background: row.color }}
                aria-hidden
              />
              <span className="min-w-0 flex-1 truncate text-xs">{row.label}</span>
              <span className="font-table text-[10.5px] tabular-nums">{row.value}</span>
            </div>
          ))}
          <Link
            to={SHIPMENTS_HREF}
            className="text-2xs text-muted-foreground hover:text-foreground pt-1"
          >
            {total} active loads
          </Link>
        </div>
      )}
    </WidgetShell>
  );
}
