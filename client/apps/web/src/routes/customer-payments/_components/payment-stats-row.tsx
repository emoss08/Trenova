import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";

export function PaymentStatsRow() {
  const t = useT();

  const { data: stats, isLoading } = useQuery(queries.ar.paymentStats());

  if (isLoading || !stats) {
    return <KpiStripSkeleton count={3} />;
  }

  return (
    <KpiStrip>
      <KpiStripItem
        label={t("Posted today")}
        value={formatCurrency(stats.postedTodayMinor / 100)}
        sub={`${stats.postedTodayCount} ${stats.postedTodayCount === 1 ? "payment" : "payments"}`}
      />
      <KpiStripItem
        label={t("Unapplied cash")}
        value={formatCurrency(stats.unappliedCashMinor / 100)}
        sub={`${stats.unappliedPaymentCount} ${
          stats.unappliedPaymentCount === 1 ? "payment" : "payments"
        } with remainder`}
        tone={stats.unappliedCashMinor > 0 ? "warning" : undefined}
      />
      <KpiStripItem
        label={t("Reversed — 30 days")}
        value={formatCurrency(stats.reversedLast30Minor / 100)}
        sub={`${stats.reversedLast30Count} ${
          stats.reversedLast30Count === 1 ? "reversal" : "reversals"
        }`}
        tone={stats.reversedLast30Count > 0 ? "danger" : undefined}
      />
    </KpiStrip>
  );
}
