import { useT } from "@trenova/shared/i18n/use-t";
import { eventTotals, trendPeak, trendRows } from "@/lib/fleet-safety-console";
import type { FleetSafetyKindRow, FleetSafetyTrendPoint } from "@/lib/graphql/fleet-safety";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { safetyEventKindLabel, trendDirection } from "@trenova/shared/lib/csa";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { MinusIcon, TrendingDownIcon, TrendingUpIcon } from "lucide-react";
import { useMemo } from "react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

const chartConfig = {
  events: {
    label: "Events",
    color: "var(--brand)",
  },
} satisfies ChartConfig;

type TrendCardProps = {
  trend: readonly FleetSafetyTrendPoint[];
  kinds: readonly FleetSafetyKindRow[];
  windowMonths: number;
};

/**
 * Events by month, one series, with what each month was made of on hover. A
 * single bar per month rather than a stack of kinds: the question is whether
 * the fleet is getting better or worse, and a stack answers a different one.
 */
export function TrendCard({ trend, kinds, windowMonths }: TrendCardProps) {
  const t = useT();

  const rows = useMemo(() => trendRows(trend), [trend]);
  const direction = useMemo(() => trendDirection(rows.map((row) => row.events)), [rows]);
  const peak = useMemo(() => trendPeak(trend), [trend]);
  const totals = useMemo(() => eventTotals(kinds), [kinds]);

  return (
    <section aria-labelledby="trend-heading" className="bg-card overflow-hidden rounded-lg border">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <h3 id="trend-heading" className="text-sm font-medium">
            {t("Events by month")}
          </h3>
          <span className="text-muted-foreground text-xs">{t("last {0} months", windowMonths)}</span>
        </div>
        <span className="text-muted-foreground flex items-center gap-1 text-xs" aria-live="polite">
          {direction === "up" ? (
            <TrendingUpIcon className="size-3.5" aria-hidden />
          ) : direction === "down" ? (
            <TrendingDownIcon className="size-3.5" aria-hidden />
          ) : (
            <MinusIcon className="size-3.5" aria-hidden />
          )}
          {direction === "up" ? "Rising" : direction === "down" ? "Falling" : "Holding steady"}
          {peak && rows.length > 1
            ? ` · busiest ${formatUnixInUserTimezone(peak.periodStart, { month: "short", year: "numeric", timezone: "UTC" })}`
            : ""}
        </span>
      </header>

      <div className="p-3">
        {rows.length === 0 ? (
          <p className="text-muted-foreground rounded-lg border border-dashed p-4 text-center text-sm">
            {t("No safety events in this window.")}
          </p>
        ) : (
          <ChartContainer config={chartConfig} className="h-44 w-full">
            <BarChart data={rows} margin={{ left: 0, right: 8, top: 8 }} barCategoryGap="20%">
              <CartesianGrid vertical={false} strokeDasharray="3 3" />
              <XAxis
                dataKey="label"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={24}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                width={28}
                allowDecimals={false}
                domain={[0, "auto"]}
              />
              <ChartTooltip
                cursor={{ fill: "var(--accent)" }}
                content={
                  <ChartTooltipContent
                    labelFormatter={(_label, payload) => {
                      const row = payload?.[0]?.payload as (typeof rows)[number] | undefined;
                      return row
                        ? formatUnixInUserTimezone(row.periodStart, {
                            month: "long",
                            year: "numeric",
                            timezone: "UTC",
                          })
                        : "";
                    }}
                    formatter={(value, _name, item) => {
                      const row = item.payload as (typeof rows)[number];
                      return (
                        <div className="flex flex-col gap-0.5 text-xs">
                          <span className="font-medium tabular-nums">{t("{0} events", value)}</span>
                          <span className="text-muted-foreground tabular-nums">
                            {t("{0} accidents · {1} preventable · {2} out of service · {3} points", row.accidents, row.preventable, row.outOfService, row.points)}
                          </span>
                        </div>
                      );
                    }}
                  />
                }
              />
              <Bar
                dataKey="events"
                fill="var(--color-events)"
                radius={[4, 4, 0, 0]}
                maxBarSize={36}
              />
            </BarChart>
          </ChartContainer>
        )}
      </div>

      <ul className="divide-y border-t" aria-label={t("Events by kind")}>
        {kinds.map((kind) => (
          <li
            key={kind.kind}
            className="flex items-center justify-between gap-2 px-3 py-1.5 text-xs"
          >
            <span className="font-medium">{safetyEventKindLabel(kind.kind)}</span>
            <span className="text-muted-foreground tabular-nums">
              {kind.events}
              {kind.preventable > 0 ? ` · ${kind.preventable} preventable` : ""}
              {kind.outOfService > 0 ? ` · ${kind.outOfService} out of service` : ""}
              {kind.open > 0 ? ` · ${kind.open} open` : ""}
            </span>
          </li>
        ))}
        {kinds.length > 0 ? (
          <li className="text-muted-foreground flex items-center justify-between gap-2 px-3 py-1.5 text-xs">
            <span>{t("All kinds")}</span>
            <span className="tabular-nums">
              {t("{0} · {1} points", totals.events, totals.points)}
            </span>
          </li>
        ) : null}
      </ul>
    </section>
  );
}
