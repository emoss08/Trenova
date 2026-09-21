import type React from "react";
import { Delta, KpiCard, KpiHeader, KpiSub } from "./kpi-card";
import { KPI_VALUE_CLASS, KPI_VALUE_LG_CLASS, useInKpiStrip } from "./kpi-strip";
import { type DeltaTone, toneVar } from "./tone";

type KpiStatProps = {
  label: string;
  value: string;
  delta?: number;
  deltaLabel?: string;
  deltaTone?: DeltaTone;
  sub?: React.ReactNode;
  tone?: DeltaTone;
  icon?: React.ReactNode;
  info?: React.ReactNode;
  span?: 2 | 3;
  className?: string;
};

export function KpiStat({
  label,
  value,
  delta,
  deltaLabel,
  deltaTone,
  sub,
  tone,
  icon,
  info,
  span = 2,
  className,
}: KpiStatProps) {
  const inStrip = useInKpiStrip();

  return (
    <KpiCard span={span} density="compact" className={className}>
      <KpiHeader
        icon={
          <span className="inline-flex items-center gap-1.5">
            {tone ? (
              <span
                aria-hidden
                className="size-1.5 shrink-0 rounded-full"
                style={{ background: toneVar(tone) }}
              />
            ) : null}
            {icon}
          </span>
        }
        label={label}
        info={info}
        right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone} />}
      />
      <span className={inStrip ? KPI_VALUE_CLASS : KPI_VALUE_LG_CLASS}>{value}</span>
      <KpiSub>{sub}</KpiSub>
    </KpiCard>
  );
}
