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

type EDIVolumePoint = ResultOf<typeof EdiVolumeSeriesDocument>["ediVolumeSeries"][number];

const volumeChartConfig = {
  sentCount: {
    label: "Sent",
    theme: { light: "#2a78d6", dark: "#3987e5" },
  },
  failedCount: {
    label: "Failed",
    theme: { light: "#e34948", dark: "#e66767" },
  },
  receivedCount: {
    label: "Received",
    theme: { light: "#1baf7a", dark: "#199e70" },
  },
} satisfies ChartConfig;

const successRateChartConfig = {
  successRate: {
    label: "Success rate",
    theme: { light: "#2a78d6", dark: "#3987e5" },
  },
} satisfies ChartConfig;

function bucketLabel(point: EDIVolumePoint) {
  const date = new Date(point.bucketStart * 1000);
  if (point.bucketSeconds < 24 * 3600) {
    return date.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "numeric",
    });
  }
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export function EDITrendCharts({ points }: { points: EDIVolumePoint[] }) {
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
      <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
        No document activity in the selected time range.
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
      <div className="rounded-md border bg-background p-3">
        <h3 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
          Document volume
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
      <div className="rounded-md border bg-background p-3">
        <h3 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
          Delivery success rate
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
