export const RAIL_WIDTH_PX = 216;
export const DAY_LABEL_HEIGHT_PX = 26;
export const HOUR_TICK_HEIGHT_PX = 20;
export const COLLAPSED_ROW_HEIGHT_PX = 25;
export const COLLAPSED_BAR_HEIGHT_PX = 9;

export type TimelineDensity = "comfortable" | "compact";

export type DensityConfig = {
  laneHeightPx: number;
  barHeightPx: number;
  rowPaddingPx: number;
};

export const DENSITY_CONFIGS: Record<TimelineDensity, DensityConfig> = {
  comfortable: { laneHeightPx: 34, barHeightPx: 26, rowPaddingPx: 10 },
  compact: { laneHeightPx: 25, barHeightPx: 19, rowPaddingPx: 6 },
};

export function rowHeightPx(
  laneCount: number,
  density: TimelineDensity,
  collapsed: boolean,
): number {
  if (collapsed) return COLLAPSED_ROW_HEIGHT_PX;
  const config = DENSITY_CONFIGS[density];
  return laneCount * config.laneHeightPx + config.rowPaddingPx;
}
