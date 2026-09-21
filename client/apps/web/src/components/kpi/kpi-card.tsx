import { cn } from "@trenova/shared/lib/utils";
import type React from "react";
import { KPI_STRIP_CELL_CLASS, KpiDelta, useInKpiStrip } from "./kpi-strip";
import type { DeltaTone } from "./tone";

type Density = "default" | "compact";

type KpiCardProps = {
  span: 2 | 3;
  density?: Density;
  className?: string;
  children: React.ReactNode;
};

export function KpiCard({ span, density = "default", className, children }: KpiCardProps) {
  const inStrip = useInKpiStrip();

  if (inStrip) {
    return (
      <div className={cn(KPI_STRIP_CELL_CLASS, "flex flex-col gap-1.5", className)}>{children}</div>
    );
  }

  return (
    <div
      className={cn(
        "border-border bg-card flex flex-col gap-2 rounded-lg border p-3",
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
      <div className="text-foreground-subtle flex min-w-0 items-center gap-1.5 text-xs font-medium">
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
  return (
    <span className="text-xs">
      <KpiDelta delta={delta} label={deltaLabel} tone={deltaTone} />
    </span>
  );
}

type KpiSubProps = {
  children: React.ReactNode;
};

export function KpiSub({ children }: KpiSubProps) {
  return <div className="text-foreground-muted mt-auto text-xs leading-snug">{children}</div>;
}
