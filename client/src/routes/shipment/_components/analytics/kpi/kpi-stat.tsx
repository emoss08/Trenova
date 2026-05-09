import type React from "react";
import { Delta, KpiCard, KpiHeader, KpiSub } from "./kpi-card";
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
};

export function KpiStat({
  label,
  value,
  delta,
  deltaLabel,
  deltaTone,
  sub,
  tone = "brand",
  icon,
  info,
  span = 2,
}: KpiStatProps) {
  const dot = toneVar(tone);

  return (
    <KpiCard span={span} density="compact">
      <KpiHeader
        icon={
          <span className="inline-flex items-center gap-1.5">
            <span
              aria-hidden
              className="size-1.5 shrink-0 rounded-full"
              style={{ background: dot }}
            />
            {icon}
          </span>
        }
        label={label}
        info={info}
        right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone} />}
      />
      <span className="font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums">
        {value}
      </span>
      <KpiSub>{sub}</KpiSub>
    </KpiCard>
  );
}
