import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ChartConfig } from "@trenova/shared/components/ui/chart";
import { cn } from "@trenova/shared/lib/utils";
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
  color: string;
  dotClass: string;
};

export const AGING_BUCKETS: readonly AgingBucketMeta[] = [
  {
    key: "currentMinor",
    chartKey: "current",
    label: "Current",
    color: "var(--success)",
    dotClass: "bg-success",
  },
  {
    key: "days1To30Minor",
    chartKey: "days1To30",
    label: "1–30",
    color: "var(--warning)",
    dotClass: "bg-warning",
  },
  {
    key: "days31To60Minor",
    chartKey: "days31To60",
    label: "31–60",
    color: "var(--warning-foreground)",
    dotClass: "bg-warning",
  },
  {
    key: "days61To90Minor",
    chartKey: "days61To90",
    label: "61–90",
    color: "var(--danger)",
    dotClass: "bg-danger",
  },
  {
    key: "daysOver90Minor",
    chartKey: "daysOver90",
    label: "90+",
    color: "var(--danger-subtle-foreground)",
    dotClass: "bg-danger",
  },
] as const;

export const agingChartConfig = AGING_BUCKETS.reduce<Record<string, ChartConfig[string]>>(
  (config, bucket) => {
    config[bucket.chartKey] = {
      label: bucket.label,
      color: bucket.color,
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
  const t = useT();

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
              className="text-muted-foreground inline-flex items-center gap-1.5 text-xs"
            >
              <span className={cn("size-2 rounded-full", bucket.dotClass)} />
              {t(bucket.label)} · {share.toFixed(0)}%
            </span>
          );
        })}
      </div>
    </div>
  );
}

export function AgingBadge({ daysPastDue }: { daysPastDue: number }) {
  const t = useT();

  if (daysPastDue <= 0) {
    return <Badge variant="success">{t("Current")}</Badge>;
  }
  if (daysPastDue <= 30) {
    return <Badge variant="warning">{t("{0}d overdue", daysPastDue)}</Badge>;
  }
  return <Badge variant="danger">{t("{0}d overdue", daysPastDue)}</Badge>;
}
