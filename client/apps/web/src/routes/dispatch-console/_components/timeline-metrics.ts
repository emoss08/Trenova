import type { TimelineZoom } from "@/components/timeline/timeline-scale";

/**
 * The console's window lengths and how closely the timeline draws each. They live outside the
 * timeline so the URL state and the toolbar can read them without pulling the timeline into
 * the initial chunk.
 */

export type TimelineZoomDays = 1 | 2 | 3;

export const TIMELINE_ZOOM_OPTIONS: readonly {
  id: TimelineZoomDays;
  label: string;
  zoom: TimelineZoom;
}[] = [
  { id: 1, label: "Day", zoom: { pxPerHour: 64, tickHours: 2, hourLabels: true } },
  { id: 2, label: "2 Days", zoom: { pxPerHour: 32, tickHours: 4, hourLabels: true } },
  { id: 3, label: "3 Days", zoom: { pxPerHour: 24, tickHours: 6, hourLabels: true } },
] as const;

export function timelineZoom(days: TimelineZoomDays): TimelineZoom {
  return (TIMELINE_ZOOM_OPTIONS.find((option) => option.id === days) ?? TIMELINE_ZOOM_OPTIONS[0])
    .zoom;
}
