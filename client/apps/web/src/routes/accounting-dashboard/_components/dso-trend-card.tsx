import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { AR_DSO_TARGET_DAYS } from "@/lib/accounting-constants";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { CartesianGrid, Line, LineChart, ReferenceLine, XAxis, YAxis } from "recharts";
import { RangeToggle } from "./range-toggle";

const dsoChartConfig = {
  dso: {
    label: "DSO (days)",
    theme: { light: "#2a78d6", dark: "#3987e5" },
  },
} satisfies ChartConfig;

function shortDate(unixSeconds: number) {
  return new Date(unixSeconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

export function DsoTrendCard() {
  const [range, setRange] = useState<number>(13);
  const { data: trend, isLoading } = useQuery(queries.ar.dsoTrend(range));

  const chartData = useMemo(
    () =>
      (trend ?? []).map((point) => ({
        label: shortDate(point.periodEnd),
        dso: Number(point.dsoDays.toFixed(1)),
        billed: point.billedMinor,
      })),
    [trend],
  );

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">DSO trend</CardTitle>
        <RangeToggle value={range} onChange={setRange} />
      </CardHeader>
      <CardContent className="p-4">
        {isLoading ? (
          <Skeleton className="h-56 w-full" />
        ) : chartData.length === 0 ? (
          <div className="flex h-56 items-center justify-center text-sm text-muted-foreground">
            No billing activity yet
          </div>
        ) : (
          <ChartContainer config={dsoChartConfig} className="h-56 w-full">
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
                tickFormatter={(value: number) => `${value}d`}
              />
              <ChartTooltip content={<ChartTooltipContent />} />
              <ReferenceLine
                y={AR_DSO_TARGET_DAYS}
                stroke="var(--muted-foreground)"
                strokeDasharray="4 4"
                strokeOpacity={0.5}
                label={{
                  value: `target ${AR_DSO_TARGET_DAYS}d`,
                  position: "insideTopRight",
                  fontSize: 10,
                  fill: "var(--muted-foreground)",
                }}
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
        )}
      </CardContent>
    </Card>
  );
}
