import type { LucideIcon } from "lucide-react";
import type React from "react";
import { KpiCard, KpiHeader, KpiSub } from "./kpi/kpi-card";

type KPICardProps = {
  label: string;
  value: string;
  icon: LucideIcon;
  detail?: React.ReactNode;
  children?: React.ReactNode;
};

export function KPICard({ label, value, icon: Icon, detail, children }: KPICardProps) {
  return (
    <KpiCard span={2}>
      <KpiHeader icon={<Icon className="size-[11px]" />} label={label} />
      <span className="font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums">
        {value}
      </span>
      {children ?? <KpiSub>{detail}</KpiSub>}
    </KpiCard>
  );
}
