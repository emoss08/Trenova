import { useT } from "@trenova/shared/i18n/use-t";
import { AgingDistributionBar } from "@/components/accounting/aging-buckets";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import type { Tone } from "@/components/kpi/tone";
import { SectionPanel } from "@/components/section-panel";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { ARCustomerProfile } from "@/lib/graphql/accounts-receivable";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { Bar, BarChart, XAxis } from "recharts";
import { formatUnixDateMedium, formatUnixInUserTimezone } from "@trenova/shared/lib/date";

const collectionsChartConfig = {
  collected: {
    label: "Collected",
    color: "var(--success)",
  },
} satisfies ChartConfig;

// Buckets are keyed to UTC month starts, so the label reads them back in UTC —
// projecting into the viewer's zone would slide a bucket into the month before.
function monthLabel(unixSeconds: number) {
  return formatUnixInUserTimezone(unixSeconds, { month: "short", timezone: "UTC" });
}

function formatDateOrDash(unixSeconds: number) {
  return formatUnixDateMedium(unixSeconds, { fallback: "—" });
}

export function CustomerSnapshotHeader({
  profile,
  isLoading,
}: {
  profile: ARCustomerProfile | undefined;
  isLoading: boolean;
}) {
  const t = useT();

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
      <div className="flex flex-col gap-3">
        <KpiStripSkeleton count={4} size="lg" />
        <div className="grid gap-3 xl:grid-cols-2">
          <Skeleton className="h-40 rounded-lg" />
          <Skeleton className="h-40 rounded-lg" />
        </div>
      </div>
    );
  }

  if (!profile) return null;

  const snapshot = profile.snapshot;
  const utilization = profile.creditUtilization;
  const utilizationPct = Math.min(utilization * 100, 100);
  const utilizationBarClass =
    utilization >= 1 ? "bg-danger" : utilization >= 0.75 ? "bg-warning" : "bg-success";
  const hasCreditLimit = snapshot.hasCreditLimit && snapshot.creditLimitMinor > 0;

  const score = profile.delinquencyScore;
  const scoreTone: Tone = score >= 60 ? "danger" : score >= 30 ? "warning" : "success";

  return (
    <div className="flex flex-col gap-3">
      <KpiStrip>
        <KpiStripItem
          size="lg"
          label={t("Open balance")}
          value={formatCurrency(snapshot.totalOpenMinor / 100)}
          sub={
            <span className="tabular-nums">
              {t(
                "{0} overdue · {1} open",
                formatCurrency(snapshot.overdueMinor / 100),
                snapshot.openInvoiceCount,
              )}
            </span>
          }
        />
        <KpiStripItem
          size="lg"
          label={t("Credit utilization")}
          value={
            hasCreditLimit ? (
              <span className="flex flex-col gap-1.5">
                <span>{(utilization * 100).toFixed(0)}%</span>
                <span aria-hidden className="bg-muted block h-1 w-full overflow-hidden rounded-full">
                  <span
                    className={cn("block h-full rounded-full", utilizationBarClass)}
                    style={{ width: `${utilizationPct}%` }}
                  />
                </span>
              </span>
            ) : (
              <span className="text-muted-foreground">—</span>
            )
          }
          sub={
            hasCreditLimit ? (
              <span className="tabular-nums">
                {t("of {0} limit", formatCurrency(snapshot.creditLimitMinor / 100))}
              </span>
            ) : (
              t("no credit limit set")
            )
          }
        />
        <KpiStripItem
          size="lg"
          label={t("DSO / days to pay")}
          value={
            <>
              {t("{0}d", profile.dsoDays.toFixed(0))}
              <span className="text-muted-foreground ml-2 text-sm font-normal tabular-nums">
                / {snapshot.avgDaysToPay.toFixed(0)}
                {t("d avg")}
              </span>
            </>
          }
          sub="trailing 91d / 12mo"
        />
        <KpiStripItem
          size="lg"
          label={t("Delinquency score")}
          tone={scoreTone}
          value={score.toFixed(0)}
          sub={t("0 low risk · 100 high risk")}
        />
      </KpiStrip>

      <div className="grid gap-3 xl:grid-cols-2">
        <SectionPanel title={t("Payments — trailing 12 months")}>
          <div className="p-3">
            {chartData.length === 0 ? (
              <div className="text-muted-foreground flex h-28 items-center justify-center text-xs">
                {t("No payments received yet")}
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
                  <Bar dataKey="collected" fill="var(--color-collected)" radius={[3, 3, 0, 0]} />
                </BarChart>
              </ChartContainer>
            )}
          </div>
        </SectionPanel>

        <SectionPanel title={t("Account details")}>
          <div className="flex flex-col gap-3 p-3">
            <DescriptionList columns={2}>
              <DescriptionItem label={t("Oldest open invoice")} numeric>
                {snapshot.oldestOpenInvoiceDate ? (
                  `${formatDateOrDash(snapshot.oldestOpenInvoiceDate)} · ${snapshot.oldestDaysPastDue}d past due`
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Last payment")} numeric>
                {snapshot.lastPaymentDate ? (
                  `${formatCurrency(snapshot.lastPaymentMinor / 100)} on ${formatDateOrDash(snapshot.lastPaymentDate)}`
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Unapplied cash")} numeric>
                {formatCurrency(snapshot.unappliedCashMinor / 100)}
              </DescriptionItem>
              <DescriptionItem label={t("Billed trailing 91d")} numeric>
                {formatCurrency(snapshot.billedTrailing91Minor / 100)}
              </DescriptionItem>
            </DescriptionList>
            {snapshot.buckets.totalOpenMinor > 0 ? (
              <AgingDistributionBar totals={snapshot.buckets} />
            ) : null}
          </div>
        </SectionPanel>
      </div>
    </div>
  );
}
