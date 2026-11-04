import { cn } from "@trenova/shared/lib/utils";
import { m } from "motion/react";

/** One easing for every bar in the detention surfaces, so they feel like one system. */
export const BAR_TRANSITION = { duration: 0.45, ease: [0.22, 1, 0.36, 1] } as const;

/**
 * A bar that grows out of a centre line so gains and losses read as directions
 * rather than as signs a reader has to parse.
 */
export function DivergingBar({
  value,
  scale,
  className,
}: {
  value: number;
  scale: number;
  className?: string;
}) {
  const positive = value >= 0;
  const width = scale <= 0 ? 0 : (Math.abs(value) / scale) * 50;

  return (
    <div className={cn("bg-muted relative h-1.5 w-24 shrink-0 rounded-full", className)}>
      <span className="bg-border absolute inset-y-0 left-1/2 w-px -translate-x-1/2" />
      <m.span
        className={cn(
          "absolute inset-y-0 rounded-full",
          positive ? "left-1/2 bg-success" : "right-1/2 bg-danger",
        )}
        initial={false}
        animate={{ width: `${width}%` }}
        transition={BAR_TRANSITION}
      />
    </div>
  );
}
