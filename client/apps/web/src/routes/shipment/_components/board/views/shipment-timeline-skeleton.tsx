import type { TimelineRowMode } from "@/lib/shipment-board/timeline";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  TimelineCanvas,
  TimelineGroupRow,
  TimelineRow,
  type TimelineSheetItem,
} from "@/components/timeline/timeline-canvas";
import {
  BAR_HEIGHT,
  GROUP_HEIGHT,
  LANE_PAD,
  ROW_HEIGHT,
  type TimelineScale,
} from "@/components/timeline/timeline-scale";
import {
  SKELETON_LABEL_WIDTHS,
  TimelineBarSkeleton,
  TimelineLaneLabelSkeleton,
} from "@/components/timeline/timeline-skeleton-parts";
import { useTimelineRange } from "./timeline/use-timeline-range";

/**
 * How many rows each group the skeleton draws holds. The timeline passes the
 * groups of the range it is leaving, so stepping a day keeps the sheet's shape;
 * a first load fills the view with a plausible board.
 */
export type TimelineSkeletonShape = readonly number[];

const FIRST_LOAD_SHAPE: Record<TimelineRowMode, TimelineSkeletonShape> = {
  shipments: [2, 5, 9],
  coverage: [1, 10, 2],
};
/** The skeleton never draws more rows than a tall screen shows. */
const MAX_ROWS = 22;

/**
 * The timeline while it loads: the real toolbar's shapes, the real axis for
 * the range in the URL, and rows at the real rows' sizes, grouped as the
 * loaded sheet will be.
 */
export function ShipmentTimelineSkeleton({ shape }: { shape?: TimelineSkeletonShape }) {
  const t = useT();
  const { scale, today, mode, sheetRef } = useTimelineRange();

  return (
    <div
      ref={sheetRef}
      className="@container/timeline flex h-full min-h-105 flex-col"
      aria-busy="true"
      aria-label={t("Loading the timeline")}
    >
      <TimelineToolbarSkeleton />
      <ShipmentTimelineBodySkeleton scale={scale} today={today} mode={mode} shape={shape} />
    </div>
  );
}

/** The toolbar's controls at their sizes: rows, the day stepper, the range, the stage chips. */
function TimelineToolbarSkeleton() {
  return (
    <div
      aria-hidden
      className="border-subtle bg-card flex min-h-10 shrink-0 flex-wrap items-center gap-x-3 gap-y-1.5 border-b px-2.5 py-1.5"
    >
      <Skeleton className="h-7.5 w-48 rounded-lg" />
      <div className="flex items-center gap-1">
        <Skeleton className="size-7 rounded-md" />
        <Skeleton className="mx-2 h-3.5 w-32" />
        <Skeleton className="size-7 rounded-md" />
        <Skeleton className="h-7 w-12 rounded-md" />
      </div>
      <Skeleton className="h-7.5 w-40 rounded-lg" />
      <div className="ml-auto flex items-center gap-1">
        <Skeleton className="h-6 w-32 rounded-md" />
        <Skeleton className="h-6 w-24 rounded-md" />
        <Skeleton className="h-6 w-28 rounded-md" />
      </div>
    </div>
  );
}

/**
 * The sheet under the toolbar while a range loads, drawn on the real canvas:
 * its axis, night hours and now line are the range's own, and its rows and
 * lanes take the real rows' geometry.
 */
export function ShipmentTimelineBodySkeleton({
  scale,
  today,
  mode,
  shape,
}: {
  scale: TimelineScale;
  today: number;
  mode: TimelineRowMode;
  shape?: TimelineSkeletonShape;
}) {
  const t = useT();
  const groups = capRows(shape && shape.length > 0 ? shape : FIRST_LOAD_SHAPE[mode]);
  const items: TimelineSheetItem[] = [];
  let index = 0;
  groups.forEach((rows, group) => {
    items.push({
      key: `group:${group}`,
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={<Skeleton className="h-2.5 w-24" />}
          count={<Skeleton className="h-2.5 w-4" />}
        />
      ),
    });
    for (let row = 0; row < rows; row++) {
      const i = index++;
      items.push({
        key: `row:${i}`,
        height: ROW_HEIGHT,
        node:
          mode === "shipments" ? (
            <TimelineRow
              scale={scale}
              height={ROW_HEIGHT}
              label={<ShipmentLabelSkeleton index={i} />}
              lane={
                <TimelineBarSkeleton scale={scale} index={i} top={(ROW_HEIGHT - BAR_HEIGHT) / 2} />
              }
            />
          ) : (
            <TimelineRow
              scale={scale}
              height={ROW_HEIGHT}
              label={<TimelineLaneLabelSkeleton index={i} />}
              lane={
                <>
                  <TimelineBarSkeleton scale={scale} index={i} top={LANE_PAD} />
                  {i % 3 === 1 ? (
                    <TimelineBarSkeleton scale={scale} index={i + 7} top={LANE_PAD} />
                  ) : null}
                </>
              }
            />
          ),
      });
    }
  });

  return (
    <div className="relative min-h-0 flex-1" aria-hidden>
      <TimelineCanvas
        scale={scale}
        today={today}
        labelHeading={mode === "shipments" ? t("Shipment") : t("Driver or carrier")}
        items={items}
      />
    </div>
  );
}

function capRows(shape: TimelineSkeletonShape): number[] {
  let left = MAX_ROWS;
  return shape
    .map((rows) => {
      const take = Math.max(0, Math.min(rows, left));
      left -= take;
      return take;
    })
    .filter((rows) => rows > 0);
}

/** A shipment row's label: the lane, then the PRO and customer, then who has it. */
function ShipmentLabelSkeleton({ index }: { index: number }) {
  return (
    <>
      <span className="flex min-w-0 flex-1 flex-col gap-1.5">
        <Skeleton
          className={cn("h-3", SKELETON_LABEL_WIDTHS[index % SKELETON_LABEL_WIDTHS.length])}
        />
        <Skeleton className="h-2.5 w-24" />
      </span>
      <Skeleton className="size-5.5 shrink-0 rounded-full" />
    </>
  );
}
