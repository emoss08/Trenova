/**
 * Zoom steps and pixel geometry for the dispatch timeline. They live outside the timeline
 * component so the URL state and the loading skeleton can read them without pulling the
 * timeline — and the virtualizer behind it — into the initial chunk.
 */

export type TimelineZoomDays = 1 | 2 | 3;

export const TIMELINE_ZOOM_OPTIONS: readonly {
  id: TimelineZoomDays;
  label: string;
  pxPerHour: number;
  hourTickStep: number;
}[] = [
  { id: 1, label: "Day", pxPerHour: 64, hourTickStep: 2 },
  { id: 2, label: "2 Days", pxPerHour: 32, hourTickStep: 4 },
  { id: 3, label: "3 Days", pxPerHour: 26, hourTickStep: 6 },
] as const;

export function timelineZoomConfig(zoom: TimelineZoomDays) {
  return TIMELINE_ZOOM_OPTIONS.find((option) => option.id === zoom) ?? TIMELINE_ZOOM_OPTIONS[0];
}

export const RAIL_WIDTH_PX = 216;
export const DAY_LABEL_HEIGHT_PX = 26;
export const HOUR_TICK_HEIGHT_PX = 20;
export const LANE_HEIGHT_PX = 30;
export const BAR_HEIGHT_PX = 22;
export const ROW_PADDING_PX = 8;

/** A single-lane row, which is what every row measures until work stacks on it. */
export const TIMELINE_ROW_HEIGHT_PX = LANE_HEIGHT_PX + ROW_PADDING_PX;
export const TIMELINE_HEADER_HEIGHT_PX = DAY_LABEL_HEIGHT_PX + HOUR_TICK_HEIGHT_PX;
