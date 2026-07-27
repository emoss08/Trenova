import type React from "react";
import { Delta, KpiCard, KpiHeader, KpiSub } from "./kpi-card";
import type { DeltaTone } from "./tone";

type KpiGoalBarProps = {
  label: string;
  value: string;
  unit?: string;
  delta?: number;
  deltaLabel?: string;
  deltaTone?: DeltaTone;
  sub?: React.ReactNode;
  actual: number;
  target: number;
  max: number;
  icon?: React.ReactNode;
  info?: React.ReactNode;
  span?: 2 | 3;
};

export function KpiGoalBar({
  label,
  value,
  unit,
  delta,
  deltaLabel,
  deltaTone,
  sub,
  actual,
  target,
  max,
  icon,
  info,
  span = 2,
}: KpiGoalBarProps) {
  const actualPct = Math.min(100, Math.max(0, (actual / max) * 100));
  const targetPct = Math.min(100, Math.max(0, (target / max) * 100));
  const onGoal = actual <= target;
  const fillColor = onGoal ? "var(--success)" : "var(--warning)";

  return (
    <KpiCard span={span}>
      <KpiHeader
        icon={icon}
        label={label}
        info={info}
        right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone} />}
      />
      <div className="flex items-baseline gap-1">
        <span className="font-mono text-[22px] leading-none font-semibold tracking-tight tabular-nums">
          {value}
        </span>
        {unit && <span className="font-mono text-[11px] text-muted-foreground">{unit}</span>}
      </div>
      <div className="relative mt-0.5 h-1.5 rounded-sm bg-muted">
        <div
          className="absolute inset-y-0 left-0 rounded-sm"
          style={{
            width: `${actualPct}%`,
            background: fillColor,
          }}
        />
        <div
          title={`Target ${target}${unit ?? ""}`}
          className="absolute -top-0.5 -bottom-0.5 w-0.5 rounded-[1px] bg-foreground/55"
          style={{ left: `calc(${targetPct}% - 1px)` }}
        />
      </div>
      <KpiSub>{sub}</KpiSub>
    </KpiCard>
  );
}
