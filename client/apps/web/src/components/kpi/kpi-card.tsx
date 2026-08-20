import { cn } from "@trenova/shared/lib/utils";
import type React from "react";
import { type DeltaTone, toneVar } from "./tone";

type Density = "default" | "compact";

type KpiCardProps = {
  span: 2 | 3;
  density?: Density;
  className?: string;
  children: React.ReactNode;
};

export function KpiCard({ span, density = "default", className, children }: KpiCardProps) {
  return (
    <div
      className={cn(
        "border-border/80 bg-card hover:border-border flex flex-col gap-2 rounded-md border p-3 transition-colors",
        density === "compact" ? "h-[var(--kpi-h-sm)]" : "h-[var(--kpi-h)]",
        span === 3 ? "col-span-3" : "col-span-2",
        className,
      )}
    >
      {children}
    </div>
  );
}

type KpiHeaderProps = {
  icon?: React.ReactNode;
  label: string;
  info?: React.ReactNode;
  right?: React.ReactNode;
};

export function KpiHeader({ icon, label, info, right }: KpiHeaderProps) {
  return (
    <div className="flex min-h-[14px] items-center justify-between">
      <div className="cc-label flex items-center gap-1.5">
        {icon}
        <span>{label}</span>
        {info}
      </div>
      {right}
    </div>
  );
}

type DeltaProps = {
  delta: number | undefined | null;
  deltaLabel?: string;
  deltaTone?: DeltaTone;
};

export function Delta({ delta, deltaLabel, deltaTone }: DeltaProps) {
  if (delta === undefined || delta === null) return null;
  const positive = delta >= 0;
  const color = deltaTone ? toneVar(deltaTone) : positive ? toneVar("success") : toneVar("danger");

  return (
    <span
      className="inline-flex items-center gap-0.5 rounded-sm px-1.5 py-px font-mono text-[10.5px] tabular-nums"
      style={{
        color,
        background: `color-mix(in oklch, ${color} 12%, transparent)`,
      }}
    >
      {positive ? "▲" : "▼"}
      {Math.abs(delta)}
      {deltaLabel ?? ""}
    </span>
  );
}

type KpiSubProps = {
  children: React.ReactNode;
};

export function KpiSub({ children }: KpiSubProps) {
  return (
    <div className="text-muted-foreground/80 mt-auto text-[10.5px] leading-snug">{children}</div>
  );
}
