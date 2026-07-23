import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCompactCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { Area, AreaChart, CartesianGrid, ReferenceLine, XAxis, YAxis } from "recharts";

const cashFlowChartConfig = {
  actual: {
    label: "Collected",
    theme: { light: "#10b981", dark: "#34d399" },
  },
  expected: {
    label: "Invoiced due",
    theme: { light: "#94a3b8", dark: "#64748b" },
  },
  forecast: {
    label: "Open due (forecast)",
    theme: { light: "#2a78d6", dark: "#3987e5" },
  },
} satisfies ChartConfig;

function shortDate(unixSeconds: number) {
  return new Date(unixSeconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

export function CashFlowForecastCard() {
  const { data: points, isLoading } = useQuery(queries.ar.cashFlowForecast());

  const { chartData, nowLabel } = useMemo(() => {
    const data = (points ?? []).map((point) => ({
      label: shortDate(point.weekStart),
      actual: point.isForecast ? null : point.actualMinor / 100,
      expected: point.expectedMinor / 100,
      forecast: point.isForecast ? point.openDueMinor / 100 : null,
    }));
    const firstForecast = (points ?? []).find((point) => point.isForecast);
    return {
      chartData: data,
      nowLabel: firstForecast ? shortDate(firstForecast.weekStart) : undefined,
    };
  }, [points]);

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">Cash-flow forecast — 90 days</CardTitle>
        <span className="text-xs text-muted-foreground">weekly, collected vs due</span>
      </CardHeader>
      <CardContent className="p-4">
        {isLoading ? (
          <Skeleton className="h-56 w-full" />
        ) : chartData.length === 0 ? (
          <div className="flex h-56 items-center justify-center text-sm text-muted-foreground">
            No receivables activity yet
          </div>
        ) : (
          <ChartContainer config={cashFlowChartConfig} className="h-56 w-full">
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
              {nowLabel ? (
                <ReferenceLine
                  x={nowLabel}
                  stroke="var(--muted-foreground)"
                  strokeDasharray="4 4"
                  strokeOpacity={0.5}
                  label={{
                    value: "now",
                    position: "insideTopLeft",
                    fontSize: 10,
                    fill: "var(--muted-foreground)",
                  }}
                />
              ) : null}
              <Area
                dataKey="expected"
                type="monotone"
                stroke="var(--color-expected)"
                strokeWidth={1.5}
                strokeDasharray="5 3"
                fill="transparent"
              />
              <Area
                dataKey="actual"
                type="monotone"
                stroke="var(--color-actual)"
                strokeWidth={2}
                fill="var(--color-actual)"
                fillOpacity={0.12}
              />
              <Area
                dataKey="forecast"
                type="monotone"
                stroke="var(--color-forecast)"
                strokeWidth={2}
                strokeDasharray="6 3"
                fill="var(--color-forecast)"
                fillOpacity={0.08}
              />
              <ChartLegend content={<ChartLegendContent />} />
            </AreaChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
