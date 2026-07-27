import type React from "react";
import { Delta, KpiCard, KpiHeader, KpiSub } from "./kpi-card";
import type { DeltaTone } from "./tone";

type KpiRingProps = {
  label: string;
  value: string;
  unit?: string;
  delta?: number;
  deltaLabel?: string;
  deltaTone?: DeltaTone;
  sub?: React.ReactNode;
  ringValue: number;
  ringMax?: number;
  target?: number;
  icon?: React.ReactNode;
  info?: React.ReactNode;
  span?: 2 | 3;
};

export function KpiRing({
  label,
  value,
  unit,
  delta,
  deltaLabel,
  deltaTone,
  sub,
  ringValue,
  ringMax = 100,
  target,
  icon,
  info,
  span = 2,
}: KpiRingProps) {
  const pct = Math.min(100, Math.max(0, (ringValue / ringMax) * 100));
  const onTarget = target !== undefined ? ringValue >= target : true;
  const ringColor = onTarget ? "var(--success)" : "var(--warning)";

  return (
    <KpiCard span={span}>
      <KpiHeader
        icon={icon}
        label={label}
        info={info}
        right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone} />}
      />
      <div className="mt-0.5 flex items-center gap-2.5">
        <Ring value={pct} color={ringColor} />
        <div className="flex min-w-0 flex-col gap-0.5">
          <div className="flex items-baseline gap-0.5">
            <span className="font-mono text-[22px] leading-none font-semibold tracking-tight tabular-nums">
              {value}
            </span>
            {unit && <span className="font-mono text-[11px] text-muted-foreground">{unit}</span>}
          </div>
          {target !== undefined && (
            <span className="font-mono text-[9.5px] tracking-wide text-muted-foreground/80 uppercase">
              Target {target}
              {unit ?? ""}
            </span>
          )}
        </div>
      </div>
      <KpiSub>{sub}</KpiSub>
    </KpiCard>
  );
}

type RingProps = {
  value: number;
  color: string;
  size?: number;
  stroke?: number;
};

export function Ring({ value, color, size = 42, stroke = 4 }: RingProps) {
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (Math.max(0, Math.min(100, value)) / 100) * circumference;

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-hidden>
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        stroke="color-mix(in oklch, currentColor 12%, transparent)"
        strokeWidth={stroke}
        fill="none"
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        stroke={color}
        strokeWidth={stroke}
        fill="none"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        strokeLinecap="round"
        transform={`rotate(-90 ${size / 2} ${size / 2})`}
      />
    </svg>
  );
}
