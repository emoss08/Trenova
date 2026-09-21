import { useT } from "@trenova/shared/i18n/use-t";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import type { EdiVolumeSeriesDocument } from "@trenova/graphql/generated/graphql";
import type { ResultOf } from "@graphql-typed-document-node/core";
import { useMemo } from "react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { formatUnixInUserTimezone, formatUnixMonthDay } from "@trenova/shared/lib/date";

type EDIVolumePoint = ResultOf<typeof EdiVolumeSeriesDocument>["ediVolumeSeries"][number];

const volumeChartConfig = {
  sentCount: {
    label: "Sent",
    color: "var(--info)",
  },
  failedCount: {
    label: "Failed",
    color: "var(--danger)",
  },
  receivedCount: {
    label: "Received",
    color: "var(--success)",
  },
} satisfies ChartConfig;

const successRateChartConfig = {
  successRate: {
    label: "Success rate",
    color: "var(--info)",
  },
} satisfies ChartConfig;

function bucketLabel(point: EDIVolumePoint) {
  if (point.bucketSeconds < 24 * 3600) {
    return formatUnixInUserTimezone(point.bucketStart, {
      month: "short",
      day: "numeric",
      hour: "numeric",
    });
  }
  return formatUnixMonthDay(point.bucketStart);
}

export function EDITrendCharts({ points }: { points: EDIVolumePoint[] }) {
  const t = useT();

  const data = useMemo(
    () =>
      points.map((point) => ({
        ...point,
        label: bucketLabel(point),
        successRate:
          point.sentCount + point.failedCount > 0
            ? Math.round((point.sentCount / (point.sentCount + point.failedCount)) * 1000) / 10
            : null,
      })),
    [points],
  );

  if (data.length === 0) {
    return (
      <div className="bg-background text-muted-foreground rounded-md border p-6 text-sm">
        {t("No document activity in the selected time range.")}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
      <div className="bg-background rounded-md border p-3">
        <h3 className="text-muted-foreground text-xs font-semibold">
          {t("Document volume")}
        </h3>
        <ChartContainer config={volumeChartConfig} className="mt-2 aspect-auto! h-[220px] w-full">
          <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid vertical={false} strokeOpacity={0.35} />
            <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />
            <YAxis tickLine={false} axisLine={false} width={36} allowDecimals={false} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <ChartLegend content={<ChartLegendContent />} />
            <Line
              type="monotone"
              dataKey="sentCount"
              stroke="var(--color-sentCount)"
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
            />
            <Line
              type="monotone"
              dataKey="failedCount"
              stroke="var(--color-failedCount)"
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
            />
            <Line
              type="monotone"
              dataKey="receivedCount"
              stroke="var(--color-receivedCount)"
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
            />
          </LineChart>
        </ChartContainer>
      </div>
      <div className="bg-background rounded-md border p-3">
        <h3 className="text-muted-foreground text-xs font-semibold">
          {t("Delivery success rate")}
        </h3>
        <ChartContainer
          config={successRateChartConfig}
          className="mt-2 aspect-auto! h-[220px] w-full"
        >
          <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid vertical={false} strokeOpacity={0.35} />
            <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />
            <YAxis
              tickLine={false}
              axisLine={false}
              width={36}
              domain={[0, 100]}
              tickFormatter={(value: number) => `${value}%`}
            />
            <ChartTooltip
              content={<ChartTooltipContent formatter={(value) => `${String(value)}%`} />}
            />
            <Line
              type="monotone"
              dataKey="successRate"
              stroke="var(--color-successRate)"
              strokeWidth={2}
              dot={false}
              connectNulls
              isAnimationActive={false}
            />
          </LineChart>
        </ChartContainer>
      </div>
    </div>
  );
}
