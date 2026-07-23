import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  AR_CEI_HEALTHY_THRESHOLD,
  AR_CEI_WARNING_THRESHOLD,
} from "@/lib/accounting-constants";
import type { ARCollectionPerformance } from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";

export function CollectionsPerformanceCard() {
  const { data: performance, isLoading } = useQuery(queries.ar.collectionPerformance());

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">Collections performance</CardTitle>
        <span className="text-xs text-muted-foreground">trailing 91 days</span>
      </CardHeader>
      <CardContent className="p-4">
        {isLoading || !performance ? (
          <Skeleton className="h-56 w-full" />
        ) : (
          <PerformanceBody performance={performance} />
        )}
      </CardContent>
    </Card>
  );
}

function PerformanceBody({ performance }: { performance: ARCollectionPerformance }) {
  const totals = performance.totals;
  const collectedShare =
    totals.creditSalesMinor > 0
      ? Math.min((totals.collectedMinor / totals.creditSalesMinor) * 100, 100)
      : 0;
  const ceiClass =
    performance.cei >= AR_CEI_HEALTHY_THRESHOLD
      ? "text-emerald-600 dark:text-emerald-400"
      : performance.cei >= AR_CEI_WARNING_THRESHOLD
        ? "text-amber-600 dark:text-amber-400"
        : "text-red-600 dark:text-red-400";
  const ceiBarClass =
    performance.cei >= AR_CEI_HEALTHY_THRESHOLD
      ? "bg-emerald-500 dark:bg-emerald-400"
      : performance.cei >= AR_CEI_WARNING_THRESHOLD
        ? "bg-amber-500 dark:bg-amber-400"
        : "bg-red-500 dark:bg-red-400";

  return (
    <div className="flex h-56 flex-col justify-between">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <p className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
            Collection Effectiveness
          </p>
          <p className={cn("mt-1 text-3xl font-semibold tracking-tight tabular-nums", ceiClass)}>
            {performance.cei.toFixed(0)}%
          </p>
          <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <m.div
              className={cn("h-full rounded-full", ceiBarClass)}
              initial={{ width: 0 }}
              animate={{ width: `${Math.min(performance.cei, 100)}%` }}
              transition={{ duration: 0.6, ease: "easeOut" }}
            />
          </div>
        </div>
        <div>
          <p className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
            Avg Days to Pay
          </p>
          <p className="mt-1 text-3xl font-semibold tracking-tight tabular-nums">
            {totals.avgDaysToPay.toFixed(1)}d
          </p>
          <p className="mt-2 text-[11px] text-muted-foreground tabular-nums">
            {totals.applicationCount} applications in period
          </p>
        </div>
      </div>

      <div>
        <div className="flex items-baseline justify-between text-xs">
          <span className="text-muted-foreground">Collected vs invoiced</span>
          <span className="font-medium tabular-nums">
            {formatCurrency(totals.collectedMinor / 100)} /{" "}
            {formatCurrency(totals.creditSalesMinor / 100)}
          </span>
        </div>
        <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <m.div
            className="h-full rounded-full bg-emerald-500 dark:bg-emerald-400"
            initial={{ width: 0 }}
            animate={{ width: `${collectedShare}%` }}
            transition={{ duration: 0.6, ease: "easeOut" }}
          />
        </div>
      </div>

      <div className="grid grid-cols-3 divide-x rounded-md border bg-muted/30">
        <RateStat
          label="Write-off"
          value={`${(performance.writeOffRatio * 100).toFixed(1)}%`}
          detail={formatCurrency(totals.shortPayMinor / 100)}
          alert={performance.writeOffRatio > 0.02}
        />
        <RateStat
          label="Short-pay rate"
          value={`${(performance.shortPayRate * 100).toFixed(1)}%`}
          detail={`${totals.shortPayApplicationCount} of ${totals.applicationCount || 0}`}
          alert={performance.shortPayRate > 0.1}
        />
        <RateStat
          label="Dispute rate"
          value={`${(performance.disputeRate * 100).toFixed(1)}%`}
          detail={`${totals.disputedInvoiceCount} invoices`}
          alert={performance.disputeRate > 0.05}
        />
      </div>
    </div>
  );
}

function RateStat({
  label,
  value,
  detail,
  alert,
}: {
  label: string;
  value: string;
  detail: string;
  alert: boolean;
}) {
  return (
    <div className="px-3 py-2.5">
      <p className="text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <p
        className={cn(
          "mt-0.5 text-lg font-semibold tabular-nums",
          alert && "text-red-600 dark:text-red-400",
        )}
      >
        {value}
      </p>
      <p className="text-[10px] text-muted-foreground tabular-nums">{detail}</p>
    </div>
  );
}
