import { CheckIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId } from "react";
import { useCountUp } from "./use-briefing-motion";

const RADIUS = 46;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;
/** The sliver of fill a ring with nothing cleared still shows, so it reads as a ring. */
const MIN_FILL = 0.02;
/** The arc that sweeps the track while the briefing is on its way. */
const SWEEP = 0.18;

/**
 * How much is left for you, and how much of the day's work has cleared. While
 * the briefing loads one arc sweeps the empty track; then the fill runs to
 * its value and the number counts up. With nothing left it is a check.
 */
export function Ring({
  progress,
  count,
  busy,
}: {
  /** Between 0 and 1. */
  progress: number;
  count: number;
  busy: boolean;
}) {
  const t = useT();
  const gradient = `tb-ring-${useId().replace(/[^a-zA-Z0-9_-]/g, "")}`;
  const shown = useCountUp(count, !busy);
  const full = !busy && progress >= 1;
  const offset = busy ? CIRCUMFERENCE : CIRCUMFERENCE * (1 - Math.max(MIN_FILL, progress));

  return (
    <div
      className="tb-ring relative size-27 shrink-0"
      data-busy={busy}
      role="img"
      aria-label={
        busy
          ? t("Going through the day")
          : full
            ? t("Nothing left for you")
            : t("{0, plural, one {# left for you} other {# left for you}}", count)
      }
    >
      <svg viewBox="0 0 108 108" className="size-full -rotate-90" aria-hidden>
        <defs>
          <linearGradient id={gradient} x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" style={{ stopColor: "var(--accent-indigo)" }} />
            <stop offset="0.55" style={{ stopColor: "var(--accent-violet)" }} />
            <stop offset="1" style={{ stopColor: "var(--accent-rose)" }} />
          </linearGradient>
        </defs>
        <circle cx="54" cy="54" r={RADIUS} className="tb-ring-track" />
        <circle
          cx="54"
          cy="54"
          r={RADIUS}
          className="tb-ring-sweep"
          strokeDasharray={`${CIRCUMFERENCE * SWEEP} ${CIRCUMFERENCE}`}
        />
        <circle
          cx="54"
          cy="54"
          r={RADIUS}
          className="tb-ring-fill"
          stroke={`url(#${gradient})`}
          strokeDasharray={CIRCUMFERENCE}
          strokeDashoffset={offset}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center leading-none">
        {full ? (
          <CheckIcon aria-hidden className="animate-confirm size-7.5" strokeWidth={2.4} />
        ) : (
          <>
            <span className="font-mono text-3xl font-semibold tabular-nums transition-opacity duration-300">
              {busy ? "" : shown}
            </span>
            <span className="text-2xs text-foreground-subtle mt-1">{busy ? "" : t("for you")}</span>
          </>
        )}
      </div>
    </div>
  );
}
