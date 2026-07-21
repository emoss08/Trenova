export const RAIL_WIDTH_PX = 216;
export const LANE_HEIGHT_PX = 34;
export const BAR_HEIGHT_PX = 26;
export const ROW_PADDING_PX = 10;
export const DAY_LABEL_HEIGHT_PX = 26;
export const HOUR_TICK_HEIGHT_PX = 20;

export function rowHeightPx(laneCount: number): number {
  return laneCount * LANE_HEIGHT_PX + ROW_PADDING_PX;
}
