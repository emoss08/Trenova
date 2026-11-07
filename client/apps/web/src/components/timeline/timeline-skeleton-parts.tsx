import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { BAR_HEIGHT, type TimelineScale } from "./timeline-scale";

export const SKELETON_LABEL_WIDTHS = ["w-32", "w-28", "w-36", "w-24", "w-30"] as const;

/** A lane's label while it loads: the avatar, then the name and a line of detail. */
export function TimelineLaneLabelSkeleton({ index }: { index: number }) {
  return (
    <>
      <Skeleton className="size-5.5 shrink-0 rounded-full" />
      <span className="flex min-w-0 flex-1 flex-col gap-1.5">
        <Skeleton
          className={cn("h-3", SKELETON_LABEL_WIDTHS[(index + 2) % SKELETON_LABEL_WIDTHS.length])}
        />
        <Skeleton className="h-2.5 w-20" />
      </span>
    </>
  );
}

/** Hours a placeholder bar runs, so it reads at the scale of real work for the days shown. */
function barHours(days: number): readonly [number, number] {
  if (days <= 1) return [3, 9];
  if (days <= 3) return [8, 30];
  return [16, 56];
}

/**
 * A placeholder bar at real work's scale for the range: a few hours on a day, a day or two
 * on a week. Its place is fixed by its row, so the sheet holds still.
 */
export function TimelineBarSkeleton({
  scale,
  index,
  top,
}: {
  scale: TimelineScale;
  index: number;
  top: number;
}) {
  const [minHours, maxHours] = barHours(scale.days.length - 1);
  const hours = minHours + ((index * 7) % (maxHours - minHours + 1));
  const width = Math.min(scale.x(scale.start + hours * 3_600), scale.width * 0.6);
  const left = (((index * 37) % 64) / 100) * (scale.width - width) + scale.width * 0.04;

  return (
    <Skeleton className="absolute rounded-[5px]" style={{ left, width, top, height: BAR_HEIGHT }} />
  );
}
