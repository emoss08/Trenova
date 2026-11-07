import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  TimelineCanvas,
  TimelineGroupRow,
  TimelineRow,
  type TimelineSheetItem,
} from "@/components/timeline/timeline-canvas";
import {
  GROUP_HEIGHT,
  LANE_PAD,
  ROW_HEIGHT,
  type TimelineScale,
} from "@/components/timeline/timeline-scale";
import {
  TimelineBarSkeleton,
  TimelineLaneLabelSkeleton,
} from "@/components/timeline/timeline-skeleton-parts";

/**
 * Every pane of the console is code-split, so these stand in twice: as the Suspense
 * fallback while a pane's chunk arrives, and as the pane's own loading state while the
 * board query runs. Sharing them is what keeps the two indistinguishable — the layout
 * never shifts between "loading the code" and "loading the data".
 */

/** The timeline's skeleton: the pool lane, then this many driver lanes. */
const TIMELINE_DRIVER_ROWS = 10;

/** The queue's columns, shared by its rows, its heading and this skeleton. */
export const QUEUE_GRID_CLASS = "grid grid-cols-[10.5rem_minmax(0,1fr)_minmax(0,18.75rem)] gap-3.5";

/** The queue as it will be drawn: two group headings over two-line rows. */
export function MoveQueueRowsSkeleton() {
  return (
    <div className="flex min-h-0 flex-col overflow-hidden" aria-hidden>
      {QUEUE_SKELETON_GROUPS.map((rows, group) => (
        <div key={group} className="flex flex-col">
          <div className="flex h-7 items-center gap-2 bg-background px-3.5">
            <Skeleton className="size-1.5 rounded-full" />
            <Skeleton className="h-2.5 w-16" />
            <Skeleton className="h-2.5 w-40" />
            <Skeleton className="ml-auto h-2.5 w-4" />
          </div>
          {Array.from({ length: rows }, (_, row) => (
            <div key={row} className={`${QUEUE_GRID_CLASS} mx-1.5 items-center px-2 py-2`}>
              <div className="flex flex-col gap-1.5">
                <Skeleton className="h-2.5 w-24" />
                <Skeleton className="h-2 w-20" />
              </div>
              <div className="flex flex-col gap-1.5">
                <Skeleton className="h-2.5 w-3/5" />
                <Skeleton className="h-2 w-2/5" />
              </div>
              <div className="flex flex-col gap-1.5">
                <Skeleton className="h-2.5 w-4/5" />
                <Skeleton className="h-2 w-3/5" />
              </div>
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

const QUEUE_SKELETON_GROUPS = [3, 5, 3] as const;

/**
 * The timeline while it loads, on the real canvas: the window's own axis, night hours and now
 * line, the pool lane, then driver lanes at the loaded lanes' geometry.
 */
export function DispatchTimelineSkeleton({
  scale,
  today,
}: {
  scale: TimelineScale;
  today: number;
}) {
  const t = useT();
  const items: TimelineSheetItem[] = [
    {
      key: "group:pool",
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={<Skeleton className="h-2.5 w-24" />}
          count={<Skeleton className="h-2.5 w-4" />}
        />
      ),
    },
    {
      key: "lane:pool",
      height: ROW_HEIGHT,
      node: (
        <TimelineRow
          scale={scale}
          height={ROW_HEIGHT}
          label={<TimelineLaneLabelSkeleton index={0} />}
          lane={
            <>
              <TimelineBarSkeleton scale={scale} index={3} top={LANE_PAD} />
              <TimelineBarSkeleton scale={scale} index={8} top={LANE_PAD} />
            </>
          }
        />
      ),
    },
    {
      key: "group:drivers",
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={<Skeleton className="h-2.5 w-16" />}
          count={<Skeleton className="h-2.5 w-4" />}
        />
      ),
    },
  ];
  for (let i = 1; i <= TIMELINE_DRIVER_ROWS; i++) {
    items.push({
      key: `lane:${i}`,
      height: ROW_HEIGHT,
      node: (
        <TimelineRow
          scale={scale}
          height={ROW_HEIGHT}
          label={<TimelineLaneLabelSkeleton index={i} />}
          lane={i % 3 === 2 ? null : <TimelineBarSkeleton scale={scale} index={i} top={LANE_PAD} />}
        />
      ),
    });
  }

  return (
    <div
      className="relative min-h-0 flex-1"
      aria-busy="true"
      aria-label={t("Loading the timeline")}
    >
      <TimelineCanvas
        scale={scale}
        today={today}
        labelHeading={t("Move or driver")}
        items={items}
      />
    </div>
  );
}

export function InspectorSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-3 p-3.5" aria-busy="true" aria-label={t("Loading move")}>
      <div className="flex flex-col gap-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-4 w-56" />
        <Skeleton className="h-2.5 w-44" />
        <div className="mt-2 grid grid-cols-[6rem_minmax(0,1fr)] gap-x-3 gap-y-2">
          {Array.from({ length: 4 }, (_, index) => (
            <div key={index} className="contents">
              <Skeleton className="h-2.5 w-16" />
              <Skeleton className="h-2.5 w-40" />
            </div>
          ))}
        </div>
      </div>
      <Skeleton className="h-44 rounded-md" />
      <div className="flex flex-col gap-1 rounded-md border border-border p-1">
        {Array.from({ length: 4 }, (_, index) => (
          <Skeleton key={index} className="h-11 rounded-md" />
        ))}
      </div>
    </div>
  );
}
