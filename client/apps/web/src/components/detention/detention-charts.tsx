import { cn } from "@trenova/shared/lib/utils";
import { m } from "motion/react";
import { useState } from "react";

/** One easing for every bar in the detention surfaces, so they feel like one system. */
export const BAR_TRANSITION = { duration: 0.45, ease: [0.22, 1, 0.36, 1] } as const;

export type ShareSegment = {
  key: string;
  label: string;
  value: number;
  /** Fill utility for the segment, e.g. `bg-emerald-500 dark:bg-emerald-400`. */
  className: string;
  /** Secondary legend text — usually the formatted amount. */
  caption?: string;
};

/**
 * A single track split by share, plus a legend that cross-highlights it.
 *
 * Pointing at a legend entry dims every other segment, which is the fastest way
 * to answer "which slice is that" without a tooltip chasing the cursor; clicking
 * pins the highlight so the comparison survives the cursor moving away.
 */
export function ShareBreakdown({
  segments,
  className,
  trackClassName,
}: {
  segments: readonly ShareSegment[];
  className?: string;
  trackClassName?: string;
}) {
  const [pinned, setPinned] = useState<string | null>(null);
  const [hovered, setHovered] = useState<string | null>(null);
  const active = pinned ?? hovered;

  const visible = segments.filter((segment) => segment.value > 0);
  const total = visible.reduce((sum, segment) => sum + segment.value, 0);

  if (total <= 0) {
    return null;
  }

  return (
    <div className={className}>
      <div
        className={cn(
          "bg-muted flex h-2.5 w-full gap-px overflow-hidden rounded-full",
          trackClassName,
        )}
      >
        {visible.map((segment, index) => (
          <m.div
            key={segment.key}
            className={cn(
              "h-full transition-opacity duration-200 first:rounded-l-full last:rounded-r-full",
              segment.className,
              active && active !== segment.key && "opacity-25",
            )}
            initial={{ width: 0 }}
            animate={{ width: `${(segment.value / total) * 100}%` }}
            transition={{ ...BAR_TRANSITION, delay: index * 0.06 }}
          />
        ))}
      </div>

      <div className="mt-2.5 flex flex-wrap gap-x-4 gap-y-1.5">
        {visible.map((segment) => (
          <button
            key={segment.key}
            type="button"
            aria-pressed={pinned === segment.key}
            onPointerEnter={() => setHovered(segment.key)}
            onPointerLeave={() => setHovered(null)}
            onFocus={() => setHovered(segment.key)}
            onBlur={() => setHovered(null)}
            onClick={() =>
              setPinned((current) => (current === segment.key ? null : segment.key))
            }
            className={cn(
              "text-2xs hover:text-foreground focus-visible:ring-ring/50 inline-flex items-center gap-1.5 rounded-sm transition-colors outline-none focus-visible:ring-[3px]",
              pinned === segment.key ? "text-foreground" : "text-muted-foreground",
            )}
          >
            <span className={cn("size-2 shrink-0 rounded-full", segment.className)} />
            <span className="font-medium">{segment.label}</span>
            {segment.caption ? (
              <span className="tabular-nums">{segment.caption}</span>
            ) : null}
            <span className="tabular-nums opacity-60">
              {Math.round((segment.value / total) * 100)}%
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}

/** A hairline progress track for a single 0–1 ratio. */
export function Meter({
  value,
  className,
  barClassName,
  delay = 0,
}: {
  value: number;
  className?: string;
  barClassName?: string;
  delay?: number;
}) {
  const clamped = Math.min(Math.max(value, 0), 1);

  return (
    <div className={cn("bg-muted h-1 w-full overflow-hidden rounded-full", className)}>
      <m.div
        className={cn("bg-foreground h-full rounded-full", barClassName)}
        initial={{ width: 0 }}
        animate={{ width: `${clamped * 100}%` }}
        transition={{ ...BAR_TRANSITION, delay }}
      />
    </div>
  );
}

/**
 * A bar that grows out of a centre line so gains and losses read as directions
 * rather than as signs a reader has to parse.
 */
export function DivergingBar({
  value,
  scale,
  className,
  delay = 0,
}: {
  value: number;
  scale: number;
  className?: string;
  delay?: number;
}) {
  const positive = value >= 0;
  const width = scale <= 0 ? 0 : (Math.abs(value) / scale) * 50;

  return (
    <div className={cn("bg-muted relative h-1.5 w-24 shrink-0 rounded-full", className)}>
      <span className="bg-border absolute inset-y-0 left-1/2 w-px -translate-x-1/2" />
      <m.span
        className={cn(
          "absolute inset-y-0 rounded-full",
          positive
            ? "left-1/2 bg-emerald-500 dark:bg-emerald-400"
            : "right-1/2 bg-red-500 dark:bg-red-400",
        )}
        initial={{ width: 0 }}
        animate={{ width: `${width}%` }}
        transition={{ ...BAR_TRANSITION, delay }}
      />
    </div>
  );
}

/**
 * Median-to-p90 dwell on a shared scale. The solid run is the typical stop and
 * the faded run is the tail — the gap between them is the planning risk, which
 * a single average would erase.
 */
export function DwellSpreadRail({
  median,
  p90,
  scale,
  className,
  delay = 0,
}: {
  median: number;
  p90: number;
  scale: number;
  className?: string;
  delay?: number;
}) {
  const share = (value: number) =>
    scale <= 0 ? 0 : Math.min(Math.max(value / scale, 0), 1) * 100;

  return (
    <div
      className={cn(
        "bg-muted relative h-1.5 w-full overflow-hidden rounded-full",
        className,
      )}
    >
      <m.span
        className="bg-foreground/25 absolute inset-y-0 left-0 rounded-full"
        initial={{ width: 0 }}
        animate={{ width: `${share(p90)}%` }}
        transition={{ ...BAR_TRANSITION, delay }}
      />
      <m.span
        className="bg-foreground absolute inset-y-0 left-0 rounded-full"
        initial={{ width: 0 }}
        animate={{ width: `${share(median)}%` }}
        transition={{ ...BAR_TRANSITION, delay: delay + 0.06 }}
      />
    </div>
  );
}
