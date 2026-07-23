import { AgingDistributionBar } from "@/components/accounting/aging-buckets";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import type { ARCustomerProfile } from "@/lib/graphql/accounts-receivable";
import { cn, formatCurrency } from "@/lib/utils";
import { m } from "motion/react";
import { useMemo } from "react";
import { Bar, BarChart, XAxis } from "recharts";

const collectionsChartConfig = {
  collected: {
    label: "Collected",
    theme: { light: "#10b981", dark: "#34d399" },
  },
} satisfies ChartConfig;

function monthLabel(unixSeconds: number) {
  return new Date(unixSeconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    timeZone: "UTC",
  });
}

function formatDateOrDash(unixSeconds: number) {
  if (!unixSeconds) return "—";
  return new Date(unixSeconds * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function CustomerSnapshotHeader({
  profile,
  isLoading,
}: {
  profile: ARCustomerProfile | undefined;
  isLoading: boolean;
}) {
  const chartData = useMemo(
    () =>
      (profile?.snapshot.monthlyCollections ?? []).map((point) => ({
        label: monthLabel(point.monthStart),
        collected: point.amountMinor / 100,
      })),
    [profile],
  );

  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-[92px] rounded-md" />
          ))}
        </div>
        <div className="grid gap-3 xl:grid-cols-2">
          <Skeleton className="h-40 rounded-md" />
          <Skeleton className="h-40 rounded-md" />
        </div>
      </div>
    );
  }

  if (!profile) return null;

  const snapshot = profile.snapshot;
  const utilization = profile.creditUtilization;
  const utilizationPct = Math.min(utilization * 100, 100);
  const utilizationBarClass =
    utilization >= 1
      ? "bg-red-500 dark:bg-red-400"
      : utilization >= 0.75
        ? "bg-amber-500 dark:bg-amber-400"
        : "bg-emerald-500 dark:bg-emerald-400";

  const score = profile.delinquencyScore;
  const scoreClass =
    score >= 60
      ? "text-red-600 dark:text-red-400"
      : score >= 30
        ? "text-amber-600 dark:text-amber-400"
        : "text-emerald-600 dark:text-emerald-400";

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
        <SnapshotTile index={0} label="Open Balance">
          <p className="text-2xl font-semibold tracking-tight tabular-nums">
            {formatCurrency(snapshot.totalOpenMinor / 100)}
          </p>
          <p className="mt-0.5 text-[11px] text-muted-foreground tabular-nums">
            {formatCurrency(snapshot.overdueMinor / 100)} overdue ·{" "}
            {snapshot.openInvoiceCount} open
          </p>
        </SnapshotTile>
        <SnapshotTile index={1} label="Credit Utilization">
          {snapshot.hasCreditLimit && snapshot.creditLimitMinor > 0 ? (
            <>
              <p className="text-2xl font-semibold tracking-tight tabular-nums">
                {(utilization * 100).toFixed(0)}%
              </p>
              <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-muted">
                <m.div
                  className={cn("h-full rounded-full", utilizationBarClass)}
                  initial={{ width: 0 }}
                  animate={{ width: `${utilizationPct}%` }}
                  transition={{ duration: 0.6, ease: "easeOut" }}
                />
              </div>
              <p className="mt-1 text-[11px] text-muted-foreground tabular-nums">
                of {formatCurrency(snapshot.creditLimitMinor / 100)} limit
              </p>
            </>
          ) : (
            <>
              <p className="text-2xl font-semibold tracking-tight text-muted-foreground">—</p>
              <p className="mt-0.5 text-[11px] text-muted-foreground">no credit limit set</p>
            </>
          )}
        </SnapshotTile>
        <SnapshotTile index={2} label="DSO / Days to Pay">
          <p className="text-2xl font-semibold tracking-tight tabular-nums">
            {profile.dsoDays.toFixed(0)}d
            <span className="ml-2 text-sm font-medium text-muted-foreground tabular-nums">
              / {snapshot.avgDaysToPay.toFixed(0)}d avg
            </span>
          </p>
          <p className="mt-0.5 text-[11px] text-muted-foreground">trailing 91d / 12mo</p>
        </SnapshotTile>
        <SnapshotTile index={3} label="Delinquency Score">
          <p className={cn("text-2xl font-semibold tracking-tight tabular-nums", scoreClass)}>
            {score.toFixed(0)}
          </p>
          <p className="mt-0.5 text-[11px] text-muted-foreground">0 low risk · 100 high risk</p>
        </SnapshotTile>
      </div>

      <div className="grid gap-3 xl:grid-cols-2">
        <Card className="gap-0 p-0">
          <CardHeader className="border-b px-4 py-2.5">
            <CardTitle className="text-xs font-medium">Payments — trailing 12 months</CardTitle>
          </CardHeader>
          <CardContent className="p-3">
            {chartData.length === 0 ? (
              <div className="flex h-28 items-center justify-center text-xs text-muted-foreground">
                No payments received yet
              </div>
            ) : (
              <ChartContainer config={collectionsChartConfig} className="h-28 w-full">
                <BarChart data={chartData} margin={{ left: 4, right: 4, top: 4 }}>
                  <XAxis
                    dataKey="label"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={6}
                    fontSize={10}
                  />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Bar
                    dataKey="collected"
                    fill="var(--color-collected)"
                    radius={[3, 3, 0, 0]}
                  />
                </BarChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>

        <Card className="gap-0 p-0">
          <CardHeader className="border-b px-4 py-2.5">
            <CardTitle className="text-xs font-medium">Account details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 p-4">
            <div className="grid grid-cols-2 gap-x-6 gap-y-2 text-xs">
              <DetailRow
                label="Oldest open invoice"
                value={
                  snapshot.oldestOpenInvoiceDate
                    ? `${formatDateOrDash(snapshot.oldestOpenInvoiceDate)} · ${snapshot.oldestDaysPastDue}d past due`
                    : "—"
                }
              />
              <DetailRow
                label="Last payment"
                value={
                  snapshot.lastPaymentDate
                    ? `${formatCurrency(snapshot.lastPaymentMinor / 100)} on ${formatDateOrDash(snapshot.lastPaymentDate)}`
                    : "—"
                }
              />
              <DetailRow
                label="Unapplied cash"
                value={formatCurrency(snapshot.unappliedCashMinor / 100)}
              />
              <DetailRow
                label="Billed trailing 91d"
                value={formatCurrency(snapshot.billedTrailing91Minor / 100)}
              />
            </div>
            {snapshot.buckets.totalOpenMinor > 0 ? (
              <AgingDistributionBar totals={snapshot.buckets} />
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function SnapshotTile({
  index,
  label,
  children,
}: {
  index: number;
  label: string;
  children: React.ReactNode;
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
          <div className="mt-1">{children}</div>
        </CardContent>
      </Card>
    </m.div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-muted-foreground">{label}</p>
      <p className="mt-0.5 font-medium tabular-nums">{value}</p>
    </div>
  );
}
