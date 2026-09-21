import { KpiStripItem } from "@/components/kpi/kpi-strip";
import type { ReactNode } from "react";

type DeskMetricProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  size?: "md" | "lg";
  valueClassName?: string;
  className?: string;
};

export function DeskMetric({
  label,
  value,
  sub,
  size = "md",
  valueClassName,
  className,
}: DeskMetricProps) {
  return (
    <KpiStripItem
      label={label}
      value={valueClassName ? <span className={valueClassName}>{value}</span> : value}
      sub={sub}
      size={size}
      className={className}
    />
  );
}
