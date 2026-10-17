import { useT } from "@trenova/shared/i18n/use-t";
import { AGING_BUCKETS, agingChartConfig } from "@/components/accounting/aging-buckets";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@trenova/shared/components/ui/chart";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { SectionPanel } from "@/components/section-panel";
import { ar } from "@/lib/queries/ar";
import { cn, formatCompactCurrency, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Area, AreaChart, CartesianGrid, Cell, Pie, PieChart, XAxis, YAxis } from "recharts";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import {
  ChartBodySkeleton,
  PANEL_CHART_HEIGHT,
  PanelChartBody,
  PanelQuiet,
} from "./dashboard-panel";
import { DEFAULT_TREND_RANGE, RangeToggle, type TrendRangeWeeks } from "./range-toggle";

type AgingView = "snapshot" | "trend";

export function AgingCard() {
  const t = useT();

  const [view, setView] = useState<AgingView>("snapshot");
  const [range, setRange] = useState<TrendRangeWeeks>(DEFAULT_TREND_RANGE);

  return (
    <SectionPanel
      title={t("Aging")}
      hint={view === "snapshot" ? t("open balance by days past due") : undefined}
      action={
        <div className="flex items-center gap-2">
          {view === "trend" ? <RangeToggle value={range} onChange={setRange} /> : null}
          <SegmentedControl<AgingView>
            aria-label={t("Aging view")}
            value={view}
            onValueChange={setView}
            items={[
              { value: "snapshot", label: t("Snapshot") },
              { value: "trend", label: t("Trend") },
            ]}
          />
        </div>
      }
    >
      {view === "snapshot" ? <AgingSnapshot /> : <AgingTrend range={range} />}
    </SectionPanel>
  );
}

function AgingSnapshot() {
  const t = useT();

  const { data: kpis, isLoading } = useQuery(ar.dashboardKpis());

  const buckets = kpis?.overview.buckets;
  const chartData = useMemo(
    () =>
      AGING_BUCKETS.map((bucket) => ({
        name: bucket.chartKey,
        label: bucket.label,
        value: (buckets?.[bucket.key] ?? 0) / 100,
      })).filter((entry) => entry.value > 0),
    [buckets],
  );

  if (isLoading) {
    return <ChartBodySkeleton label={t("Loading aging")} />;
  }

  const totalOpen = (buckets?.totalOpenMinor ?? 0) / 100;
  if (chartData.length === 0) {
    return <PanelQuiet>{t("Nothing outstanding — all caught up")}</PanelQuiet>;
  }

  return (
    <div className={cn("flex items-center gap-4 p-3", PANEL_CHART_HEIGHT)}>
      <div className="relative h-full flex-1">
        <ChartContainer config={agingChartConfig} className="h-full w-full">
          <PieChart>
            <ChartTooltip content={<ChartTooltipContent nameKey="name" hideLabel />} />
            <Pie
              data={chartData}
              dataKey="value"
              nameKey="name"
              innerRadius="62%"
              outerRadius="88%"
              paddingAngle={2}
              strokeWidth={0}
              isAnimationActive
            >
              {chartData.map((entry) => (
                <Cell key={entry.name} fill={`var(--color-${entry.name})`} />
              ))}
            </Pie>
          </PieChart>
        </ChartContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-lg font-semibold tabular-nums">
            {formatCompactCurrency(totalOpen)}
          </span>
          <span className="text-muted-foreground text-xs">{t("total open")}</span>
        </div>
      </div>
      <div className="w-44 shrink-0 space-y-2">
        {AGING_BUCKETS.map((bucket) => {
          const amount = (buckets?.[bucket.key] ?? 0) / 100;
          const share = totalOpen > 0 ? (amount / totalOpen) * 100 : 0;
          return (
            <div key={bucket.key} className="flex items-center gap-2 text-xs">
              <span className={cn("size-2 shrink-0 rounded-full", bucket.dotClass)} />
              <span className="text-muted-foreground w-10">{t(bucket.label)}</span>
              <span className="flex-1 text-right tabular-nums">{formatCurrency(amount)}</span>
              <span className="text-muted-foreground w-9 text-right tabular-nums">
                {share.toFixed(0)}%
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function AgingTrend({ range }: { range: TrendRangeWeeks }) {
  const t = useT();

  const { data: trend, isLoading } = useQuery(ar.agingTrend(range));

  const chartData = useMemo(
    () =>
      (trend ?? []).map((point) => ({
        label: formatUnixMonthDay(point.periodEnd),
        current: point.buckets.currentMinor / 100,
        days1To30: point.buckets.days1To30Minor / 100,
        days31To60: point.buckets.days31To60Minor / 100,
        days61To90: point.buckets.days61To90Minor / 100,
        daysOver90: point.buckets.daysOver90Minor / 100,
      })),
    [trend],
  );

  if (isLoading) {
    return <ChartBodySkeleton label={t("Loading the aging trend")} />;
  }

  if (chartData.length === 0) {
    return <PanelQuiet>{t("No open receivables history yet")}</PanelQuiet>;
  }

  return (
    <PanelChartBody>
      <ChartContainer config={agingChartConfig} className="h-full w-full">
        <AreaChart data={chartData} margin={{ left: 4, right: 12, top: 8 }}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" />
          <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} minTickGap={32} />
          <YAxis
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            width={52}
            tickFormatter={(value: number) => formatCompactCurrency(value)}
          />
          <ChartTooltip content={<ChartTooltipContent />} />
          {AGING_BUCKETS.map((bucket) => (
            <Area
              key={bucket.chartKey}
              dataKey={bucket.chartKey}
              type="monotone"
              stackId="aging"
              stroke={`var(--color-${bucket.chartKey})`}
              fill={`var(--color-${bucket.chartKey})`}
              fillOpacity={0.35}
              strokeWidth={1}
            />
          ))}
          <ChartLegend content={<ChartLegendContent />} />
        </AreaChart>
      </ChartContainer>
    </PanelChartBody>
  );
}
