import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { m, useReducedMotion } from "motion/react";

export type CompositionSegment = {
  key: string;
  label: string;
  value: number;
  /** Fill for the segment. Defaults step down in opacity so one hue carries the whole bar. */
  className?: string;
};

type CompositionBarProps = {
  segments: CompositionSegment[];
  /** Denominator when the segments do not add up to the whole. */
  total?: number;
  size?: "sm" | "default";
  showLegend?: boolean;
  /** How a value reads in the legend and the accessible name; defaults to the raw number. */
  formatValue?: (value: number) => string;
  className?: string;
  "aria-label"?: string;
};

const identity = (value: number) => String(value);

const DEFAULT_FILLS = ["bg-brand", "bg-brand/55", "bg-brand/30", "bg-brand/15"];

/**
 * One whole split into its parts: a single track whose segments share a hue
 * and step down in weight, so the reader sees proportion before colour. A
 * segment worth nothing is left out rather than drawn as a sliver.
 */
export function CompositionBar({
  segments,
  total,
  size = "default",
  showLegend = true,
  formatValue = identity,
  className,
  "aria-label": ariaLabel,
}: CompositionBarProps) {
  const t = useT();

  const reduceMotion = useReducedMotion();
  const sum = segments.reduce((acc, segment) => acc + Math.max(0, segment.value), 0);
  const denominator = Math.max(total ?? sum, sum, 0);
  const drawn = segments.filter((segment) => segment.value > 0);
  const summary = segments
    .map((segment) => `${segment.label} ${formatValue(segment.value)}`)
    .join(", ");

  return (
    <div className={cn("flex min-w-0 flex-col gap-1.5", className)}>
      <div
        role="img"
        aria-label={ariaLabel ? `${ariaLabel}: ${summary}` : summary}
        className={cn(
          "flex w-full overflow-hidden rounded-full bg-muted",
          size === "sm" ? "h-1" : "h-1.5",
        )}
      >
        {denominator > 0
          ? drawn.map((segment, index) => {
              const share = (segment.value / denominator) * 100;
              return (
                <m.span
                  key={segment.key}
                  data-slot="composition-segment"
                  className={cn(
                    "h-full shrink-0 not-first:ml-px",
                    segment.className ?? DEFAULT_FILLS[index % DEFAULT_FILLS.length],
                  )}
                  initial={reduceMotion ? false : { width: 0 }}
                  animate={{ width: `${share}%` }}
                  transition={{ duration: 0.6, delay: index * 0.06, ease: "easeOut" }}
                />
              );
            })
          : null}
      </div>
      {showLegend ? (
        <dl className="flex flex-wrap gap-x-3 gap-y-0.5 text-2xs text-muted-foreground">
          {segments.map((segment, index) => (
            <div key={segment.key} className="flex items-center gap-1">
              <span
                aria-hidden
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  segment.className ?? DEFAULT_FILLS[index % DEFAULT_FILLS.length],
                )}
              />
              <dt>{t(segment.label)}</dt>
              <dd className="font-medium text-foreground tabular-nums">
                {formatValue(segment.value)}
              </dd>
            </div>
          ))}
        </dl>
      ) : null}
    </div>
  );
}
