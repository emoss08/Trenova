import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  AR_CEI_HEALTHY_THRESHOLD,
  AR_CEI_WARNING_THRESHOLD,
  AR_DSO_TARGET_DAYS,
} from "@/lib/accounting-constants";
import type { ARDashboardKpis } from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  BanknoteIcon,
  GaugeIcon,
  TimerIcon,
  TrendingDownIcon,
  TrendingUpIcon,
  WalletIcon,
} from "lucide-react";
import { m } from "motion/react";
import { Link } from "react-router";

export function ARKpiRow() {
  const { data: kpis, isLoading } = useQuery(queries.ar.dashboardKpis());

  if (isLoading || !kpis) {
    return (
      <div className="grid grid-cols-2 gap-2.5 md:grid-cols-3 xl:grid-cols-5">
        {Array.from({ length: 5 }).map((_, index) => (
          <Skeleton key={index} className="h-[104px] w-full rounded-md" />
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-2.5 md:grid-cols-3 xl:grid-cols-5">
      <KpiCard
        index={0}
        icon={WalletIcon}
        label="AR Outstanding"
        value={formatCurrency(kpis.overview.totalOpenMinor / 100)}
        detail={`${kpis.overview.openInvoiceCount} open ${
          kpis.overview.openInvoiceCount === 1 ? "invoice" : "invoices"
        }`}
        to="/accounting/ar/aging"
      />
      <DsoKpiCard kpis={kpis} />
      <CeiKpiCard kpis={kpis} />
      <KpiCard
        index={3}
        icon={BanknoteIcon}
        label="Unapplied Cash"
        value={formatCurrency(kpis.overview.unappliedCashMinor / 100)}
        detail="awaiting application"
        to="/accounting/ar/payments"
      />
      <KpiCard
        index={4}
        icon={AlertTriangleIcon}
        label="Overdue"
        value={`${kpis.overduePercent.toFixed(1)}%`}
        valueClassName={
          kpis.overduePercent >= 25 ? "text-red-600 dark:text-red-400" : undefined
        }
        detail={`${formatCurrency(kpis.overview.overdueMinor / 100)} past due`}
        to="/accounting/ar/open-items"
      />
    </div>
  );
}

function DsoKpiCard({ kpis }: { kpis: ARDashboardKpis }) {
  const delta = kpis.dsoDeltaDays;
  const isUp = delta > 0.05;
  const isDown = delta < -0.05;
  const overTarget = kpis.currentDsoDays > AR_DSO_TARGET_DAYS;

  return (
    <KpiShell index={1} icon={TimerIcon} label="Days Sales Outstanding">
      <div className="flex items-baseline gap-2">
        <p
          className={cn(
            "text-2xl font-semibold tracking-tight tabular-nums",
            overTarget && "text-red-600 dark:text-red-400",
          )}
        >
          {kpis.currentDsoDays.toFixed(1)}d
        </p>
        {(isUp || isDown) && (
          <span
            className={cn(
              "flex items-center gap-0.5 text-xs font-medium tabular-nums",
              isUp && "text-red-600 dark:text-red-400",
              isDown && "text-emerald-600 dark:text-emerald-400",
            )}
          >
            {isUp ? (
              <TrendingUpIcon className="size-3" />
            ) : (
              <TrendingDownIcon className="size-3" />
            )}
            {delta > 0 ? "+" : ""}
            {delta.toFixed(1)}d
          </span>
        )}
      </div>
      <p className="text-[11px] text-muted-foreground">
        target &lt; {AR_DSO_TARGET_DAYS}d · vs 4 weeks ago
      </p>
    </KpiShell>
  );
}

function CeiKpiCard({ kpis }: { kpis: ARDashboardKpis }) {
  const cei = kpis.cei;
  const barClass =
    cei >= AR_CEI_HEALTHY_THRESHOLD
      ? "bg-emerald-500 dark:bg-emerald-400"
      : cei >= AR_CEI_WARNING_THRESHOLD
        ? "bg-amber-500 dark:bg-amber-400"
        : "bg-red-500 dark:bg-red-400";

  return (
    <KpiShell index={2} icon={GaugeIcon} label="Collection Effectiveness">
      <p className="text-2xl font-semibold tracking-tight tabular-nums">{cei.toFixed(0)}%</p>
      <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-muted">
        <m.div
          className={cn("h-full rounded-full", barClass)}
          initial={{ width: 0 }}
          animate={{ width: `${Math.min(cei, 100)}%` }}
          transition={{ duration: 0.6, ease: "easeOut" }}
        />
      </div>
    </KpiShell>
  );
}

function KpiShell({
  index,
  icon: Icon,
  label,
  to,
  children,
}: {
  index: number;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  to?: string;
  children: React.ReactNode;
}) {
  const card = (
    <Card
      className={cn(
        "h-full gap-0 overflow-hidden rounded-md",
        to && "transition-colors hover:bg-muted/40",
      )}
    >
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
        <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
          {label}
        </CardTitle>
        <span className="inline-flex size-7 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon className="size-4" />
        </span>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );

  return (
    <m.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: index * 0.05, ease: "easeOut" }}
    >
      {to ? <Link to={to}>{card}</Link> : card}
    </m.div>
  );
}

function KpiCard({
  index,
  icon,
  label,
  value,
  detail,
  valueClassName,
  to,
}: {
  index: number;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string;
  detail?: string;
  valueClassName?: string;
  to?: string;
}) {
  return (
    <KpiShell index={index} icon={icon} label={label} to={to}>
      <p
        className={cn(
          "text-2xl font-semibold tracking-tight tabular-nums",
          valueClassName,
        )}
      >
        {value}
      </p>
      {detail ? <p className="text-[11px] text-muted-foreground">{detail}</p> : null}
    </KpiShell>
  );
}
