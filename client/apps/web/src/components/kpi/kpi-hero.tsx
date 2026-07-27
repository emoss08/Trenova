import type React from "react";
import { Delta, KpiCard, KpiHeader, KpiSub } from "./kpi-card";
import { type Segment, SegmentedBar } from "./segmented-bar";
import { Sparkline } from "./sparkline";
import type { DeltaTone } from "./tone";

type KpiHeroProps = {
  label: string;
  value: string;
  unit?: string;
  delta?: number;
  deltaLabel?: string;
  deltaTone?: DeltaTone;
  sub?: React.ReactNode;
  sparkData?: number[];
  sparkColor?: string;
  breakdown?: Segment[];
  icon?: React.ReactNode;
  info?: React.ReactNode;
  span?: 2 | 3;
};

export function KpiHero({
  label,
  value,
  unit,
  delta,
  deltaLabel,
  deltaTone,
  sub,
  sparkData,
  sparkColor,
  breakdown,
  icon,
  info,
  span = 3,
}: KpiHeroProps) {
  const showSpark = sparkData && !breakdown;

  return (
    <KpiCard span={span} className="gap-2.5">
      <KpiHeader
        icon={icon}
        label={label}
        info={info}
        right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone} />}
      />
      <div className="flex items-baseline gap-1">
        <span className="font-mono text-[28px] leading-none font-semibold tracking-tight tabular-nums">
          {value}
        </span>
        {unit && <span className="font-mono text-xs text-muted-foreground">{unit}</span>}
      </div>
      {breakdown && <SegmentedBar segments={breakdown} />}
      <div className="mt-auto flex items-end justify-between gap-2">
        <KpiSub>{sub}</KpiSub>
        {showSpark && (
          <Sparkline data={sparkData} color={sparkColor ?? "var(--brand)"} width={88} height={24} />
        )}
      </div>
    </KpiCard>
  );
}
