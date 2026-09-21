import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { SectionPanel } from "@/components/section-panel";
import { AR_CEI_HEALTHY_THRESHOLD, AR_CEI_WARNING_THRESHOLD } from "@/lib/accounting-constants";
import type { ARCollectionPerformance } from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";

export function CollectionsPerformanceCard() {
  const t = useT();

  const { data: performance, isLoading } = useQuery(queries.ar.collectionPerformance());

  return (
    <SectionPanel title={t("Collections performance")} hint={t("trailing 91 days")}>
      <div className="p-4">
        {isLoading || !performance ? (
          <Skeleton className="h-56 w-full" />
        ) : (
          <PerformanceBody performance={performance} />
        )}
      </div>
    </SectionPanel>
  );
}

function PerformanceBody({ performance }: { performance: ARCollectionPerformance }) {
  const t = useT();

  const totals = performance.totals;
  const collectedShare =
    totals.creditSalesMinor > 0
      ? Math.min((totals.collectedMinor / totals.creditSalesMinor) * 100, 100)
      : 0;
  const ceiClass =
    performance.cei >= AR_CEI_HEALTHY_THRESHOLD
      ? "text-success-foreground"
      : performance.cei >= AR_CEI_WARNING_THRESHOLD
        ? "text-warning-foreground"
        : "text-danger-foreground";
  const ceiBarClass =
    performance.cei >= AR_CEI_HEALTHY_THRESHOLD
      ? "bg-success"
      : performance.cei >= AR_CEI_WARNING_THRESHOLD
        ? "bg-warning"
        : "bg-danger";

  return (
    <div className="flex h-56 flex-col justify-between">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <p className="text-muted-foreground text-xs font-medium">
            {t("Collection effectiveness")}
          </p>
          <p className={cn("mt-1 text-2xl font-semibold tabular-nums", ceiClass)}>
            {performance.cei.toFixed(0)}%
          </p>
          <div className="bg-muted mt-2 h-1.5 w-full overflow-hidden rounded-full">
            <div
              className={cn("h-full rounded-full", ceiBarClass)}
              style={{ width: `${Math.min(performance.cei, 100)}%` }}
            />
          </div>
        </div>
        <div>
          <p className="text-muted-foreground text-xs font-medium">{t("Avg days to pay")}</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">
            {t("{0}d", totals.avgDaysToPay.toFixed(1))}
          </p>
          <p className="text-muted-foreground mt-2 text-xs tabular-nums">
            {t("{0} applications in period", totals.applicationCount)}
          </p>
        </div>
      </div>

      <div>
        <div className="flex items-baseline justify-between text-xs">
          <span className="text-muted-foreground">{t("Collected vs invoiced")}</span>
          <span className="font-medium tabular-nums">
            {formatCurrency(totals.collectedMinor / 100)} /{" "}
            {formatCurrency(totals.creditSalesMinor / 100)}
          </span>
        </div>
        <div className="bg-muted mt-1.5 h-1.5 w-full overflow-hidden rounded-full">
          <div className="bg-success h-full rounded-full" style={{ width: `${collectedShare}%` }} />
        </div>
      </div>

      <div className="bg-muted/30 grid grid-cols-3 divide-x rounded-md border">
        <RateStat
          label={t("Write-off")}
          value={`${(performance.writeOffRatio * 100).toFixed(1)}%`}
          detail={formatCurrency(totals.shortPayMinor / 100)}
          alert={performance.writeOffRatio > 0.02}
        />
        <RateStat
          label={t("Short-pay rate")}
          value={`${(performance.shortPayRate * 100).toFixed(1)}%`}
          detail={`${totals.shortPayApplicationCount} of ${totals.applicationCount || 0}`}
          alert={performance.shortPayRate > 0.1}
        />
        <RateStat
          label={t("Dispute rate")}
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
      <p className="text-muted-foreground text-xs font-medium">{label}</p>
      <p
        className={cn(
          "mt-0.5 text-lg font-semibold tabular-nums",
          alert && "text-danger-foreground",
        )}
      >
        {value}
      </p>
      <p className="text-muted-foreground text-2xs tabular-nums">{detail}</p>
    </div>
  );
}
