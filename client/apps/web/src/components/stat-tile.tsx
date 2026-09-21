import type { ReactNode } from "react";
import { KpiStripItem } from "./kpi/kpi-strip";
import type { Tone } from "./kpi/tone";

type StatTileTone = "warn" | "info" | "danger";

const STAT_TILE_TONE: Record<StatTileTone, Tone> = {
  warn: "warning",
  info: "info",
  danger: "danger",
};

type StatTileProps = {
  label: string;
  value: ReactNode;
  sub: ReactNode;
  hint: string;
  tone?: StatTileTone;
  clickable?: boolean;
  onClick?: () => void;
  active?: boolean;
};

export function StatTile({
  label,
  value,
  sub,
  hint,
  tone,
  clickable,
  onClick,
  active,
}: StatTileProps) {
  return (
    <KpiStripItem
      label={label}
      value={value}
      sub={sub}
      hint={hint}
      tone={tone ? STAT_TILE_TONE[tone] : undefined}
      active={active}
      onClick={clickable ? onClick : undefined}
    />
  );
}
