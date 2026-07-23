import {
  AGING_BUCKETS,
  agingChartConfig,
} from "@/components/accounting/aging-buckets";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@trenova/shared/components/ui/chart";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCompactCurrency, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Area, AreaChart, CartesianGrid, Cell, Pie, PieChart, XAxis, YAxis } from "recharts";
import { RangeToggle } from "./range-toggle";

type AgingView = "snapshot" | "trend";

function shortDate(unixSeconds: number) {
  return new Date(unixSeconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

export function AgingCard() {
  const [view, setView] = useState<AgingView>("snapshot");

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">Aging</CardTitle>
        <div className="flex gap-1">
          {(["snapshot", "trend"] as const).map((option) => (
            <Button
              key={option}
              type="button"
              size="sm"
              variant={view === option ? "secondary" : "ghost"}
              onClick={() => setView(option)}
              className="h-7 px-2 text-xs capitalize"
            >
              {option}
            </Button>
          ))}
        </div>
      </CardHeader>
      <CardContent className="p-4">
        {view === "snapshot" ? <AgingSnapshot /> : <AgingTrend />}
      </CardContent>
    </Card>
  );
}

function AgingSnapshot() {
  const { data: kpis, isLoading } = useQuery(queries.ar.dashboardKpis());

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
    return <Skeleton className="h-56 w-full" />;
  }

  const totalOpen = (buckets?.totalOpenMinor ?? 0) / 100;
  if (chartData.length === 0) {
    return (
      <div className="flex h-56 items-center justify-center text-sm text-muted-foreground">
        Nothing outstanding — all caught up
      </div>
    );
  }

  return (
    <div className="flex h-56 items-center gap-4">
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
          <span className="text-[11px] text-muted-foreground">total open</span>
        </div>
      </div>
      <div className="w-44 shrink-0 space-y-2">
        {AGING_BUCKETS.map((bucket) => {
          const amount = (buckets?.[bucket.key] ?? 0) / 100;
          const share = totalOpen > 0 ? (amount / totalOpen) * 100 : 0;
          return (
            <div key={bucket.key} className="flex items-center gap-2 text-xs">
              <span className={`size-2 shrink-0 rounded-full ${bucket.dotClass}`} />
              <span className="w-10 text-muted-foreground">{bucket.label}</span>
              <span className="flex-1 text-right font-medium tabular-nums">
                {formatCurrency(amount)}
              </span>
              <span className="w-9 text-right text-muted-foreground tabular-nums">
                {share.toFixed(0)}%
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function AgingTrend() {
  const [range, setRange] = useState<number>(13);
  const { data: trend, isLoading } = useQuery(queries.ar.agingTrend(range));

  const chartData = useMemo(
    () =>
      (trend ?? []).map((point) => ({
        label: shortDate(point.periodEnd),
        current: point.buckets.currentMinor / 100,
        days1To30: point.buckets.days1To30Minor / 100,
        days31To60: point.buckets.days31To60Minor / 100,
        days61To90: point.buckets.days61To90Minor / 100,
        daysOver90: point.buckets.daysOver90Minor / 100,
      })),
    [trend],
  );

  if (isLoading) {
    return <Skeleton className="h-56 w-full" />;
  }

  if (chartData.length === 0) {
    return (
      <div className="flex h-56 items-center justify-center text-sm text-muted-foreground">
        No open receivables history yet
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex justify-end">
        <RangeToggle value={range} onChange={setRange} />
      </div>
      <ChartContainer config={agingChartConfig} className="h-[196px] w-full">
        <AreaChart data={chartData} margin={{ left: 4, right: 12, top: 8 }}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" />
          <XAxis
            dataKey="label"
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            minTickGap={32}
          />
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
    </div>
  );
}
