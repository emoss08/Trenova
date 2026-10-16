import { useT } from "@trenova/shared/i18n/use-t";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { SectionPanel } from "@/components/section-panel";
import { ar } from "@/lib/queries/ar";
import { formatCompactCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { Area, AreaChart, CartesianGrid, ReferenceLine, XAxis, YAxis } from "recharts";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { ChartBodySkeleton, PanelChartBody, PanelQuiet } from "./dashboard-panel";

const cashFlowChartConfig = {
  actual: {
    label: "Collected",
    color: "var(--success)",
  },
  expected: {
    label: "Invoiced due",
    color: "var(--foreground-subtle)",
  },
  forecast: {
    label: "Open due (forecast)",
    color: "var(--info)",
  },
  factorAdvances: {
    label: "Factor advances",
    color: "var(--accent-violet)",
  },
  reserveReleases: {
    label: "Reserve releases",
    color: "var(--accent-teal)",
  },
} satisfies ChartConfig;

export function CashFlowForecastCard() {
  const t = useT();

  const { data: points, isLoading } = useQuery(ar.cashFlowForecast());

  const { chartData, factored, nowLabel } = useMemo(() => {
    const data = (points ?? []).map((point) => ({
      label: formatUnixMonthDay(point.weekStart),
      actual: point.isForecast ? null : point.actualMinor / 100,
      expected: point.expectedMinor / 100,
      forecast: point.isForecast ? point.openDueMinor / 100 : null,
      factorAdvances: point.factorAdvancesMinor > 0 ? point.factorAdvancesMinor / 100 : null,
      reserveReleases: point.reserveReleasesMinor > 0 ? point.reserveReleasesMinor / 100 : null,
    }));
    const factored = (points ?? []).some(
      (point) => point.factorAdvancesMinor > 0 || point.reserveReleasesMinor > 0,
    );
    const firstForecast = (points ?? []).find((point) => point.isForecast);
    return {
      chartData: data,
      factored,
      nowLabel: firstForecast ? formatUnixMonthDay(firstForecast.weekStart) : undefined,
    };
  }, [points]);

  return (
    <SectionPanel
      title={t("Cash-flow forecast")}
      hint={t("90 days, weekly")}
      help={t(
        "Collected cash against what fell due each week, then the open balance by the week it is due. Factor advances and reserve releases appear once a factor holds any of it.",
      )}
    >
      {isLoading ? (
        <ChartBodySkeleton label={t("Loading the cash-flow forecast")} />
      ) : chartData.length === 0 ? (
        <PanelQuiet>{t("No receivables activity yet")}</PanelQuiet>
      ) : (
        <PanelChartBody>
          <ChartContainer config={cashFlowChartConfig} className="h-full w-full">
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
                    value: t("now"),
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
              {factored ? (
                <Area
                  dataKey="factorAdvances"
                  type="monotone"
                  stroke="var(--color-factorAdvances)"
                  strokeWidth={1.5}
                  strokeDasharray="2 3"
                  fill="transparent"
                  connectNulls
                />
              ) : null}
              {factored ? (
                <Area
                  dataKey="reserveReleases"
                  type="monotone"
                  stroke="var(--color-reserveReleases)"
                  strokeWidth={1.5}
                  strokeDasharray="2 3"
                  fill="transparent"
                  connectNulls
                />
              ) : null}
              <ChartLegend content={<ChartLegendContent />} />
            </AreaChart>
          </ChartContainer>
        </PanelChartBody>
      )}
    </SectionPanel>
  );
}
