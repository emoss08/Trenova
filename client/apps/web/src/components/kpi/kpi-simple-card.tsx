import type { LucideIcon } from "lucide-react";
import type React from "react";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
import { KPI_VALUE_CLASS, KPI_VALUE_LG_CLASS, useInKpiStrip } from "@/components/kpi/kpi-strip";

type KPICardProps = {
  label: string;
  value: string;
  icon: LucideIcon;
  detail?: React.ReactNode;
  children?: React.ReactNode;
};

export function KPICard({ label, value, icon: Icon, detail, children }: KPICardProps) {
  const inStrip = useInKpiStrip();

  return (
    <KpiCard span={2}>
      <KpiHeader icon={<Icon className="size-3" />} label={label} />
      <span className={inStrip ? KPI_VALUE_CLASS : KPI_VALUE_LG_CLASS}>{value}</span>
      {children ?? <KpiSub>{detail}</KpiSub>}
    </KpiCard>
  );
}
