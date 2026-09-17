import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { sortBasicMeasures } from "@/lib/carrier-intelligence";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { formatNumber } from "@trenova/shared/i18n/format";
import { csaBasicHint, csaBasicLabel } from "@trenova/shared/lib/csa";
import { cn, formatPercent } from "@trenova/shared/lib/utils";
import { ActivityIcon } from "lucide-react";
import { useMemo } from "react";
import { Bar, CartesianGrid, Cell, ComposedChart, Scatter, XAxis, YAxis } from "recharts";
import { IntelSectionCard } from "./intel-section-card";

export type BasicsChartCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

type BasicMeasure = NonNullable<CarrierIntelProfile["basics"]>[number];

type BasicChartRow = {
  basic: string;
  label: string;
  percentile: number;
  threshold: number | null;
  alert: boolean;
};

const chartConfig = {
  percentile: {
    label: "Percentile",
    color: "var(--brand)",
  },
  threshold: {
    label: "Intervention threshold",
    color: "var(--foreground)",
  },
  alert: {
    label: "Over threshold",
    color: "var(--destructive)",
  },
} satisfies ChartConfig;

const ROW_HEIGHT = 34;
const AXIS_ALLOWANCE = 32;

type ThresholdTickProps = {
  cx?: number;
  cy?: number;
  payload?: BasicChartRow;
};

function ThresholdTick({ cx, cy, payload }: ThresholdTickProps) {
  if (
    cx === undefined ||
    cy === undefined ||
    payload?.threshold === null ||
    payload?.threshold === undefined
  ) {
    return null;
  }
  return (
    <line
      x1={cx}
      x2={cx}
      y1={cy - 11}
      y2={cy + 11}
      stroke="var(--color-threshold)"
      strokeWidth={2}
      strokeDasharray="3 2"
    />
  );
}

function BasicFlags({ measure }: { measure: BasicMeasure }) {
  const t = useT();

  return (
    <span className="flex flex-wrap items-center gap-1">
      {measure.alert ? (
        <Badge variant="inactive" className="max-h-5">
          {t("Alert")}
        </Badge>
      ) : null}
      {measure.roadsideAlert ? (
        <Badge variant="orange" className="max-h-5">
          {t("Roadside alert")}
        </Badge>
      ) : null}
      {measure.acIndicator ? (
        <Badge variant="warning" className="max-h-5" title={t("Acute or critical violation found")}>
          {t("Acute/critical")}
        </Badge>
      ) : null}
    </span>
  );
}

export function BasicsChartCard({ profile, provider, className }: BasicsChartCardProps) {
  const t = useT();
  const measures = useMemo(() => sortBasicMeasures(profile.basics ?? []), [profile.basics]);
  const rows = useMemo<BasicChartRow[]>(
    () =>
      measures
        .filter((measure) => measure.percentile !== null)
        .map((measure) => ({
          basic: measure.basic,
          label: csaBasicLabel(measure.basic),
          percentile: measure.percentile ?? 0,
          threshold: measure.threshold,
          alert: measure.alert,
        })),
    [measures],
  );
  const anyAlert = measures.some((measure) => measure.alert);

  return (
    <IntelSectionCard
      title={t("CSA BASICs")}
      icon={ActivityIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={anyAlert ? "warning" : "none"}
      className={className}
      parts={[
        {
          section: "Basics",
          hasData: measures.length > 0,
          content: (
            <div className="flex flex-col gap-3">
              {rows.length > 0 ? (
                <figure className="flex flex-col gap-1">
                  <ChartContainer
                    config={chartConfig}
                    className="aspect-auto w-full"
                    style={{ height: rows.length * ROW_HEIGHT + AXIS_ALLOWANCE }}
                    role="img"
                    aria-label={t("Percentile against intervention threshold for each BASIC")}
                  >
                    <ComposedChart
                      data={rows}
                      layout="vertical"
                      margin={{ top: 4, right: 12, bottom: 4, left: 4 }}
                    >
                      <CartesianGrid horizontal={false} />
                      <XAxis
                        type="number"
                        domain={[0, 100]}
                        ticks={[0, 25, 50, 75, 100]}
                        tickLine={false}
                        axisLine={false}
                        tickFormatter={(value: number) => `${value}%`}
                      />
                      <YAxis
                        type="category"
                        dataKey="label"
                        width={132}
                        tickLine={false}
                        axisLine={false}
                      />
                      <ChartTooltip
                        cursor={false}
                        content={
                          <ChartTooltipContent
                            formatter={(value, name) => (
                              <span className="flex w-full justify-between gap-3">
                                <span className="text-muted-foreground">
                                  {name === "threshold"
                                    ? t("Intervention threshold")
                                    : t("Percentile")}
                                </span>
                                <span className="font-mono tabular-nums">
                                  {typeof value === "number" ? formatPercent(value, 0) : "-"}
                                </span>
                              </span>
                            )}
                          />
                        }
                      />
                      <Bar dataKey="percentile" barSize={14} radius={[0, 4, 4, 0]}>
                        {rows.map((row) => (
                          <Cell
                            key={row.basic}
                            fill={row.alert ? "var(--color-alert)" : "var(--color-percentile)"}
                          />
                        ))}
                      </Bar>
                      <Scatter dataKey="threshold" shape={<ThresholdTick />} />
                    </ComposedChart>
                  </ChartContainer>
                  <figcaption className="text-muted-foreground text-xs">
                    {t(
                      "Bars are the carrier's percentile among its peers; the dashed tick is the FMCSA intervention threshold. Higher is worse.",
                    )}
                  </figcaption>
                </figure>
              ) : null}
              <ul className="divide-y rounded-md border">
                {measures.map((measure) => (
                  <li
                    key={measure.basic}
                    className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 px-2.5 py-1.5"
                  >
                    <span className="flex min-w-0 flex-col">
                      <span
                        className={cn("truncate text-sm", measure.alert && "font-medium")}
                        title={csaBasicHint(measure.basic)}
                      >
                        {csaBasicLabel(measure.basic)}
                      </span>
                      <BasicFlags measure={measure} />
                    </span>
                    <span className="text-right text-xs tabular-nums">
                      {measure.percentile !== null ? (
                        <span className="font-mono font-medium">
                          {formatPercent(measure.percentile, 0)}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">{t("No percentile")}</span>
                      )}
                      <span className="text-muted-foreground block">
                        {measure.threshold !== null
                          ? t("threshold {0}", formatPercent(measure.threshold, 0))
                          : null}
                        {measure.measure !== null
                          ? ` · ${t("measure {0}", formatNumber(measure.measure, { maximumFractionDigits: 2 }))}`
                          : null}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ),
        },
      ]}
    />
  );
}
