import { useT } from "@trenova/shared/i18n/use-t";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { SectionPanel } from "@/components/section-panel";
import { AR_DSO_TARGET_DAYS } from "@/lib/accounting-constants";
import { ar } from "@/lib/queries/ar";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { CartesianGrid, Line, LineChart, ReferenceLine, XAxis, YAxis } from "recharts";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { ChartBodySkeleton, PanelChartBody, PanelQuiet } from "./dashboard-panel";
import { DEFAULT_TREND_RANGE, RangeToggle, type TrendRangeWeeks } from "./range-toggle";

const dsoChartConfig = {
  dso: {
    label: "DSO (days)",
    color: "var(--info)",
  },
} satisfies ChartConfig;

export function DsoTrendCard() {
  const t = useT();

  const [range, setRange] = useState<TrendRangeWeeks>(DEFAULT_TREND_RANGE);
  const { data: trend, isLoading } = useQuery(ar.dsoTrend(range));

  const chartData = useMemo(
    () =>
      (trend ?? []).map((point) => ({
        label: formatUnixMonthDay(point.periodEnd),
        dso: Number(point.dsoDays.toFixed(1)),
        billed: point.billedMinor,
      })),
    [trend],
  );

  return (
    <SectionPanel
      title={t("Days sales outstanding")}
      hint={t("target {0}d", AR_DSO_TARGET_DAYS)}
      help={t(
        "How many days of billing are still owed, worked out each week from the open balance and the credit sales behind it. Lower is faster cash.",
      )}
      action={<RangeToggle value={range} onChange={setRange} />}
    >
      {isLoading ? (
        <ChartBodySkeleton label={t("Loading the DSO trend")} />
      ) : chartData.length === 0 ? (
        <PanelQuiet>{t("No billing activity yet")}</PanelQuiet>
      ) : (
        <PanelChartBody>
          <ChartContainer config={dsoChartConfig} className="h-full w-full">
            <LineChart data={chartData} margin={{ left: 4, right: 12, top: 8 }}>
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
                width={40}
                domain={[0, "auto"]}
                tickFormatter={(value: number) => t("{0}d", value)}
              />
              <ChartTooltip content={<ChartTooltipContent />} />
              <ReferenceLine
                y={AR_DSO_TARGET_DAYS}
                stroke="var(--muted-foreground)"
                strokeDasharray="4 4"
                strokeOpacity={0.5}
              />
              <Line
                dataKey="dso"
                type="monotone"
                stroke="var(--color-dso)"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
              />
            </LineChart>
          </ChartContainer>
        </PanelChartBody>
      )}
    </SectionPanel>
  );
}
