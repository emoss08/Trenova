import {
  laneGaps,
  nextStartsInRow,
  type PackedLane,
  type TimelineSpan,
} from "@/lib/timeline/lanes";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { Fragment, type ReactNode } from "react";
import { formatTimelineTime } from "./timeline-format";
import { BAR_HEIGHT, LANE_PAD, laneBarTop, type TimelineScale } from "./timeline-scale";

/** Free-time labels need this much room to be read. */
const MIN_GAP_LABEL_PX = 64;
/** A free-time label never stretches wider than this, however long the gap. */
const MAX_GAP_LABEL_PX = 360;

/** A lane label's text: the name with an optional flag beside it, and a line of detail. */
export function TimelineLaneLabel({
  title,
  meta,
  flag,
}: {
  title: string;
  meta: ReactNode;
  flag?: ReactNode;
}) {
  return (
    <span className="flex min-w-0 flex-1 flex-col leading-tight">
      <span className="flex min-w-0 items-center gap-1.5 text-sm font-medium">
        <span className="truncate">{title}</span>
        {flag}
      </span>
      <span className="text-muted-foreground flex min-w-0 gap-1 text-xs whitespace-nowrap">
        {meta}
      </span>
    </span>
  );
}

/** The dot beside a lane's name when two of its bars overlap. */
export function TimelineConflictFlag() {
  const t = useT();
  return (
    <span
      title={t("Double-booked")}
      aria-label={t("Double-booked")}
      className="bg-danger size-1.5 shrink-0 rounded-full"
    />
  );
}

/**
 * A lane's free time from now to the range's end, each stretch labeled on a dashed rule
 * along the lane's first row: where new work fits, read without hovering anything.
 */
export function TimelineLaneGaps({
  items,
  scale,
}: {
  items: readonly Pick<TimelineSpan, "start" | "end">[];
  scale: TimelineScale;
}) {
  const t = useT();
  const now = useNow("minute");
  return (
    <>
      {laneGaps(items, Math.max(scale.start, now), scale.end).map((gap) => {
        const left = scale.x(gap.start);
        const width = scale.x(gap.end) - left;
        if (width < MIN_GAP_LABEL_PX) return null;
        const text =
          items.length === 0
            ? t("Nothing booked")
            : gap.open
              ? t("Free from {0}", formatTimelineTime(gap.start, scale))
              : t("Free {0}", formatMinutesSpan((gap.end - gap.start) / 60));
        return (
          <span
            key={gap.start}
            aria-hidden
            className="tl-gap pointer-events-none absolute flex -translate-y-1/2 items-center justify-center"
            style={{
              left,
              width: Math.min(width, MAX_GAP_LABEL_PX),
              top: LANE_PAD + BAR_HEIGHT / 2,
            }}
          >
            <span className="bg-card text-muted-foreground relative px-1.5 font-mono text-2xs whitespace-nowrap">
              {text}
            </span>
          </span>
        );
      })}
    </>
  );
}

export type LaneBarPlace = {
  top: number;
  /** Where the next bar in the same row starts, in pixels, when there is one. */
  nextLeft?: number;
  conflict: boolean;
};

/** A lane's bars at their packed rows; each page draws its own bar in the place given. */
export function TimelineLaneBars<T extends TimelineSpan>({
  packed,
  scale,
  renderBar,
}: {
  packed: PackedLane<T>;
  scale: TimelineScale;
  renderBar: (item: T, place: LaneBarPlace) => ReactNode;
}) {
  const next = nextStartsInRow(packed);
  return (
    <>
      {packed.placed.map(({ item, row }) => {
        const after = next.get(item.id);
        return (
          <Fragment key={item.id}>
            {renderBar(item, {
              top: laneBarTop(row),
              nextLeft: after === undefined ? undefined : scale.x(Math.max(after, scale.start)),
              conflict: packed.conflicts.has(item.id),
            })}
          </Fragment>
        );
      })}
    </>
  );
}
