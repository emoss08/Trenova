import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, type ChartConfig } from "@/components/ui/chart";
import { DollarSign } from "lucide-react";
import { Area, AreaChart } from "recharts";
import type { ShipmentAnalyticsData } from "../mock-data";

const chartConfig = {
  value: { label: "Revenue", color: "hsl(142, 71%, 45%)" },
} satisfies ChartConfig;

type Props = {
  data: ShipmentAnalyticsData["revenueToday"];
};

export function RevenueTodayCard({ data }: Props) {
  const { total, sparkline } = data;

  return (
    <Card className="group relative gap-0 overflow-hidden rounded-md border-border/80 pb-0 shadow-none transition-colors hover:border-border">
      <CardHeader className="relative flex flex-row items-start justify-between space-y-0 pb-2">
        <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
          Revenue Today
        </CardTitle>
        <span className="inline-flex size-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
          <DollarSign className="size-4" />
        </span>
      </CardHeader>
      <div className="relative px-4">
        <p className="text-3xl leading-none font-semibold tracking-tight">
          ${total.toLocaleString()}
        </p>
      </div>
      <ChartContainer config={chartConfig} className="mt-auto aspect-auto! h-[40px] w-full">
        <AreaChart data={sparkline} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="revenueFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="hsl(142, 71%, 45%)" stopOpacity={0.3} />
              <stop offset="100%" stopColor="hsl(142, 71%, 45%)" stopOpacity={0.03} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="value"
            stroke="hsl(142, 71%, 45%)"
            strokeWidth={1.5}
            fill="url(#revenueFill)"
            dot={false}
            isAnimationActive={false}
          />
        </AreaChart>
      </ChartContainer>
    </Card>
  );
}
