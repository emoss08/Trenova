import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";

export function PaymentStatsRow() {
  const { data: stats, isLoading } = useQuery(queries.ar.paymentStats());

  if (isLoading || !stats) {
    return (
      <div className="grid grid-cols-3 gap-2.5">
        {Array.from({ length: 3 }).map((_, index) => (
          <Skeleton key={index} className="h-[84px] rounded-md" />
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-3 gap-2.5">
      <StatTile
        index={0}
        label="Posted Today"
        value={formatCurrency(stats.postedTodayMinor / 100)}
        detail={`${stats.postedTodayCount} ${
          stats.postedTodayCount === 1 ? "payment" : "payments"
        }`}
      />
      <StatTile
        index={1}
        label="Unapplied Cash"
        value={formatCurrency(stats.unappliedCashMinor / 100)}
        detail={`${stats.unappliedPaymentCount} ${
          stats.unappliedPaymentCount === 1 ? "payment" : "payments"
        } with remainder`}
        valueClassName={
          stats.unappliedCashMinor > 0 ? "text-amber-600 dark:text-amber-400" : undefined
        }
      />
      <StatTile
        index={2}
        label="Reversed — 30 days"
        value={formatCurrency(stats.reversedLast30Minor / 100)}
        detail={`${stats.reversedLast30Count} ${
          stats.reversedLast30Count === 1 ? "reversal" : "reversals"
        }`}
        valueClassName={
          stats.reversedLast30Count > 0 ? "text-red-600 dark:text-red-400" : undefined
        }
      />
    </div>
  );
}

function StatTile({
  index,
  label,
  value,
  detail,
  valueClassName,
}: {
  index: number;
  label: string;
  value: string;
  detail: string;
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
              "mt-1 text-xl font-semibold tracking-tight tabular-nums",
              valueClassName,
            )}
          >
            {value}
          </p>
          <p className="mt-0.5 text-[11px] text-muted-foreground tabular-nums">{detail}</p>
        </CardContent>
      </Card>
    </m.div>
  );
}
