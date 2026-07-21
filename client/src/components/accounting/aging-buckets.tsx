import { Badge } from "@/components/ui/badge";
import type { ChartConfig } from "@/components/ui/chart";
import { cn } from "@/lib/utils";
import { m } from "motion/react";

export type AgingBucketKey =
  | "currentMinor"
  | "days1To30Minor"
  | "days31To60Minor"
  | "days61To90Minor"
  | "daysOver90Minor";

export type AgingBucketTotals = Record<AgingBucketKey, number> & {
  totalOpenMinor: number;
};

type AgingBucketMeta = {
  key: AgingBucketKey;
  chartKey: string;
  label: string;
  light: string;
  dark: string;
  dotClass: string;
};

export const AGING_BUCKETS: readonly AgingBucketMeta[] = [
  {
    key: "currentMinor",
    chartKey: "current",
    label: "Current",
    light: "#10b981",
    dark: "#34d399",
    dotClass: "bg-emerald-500 dark:bg-emerald-400",
  },
  {
    key: "days1To30Minor",
    chartKey: "days1To30",
    label: "1–30",
    light: "#f59e0b",
    dark: "#fbbf24",
    dotClass: "bg-amber-500 dark:bg-amber-400",
  },
  {
    key: "days31To60Minor",
    chartKey: "days31To60",
    label: "31–60",
    light: "#f97316",
    dark: "#fb923c",
    dotClass: "bg-orange-500 dark:bg-orange-400",
  },
  {
    key: "days61To90Minor",
    chartKey: "days61To90",
    label: "61–90",
    light: "#ef4444",
    dark: "#f87171",
    dotClass: "bg-red-500 dark:bg-red-400",
  },
  {
    key: "daysOver90Minor",
    chartKey: "daysOver90",
    label: "90+",
    light: "#991b1b",
    dark: "#dc2626",
    dotClass: "bg-red-800 dark:bg-red-600",
  },
] as const;

export const agingChartConfig = AGING_BUCKETS.reduce<Record<string, ChartConfig[string]>>(
  (config, bucket) => {
    config[bucket.chartKey] = {
      label: bucket.label,
      theme: { light: bucket.light, dark: bucket.dark },
    };
    return config;
  },
  {},
) satisfies ChartConfig;

export function AgingDistributionBar({
  totals,
  className,
}: {
  totals: AgingBucketTotals;
  className?: string;
}) {
  const totalOpen = totals.totalOpenMinor;
  if (totalOpen <= 0) return null;

  return (
    <div className={className}>
      <div className="flex h-2.5 w-full gap-px overflow-hidden rounded-full">
        {AGING_BUCKETS.map((bucket, index) => {
          const share = (totals[bucket.key] / totalOpen) * 100;
          if (share <= 0) return null;
          return (
            <m.div
              key={bucket.key}
              className={cn("h-full", bucket.dotClass)}
              initial={{ width: 0 }}
              animate={{ width: `${share}%` }}
              transition={{ duration: 0.5, delay: index * 0.05, ease: "easeOut" }}
            />
          );
        })}
      </div>
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1">
        {AGING_BUCKETS.map((bucket) => {
          const share = (totals[bucket.key] / totalOpen) * 100;
          return (
            <span
              key={bucket.key}
              className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground"
            >
              <span className={cn("size-2 rounded-full", bucket.dotClass)} />
              {bucket.label} · {share.toFixed(0)}%
            </span>
          );
        })}
      </div>
    </div>
  );
}

export function AgingBadge({ daysPastDue }: { daysPastDue: number }) {
  if (daysPastDue <= 0) {
    return <Badge variant="active">Current</Badge>;
  }
  if (daysPastDue <= 30) {
    return <Badge variant="orange">{daysPastDue}d overdue</Badge>;
  }
  return <Badge variant="inactive">{daysPastDue}d overdue</Badge>;
}
