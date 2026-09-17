import { useNowSeconds } from "@/hooks/use-now-seconds";
import { dailyUsageSeries, spendCapProgress, type SpendCapState } from "@/lib/carrier-intel-usage";
import {
  carrierIntelProviderLabel,
  formatOptionalDecimalCurrency,
  INTEL_EMPTY_VALUE,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelUsageSummary } from "@/lib/graphql/carrier-intel-settings";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { Progress } from "@trenova/shared/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { Bar, BarChart, XAxis } from "recharts";
import { StatStrip } from "./stat-strip";

const progressVariantByState: Record<SpendCapState, "default" | "warning" | "error"> = {
  uncapped: "default",
  within: "default",
  soft: "warning",
  exceeded: "error",
};

export type CarrierIntelUsageSummaryViewProps = {
  usage: CarrierIntelUsageSummary;
  emptyMessage?: string;
};

export function CarrierIntelUsageSummaryView({
  usage,
  emptyMessage,
}: CarrierIntelUsageSummaryViewProps) {
  const t = useT();
  const now = useNowSeconds();
  const progress = spendCapProgress(usage.monthToDate, usage.cap, usage.softCapPercent);
  const chartConfig = useMemo<ChartConfig>(
    () => ({ cost: { label: t("Cost"), color: "var(--muted-foreground)" } }),
    [t],
  );

  const totals = useMemo(
    () =>
      usage.byEndpoint.reduce(
        (sum, row) => ({
          calls: sum.calls + row.calls,
          billableUnits: sum.billableUnits + row.billableUnits,
        }),
        { calls: 0, billableUnits: 0 },
      ),
    [usage.byEndpoint],
  );

  const series = useMemo(
    () =>
      dailyUsageSeries(usage.daily, usage.monthStart, now).map((point) => ({
        ...point,
        label: formatUnixInUserTimezone(point.start, {
          month: "short",
          day: "numeric",
          timezone: "UTC",
        }),
      })),
    [now, usage.daily, usage.monthStart],
  );
  const hasUsage = usage.byEndpoint.length > 0;

  return (
    <div className="flex flex-col gap-4">
      <StatStrip
        items={[
          {
            id: "spend",
            label: t("Spend"),
            tone:
              progress.state === "exceeded"
                ? "critical"
                : progress.state === "soft"
                  ? "medium"
                  : undefined,
            value: (
              <span data-testid="usage-month-to-date">
                {formatOptionalDecimalCurrency(usage.monthToDate) ?? INTEL_EMPTY_VALUE}
              </span>
            ),
            hint:
              progress.state === "uncapped" ? (
                t("No monthly cap")
              ) : (
                <span className="flex flex-col gap-1.5 pt-1">
                  <Progress
                    value={progress.percent}
                    size="sm"
                    variant={progressVariantByState[progress.state]}
                    aria-label={t("Spend against the monthly cap")}
                  />
                  <span className="truncate">
                    {t(
                      "of {0} · warns at {1}",
                      formatOptionalDecimalCurrency(usage.cap) ?? INTEL_EMPTY_VALUE,
                      formatOptionalDecimalCurrency(progress.softCapAmount) ?? INTEL_EMPTY_VALUE,
                    )}
                  </span>
                </span>
              ),
          },
          {
            id: "billable",
            label: t("Billable units"),
            value: totals.billableUnits.toLocaleString(),
          },
          {
            id: "calls",
            label: t("Provider calls"),
            value: totals.calls.toLocaleString(),
            hint: t("Since {0}", formatUnixDateMedium(usage.monthStart, { timezone: "UTC" })),
          },
        ]}
      />

      {!hasUsage ? (
        <p className="text-muted-foreground rounded-lg border px-4 py-6 text-center text-xs">
          {emptyMessage ?? t("No provider calls this month.")}
        </p>
      ) : (
        <>
          <section className="flex flex-col gap-2" aria-label={t("Daily cost")}>
            <h4 className="text-muted-foreground text-xs font-medium">{t("Daily cost")}</h4>
            <ChartContainer config={chartConfig} className="aspect-auto h-40 w-full">
              <BarChart data={series} margin={{ left: 0, right: 0, top: 4, bottom: 0 }}>
                <XAxis
                  dataKey="label"
                  tickLine={false}
                  axisLine={false}
                  tickMargin={6}
                  minTickGap={24}
                />
                <ChartTooltip
                  cursor={false}
                  content={
                    <ChartTooltipContent
                      hideIndicator
                      formatter={(value) => formatCurrency(Number(value))}
                    />
                  }
                />
                <Bar
                  dataKey="cost"
                  fill="var(--color-cost)"
                  fillOpacity={0.55}
                  radius={[2, 2, 0, 0]}
                  maxBarSize={18}
                />
              </BarChart>
            </ChartContainer>
          </section>

          <section className="flex flex-col gap-2" aria-label={t("By endpoint")}>
            <h4 className="text-muted-foreground text-xs font-medium">{t("By endpoint")}</h4>
            <Table containerClassName="rounded-lg border">
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs">{t("Endpoint")}</TableHead>
                  <TableHead className="text-right text-xs">{t("Calls")}</TableHead>
                  <TableHead className="text-right text-xs">{t("Billable")}</TableHead>
                  <TableHead className="text-right text-xs">{t("Cost")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usage.byEndpoint.map((row) => (
                  <TableRow key={`${row.provider}-${row.endpoint}`}>
                    <TableCell>
                      <div className="flex flex-col">
                        <span className="text-sm">{row.endpoint}</span>
                        <span className="text-muted-foreground text-xs">
                          {carrierIntelProviderLabel(row.provider)}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {row.calls.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {row.billableUnits.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatOptionalDecimalCurrency(row.estimatedCost) ?? INTEL_EMPTY_VALUE}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </section>
        </>
      )}
    </div>
  );
}
