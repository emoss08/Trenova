import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type CSSProperties, type ReactNode } from "react";

const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;

export type HourRulerTone = "normal" | "warning" | "danger";

/** Warning under an hour left, danger at none: an hour is the same problem on any ruleset. */
export function hourRulerTone(remainingMs: number): HourRulerTone {
  if (remainingMs <= 0) return "danger";
  if (remainingMs < HOUR_MS) return "warning";
  return "normal";
}

/** The part of the limit already used, from 0 to 1. */
export function hourRulerUsed(remainingMs: number, limitMs: number): number {
  if (limitMs <= 0) return 1;
  return Math.min(1, Math.max(0, (limitMs - Math.max(0, remainingMs)) / limitMs));
}

const FILL_CLASS: Record<HourRulerTone, string> = {
  normal: "bg-ink",
  warning: "bg-warning",
  danger: "bg-danger",
};

const TICKS_STYLE: CSSProperties = {
  backgroundImage:
    "repeating-linear-gradient(to right, transparent 0, transparent calc(var(--tick) - 1px), var(--color-card) calc(var(--tick) - 1px), var(--color-card) var(--tick))",
};

export type HourRulerProps = {
  /** The clock's name ("Drive"). */
  name: string;
  remainingMs: number;
  limitMs: number;
  /**
   * What the limit is, after the time left: "left of 11h" in the compact size, "of 11h
   * driving" in the large one.
   */
  caption: string;
  /** One tick per this many milliseconds of the limit; an hour by default (the cycle uses 10). */
  tickMs?: number;
  /** `compact` for a panel of clocks (Overview's Right now), `large` for the hours band. */
  size?: "compact" | "large";
  /** A line under a large ruler ("4h 48m driven today"). */
  note?: ReactNode;
  className?: string;
};

/**
 * An hours-of-service clock as a ruled bar: its name, the time left, what the limit is, and a
 * track with one tick per hour whose used part is filled, in warning under an hour left and in
 * danger at none. It fills once, the first time it is shown, and moves when the numbers do.
 */
export function HourRuler({
  name,
  remainingMs,
  limitMs,
  caption,
  tickMs = HOUR_MS,
  size = "compact",
  note,
  className,
}: HourRulerProps) {
  const t = useT();
  const [drawn, setDrawn] = useState(false);
  useEffect(() => {
    if (drawn) return;
    const frame = requestAnimationFrame(() => setDrawn(true));
    return () => cancelAnimationFrame(frame);
  }, [drawn]);

  const remaining = Math.max(0, remainingMs);
  const totalMinutes = Math.floor(remaining / MINUTE_MS);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  const left = hours > 0 ? t("{0}h {1}m", hours, minutes) : t("{0}m", minutes);
  const tone = hourRulerTone(remainingMs);
  const used = hourRulerUsed(remainingMs, limitMs);
  const ticks = Math.max(1, Math.round(limitMs / tickMs));
  const large = size === "large";

  return (
    <div className={cn("flex min-w-0 flex-col", large ? "gap-0" : "gap-1.5", className)}>
      {large ? (
        <>
          <span className="text-xs font-medium text-foreground-subtle">{name}</span>
          <div className="mt-1 mb-2.5 flex flex-wrap items-baseline gap-x-2 gap-y-1">
            <b className="text-xl font-semibold tabular-nums">{left}</b>
            <span className="text-muted-foreground">{caption}</span>
          </div>
        </>
      ) : (
        <div className="flex items-baseline justify-between gap-2">
          <span className="text-xs font-medium text-foreground-subtle">{name}</span>
          <span className="tabular-nums">
            <b className="font-medium">{left}</b>{" "}
            <span className="text-muted-foreground">{caption}</span>
          </span>
        </div>
      )}
      <div
        role="meter"
        aria-label={name}
        aria-valuemin={0}
        aria-valuemax={Math.round(limitMs / MINUTE_MS)}
        aria-valuenow={totalMinutes}
        aria-valuetext={t("{0} left", left)}
        data-tone={tone}
        style={{ "--tick": `calc(100% / ${ticks})` } as CSSProperties}
        className={cn(
          "relative overflow-hidden rounded-xs bg-surface-active",
          large ? "h-2.5" : "h-2",
        )}
      >
        <span
          className={cn(
            "absolute inset-0 origin-left transition-transform duration-420 ease-settle motion-reduce:transition-none",
            FILL_CLASS[tone],
          )}
          style={{ transform: `scaleX(${drawn ? used : 0})` }}
        />
        <span aria-hidden className="pointer-events-none absolute inset-0" style={TICKS_STYLE} />
      </div>
      {large && note ? <p className="mt-2 text-xs text-muted-foreground">{note}</p> : null}
    </div>
  );
}
