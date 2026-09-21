import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { Tone } from "@/components/kpi/tone";
import type { ReactNode } from "react";
import type { StatusTone } from "./status-dot";

export type StatStripItem = {
  id: string;
  label: ReactNode;
  value: ReactNode;
  hint?: ReactNode;
  tone?: StatusTone;
  onClick?: () => void;
};

export type StatStripProps = {
  items: StatStripItem[];
  className?: string;
};

const STATUS_TONE: Record<StatusTone, Tone> = {
  critical: "danger",
  high: "warning",
  medium: "warning",
  low: "info",
  info: "muted",
  success: "success",
  neutral: "muted",
};

export function StatStrip({ items, className }: StatStripProps) {
  return (
    <KpiStrip className={className}>
      {items.map((item) => (
        <KpiStripItem
          key={item.id}
          label={item.label}
          value={item.value}
          sub={item.hint}
          tone={item.tone ? STATUS_TONE[item.tone] : undefined}
          onClick={item.onClick}
        />
      ))}
    </KpiStrip>
  );
}
