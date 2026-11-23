import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useEffect, useState, type ReactNode } from "react";

/** How long the notes take to fold away, and when they leave the page after. */
export const NOTES_LEAVE_MS = 220;

/**
 * The running turn's working notes, which fold away once the board they led to
 * is known: their height closes to nothing as they fade, with no measuring,
 * and when the fold ends they are gone from the page. While they leave nobody
 * can reach them, by keyboard or by screen reader. With reduced motion they
 * simply are not drawn. Notes that are already leaving when first drawn are
 * not drawn at all, so nothing replays.
 */
export function DeskLeavingNotes({ leaving, children }: { leaving: boolean; children: ReactNode }) {
  const reduceMotion = useReducedMotion() ?? false;
  const [gone, setGone] = useState(leaving);
  if (!leaving && gone) setGone(false);

  useEffect(() => {
    if (!leaving) return;
    const timer = window.setTimeout(() => setGone(true), NOTES_LEAVE_MS);
    return () => window.clearTimeout(timer);
  }, [leaving]);

  if (leaving && (gone || reduceMotion)) return null;
  return (
    <div
      className={cn(
        "grid transition-[grid-template-rows,opacity] duration-220 ease-out motion-reduce:transition-none",
        leaving ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr]",
      )}
      inert={leaving}
      aria-hidden={leaving || undefined}
      data-leaving={leaving || undefined}
    >
      <div className={cn("min-h-0", leaving && "overflow-hidden")}>{children}</div>
    </div>
  );
}
