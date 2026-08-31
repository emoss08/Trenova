import { KpiCard, KpiHeader } from "@/components/kpi/kpi-card";
import { ChartContainer, type ChartConfig } from "@trenova/shared/components/ui/chart";
import { Activity } from "lucide-react";
import { Area, AreaChart } from "recharts";
import type { ApiKeyAnalyticsData } from "../analytics-data";

const chartConfig = {
  value: { label: "Requests", color: "hsl(142, 71%, 45%)" },
} satisfies ChartConfig;

type Props = {
  data: ApiKeyAnalyticsData["requests30d"];
};

export function RequestsCard({ data }: Props) {
  const { total, sparkline } = data;

  return (
    <KpiCard span={2}>
      <KpiHeader icon={<Activity className="size-[11px]" />} label="Requests (30d)" />
      <div className="relative px-4">
        <p className="text-3xl leading-none font-semibold tracking-tight">
          {total.toLocaleString()}
        </p>
      </div>
      {sparkline.length > 1 ? (
        <ChartContainer config={chartConfig} className="mt-auto aspect-auto! h-4 w-full">
          <AreaChart data={sparkline} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="requestsFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="hsl(142, 71%, 45%)" stopOpacity={0.3} />
                <stop offset="100%" stopColor="hsl(142, 71%, 45%)" stopOpacity={0.03} />
              </linearGradient>
            </defs>
            <Area
              type="monotone"
              dataKey="value"
              stroke="hsl(142, 71%, 45%)"
              strokeWidth={1.5}
              fill="url(#requestsFill)"
              dot={false}
              isAnimationActive={false}
            />
          </AreaChart>
        </ChartContainer>
      ) : (
        <div className="mt-auto h-[40px]" />
      )}
    </KpiCard>
  );
}
