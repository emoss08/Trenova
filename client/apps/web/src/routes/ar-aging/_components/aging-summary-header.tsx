import {
  AgingDistributionBar,
  type AgingBucketTotals,
} from "@/components/accounting/aging-buckets";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { AR_DSO_TARGET_DAYS } from "@/lib/accounting-constants";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";

export function AgingSummaryHeader({
  totals,
  isLoading,
}: {
  totals: AgingBucketTotals | undefined;
  isLoading: boolean;
}) {
  const { data: kpis } = useQuery(queries.ar.dashboardKpis());

  if (isLoading || !totals) {
    return (
      <div className="grid gap-2.5 lg:grid-cols-5">
        {Array.from({ length: 5 }).map((_, index) => (
          <Skeleton key={index} className="h-[88px] rounded-md" />
        ))}
      </div>
    );
  }

  const totalOpen = totals.totalOpenMinor;
  const currentShare = totalOpen > 0 ? (totals.currentMinor / totalOpen) * 100 : 0;
  const overdueShare = totalOpen > 0 ? 100 - currentShare : 0;

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2.5 md:grid-cols-3 lg:grid-cols-5">
        <SummaryTile index={0} label="Total Open" value={formatCurrency(totalOpen / 100)} />
        <SummaryTile
          index={1}
          label="Current"
          value={`${currentShare.toFixed(1)}%`}
          detail={formatCurrency(totals.currentMinor / 100)}
          valueClassName="text-emerald-600 dark:text-emerald-400"
        />
        <SummaryTile
          index={2}
          label="Overdue"
          value={`${overdueShare.toFixed(1)}%`}
          detail={formatCurrency((totalOpen - totals.currentMinor) / 100)}
          valueClassName={
            overdueShare > 0 ? "text-red-600 dark:text-red-400" : undefined
          }
        />
        <SummaryTile
          index={3}
          label="Current DSO"
          value={kpis ? `${kpis.currentDsoDays.toFixed(1)}d` : "—"}
          detail={`target < ${AR_DSO_TARGET_DAYS}d`}
        />
        <SummaryTile
          index={4}
          label="CEI"
          value={kpis ? `${kpis.cei.toFixed(0)}%` : "—"}
          detail="trailing 90 days"
        />
      </div>

      {totalOpen > 0 ? (
        <Card className="gap-0 rounded-md p-0">
          <CardHeader className="px-4 pt-3 pb-2">
            <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
              Distribution
            </CardTitle>
          </CardHeader>
          <CardContent className="px-4 pb-3">
            <AgingDistributionBar totals={totals} />
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

function SummaryTile({
  index,
  label,
  value,
  detail,
  valueClassName,
}: {
  index: number;
  label: string;
  value: string;
  detail?: string;
  valueClassName?: string;
}) {
  return (
    <m.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: index * 0.05, ease: "easeOut" }}
    >
      <Card className="h-full gap-0 rounded-md py-3">
        <CardContent className="px-4">
          <p className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
            {label}
          </p>
          <p
            className={cn(
              "mt-1 text-2xl font-semibold tracking-tight tabular-nums",
              valueClassName,
            )}
          >
            {value}
          </p>
          {detail ? (
            <p className="mt-0.5 text-[11px] text-muted-foreground tabular-nums">{detail}</p>
          ) : null}
        </CardContent>
      </Card>
    </m.div>
  );
}
