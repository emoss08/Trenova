import {
  createTimelineScale,
  LABEL_WIDTH,
  type TimelineScale,
} from "@/components/timeline/timeline-scale";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { dayBoundaries } from "@/lib/timeline/time-scale";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { getTodayDate } from "@trenova/shared/lib/date";
import { Suspense, lazy, type ReactNode } from "react";
import { DispatchTimelineSkeleton } from "./console-skeletons";
import { timelineZoom } from "./timeline-metrics";
import { useDispatchWindow, type CenterMode } from "./url-state";

// The timeline carries the lanes, bars and drag targets, and the queue is what the console
// opens on; only someone who switches to it downloads it.
const loadTimeline = () => import("./dispatch-timeline");

export const DispatchTimeline = lazy(() =>
  loadTimeline().then((m) => ({ default: m.DispatchTimeline })),
);

/** Warms the timeline's chunk on hover so the switch itself never waits on the network. */
export function preloadCenterView(mode: CenterMode): void {
  if (mode === "timeline") void loadTimeline();
}

/** The console's window laid out on the shared timeline's axis, and today's start. */
export type TimelineSheet = { scale: TimelineScale; today: number };

/**
 * The timeline, framed as the queue is, with its own skeleton while either loads. The pane
 * lays the window out once, so the skeleton draws the same axis the loaded sheet will.
 */
export function TimelinePane({
  isLoading,
  children,
}: {
  isLoading: boolean;
  children: (sheet: TimelineSheet) => ReactNode;
}) {
  const [sheetRef, width] = useElementWidth<HTMLElement>();
  const timezone = useUserTimezone();
  const { range, zoom } = useDispatchWindow();
  const sheet: TimelineSheet = {
    scale: createTimelineScale(
      dayBoundaries(range.start, zoom, timezone),
      timelineZoom(zoom),
      Math.max(0, width - LABEL_WIDTH),
    ),
    today: getTodayDate(timezone),
  };
  const skeleton = <DispatchTimelineSkeleton {...sheet} />;

  return (
    <section
      ref={sheetRef}
      className="bg-card relative flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
    >
      {isLoading ? skeleton : <Suspense fallback={skeleton}>{children(sheet)}</Suspense>}
    </section>
  );
}
