import { URGENCY_STYLES, deskClockTrack } from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import type { DeskEntry } from "@trenova/shared/types/detention";
import { m } from "motion/react";
import { memo } from "react";

const WIDTH_TRANSITION = { type: "spring", stiffness: 180, damping: 30, mass: 0.7 } as const;

type DeskClockTrackProps = {
  entry: DeskEntry;
  nowSeconds: number;
  className?: string;
};

/**
 * One hairline spanning arrival to the last deadline that matters. Free time is
 * faint, the billing portion beyond it is solid, and the notice cutoff is a
 * single tick the accrual can be seen approaching — so the moment a stop turns
 * from waiting into billing is readable without parsing three timestamps.
 */
export const DeskClockTrack = memo(function DeskClockTrack({
  entry,
  nowSeconds,
  className,
}: DeskClockTrackProps) {
  const track = deskClockTrack(entry, nowSeconds);
  const styles = URGENCY_STYLES[entry.urgency];
  const consumedPercent = Math.min(track.elapsedPercent, track.freePercent);

  return (
    <div aria-hidden className={cn("bg-border relative h-1 w-full rounded-full", className)}>
      <m.div
        className="bg-foreground/15 absolute inset-y-0 left-0 rounded-full"
        initial={false}
        animate={{ width: `${consumedPercent}%` }}
        transition={WIDTH_TRANSITION}
      />
      <m.div
        className={cn("absolute inset-y-0 rounded-full", styles.bar)}
        style={{ left: `${track.freePercent}%` }}
        initial={false}
        animate={{ width: `${track.accruedPercent}%` }}
        transition={WIDTH_TRANSITION}
      />
      {track.noticePercent !== null && !track.noticePassed && (
        <span
          className="absolute -top-0.5 -bottom-0.5 w-px bg-amber-500"
          style={{ left: `${track.noticePercent}%` }}
        />
      )}
    </div>
  );
});
