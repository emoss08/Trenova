import { useNow, type ClockGranularity } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

/** Below this the countdown shows seconds and so moves every second. */
const SECONDS_SHOWN_WITHIN = 120;

export type CountdownThresholds = {
  /** Shown as a warning once the time is this many minutes away. */
  warnWithinMinutes?: number;
  /** Shown as danger once the time is this many minutes away, and when it has passed. */
  dangerWithinMinutes?: number;
};

type CountdownCellProps = CountdownThresholds & {
  /** The moment counted down to, in Unix seconds; nothing is shown without one. */
  target: number | null | undefined;
};

function granularityFor(remaining: number): ClockGranularity {
  return Math.abs(remaining) < SECONDS_SHOWN_WITHIN ? "second" : "minute";
}

/**
 * How long until a moment (an appointment, a detention clock, an hours-of-service
 * limit) or how long since it passed. It reads a clock shared by every countdown on
 * the page, so a page of them costs one timer, and only these cells redraw as it
 * moves; the row around them never does.
 */
export function CountdownCell({
  target,
  warnWithinMinutes = 60,
  dangerWithinMinutes = 15,
}: CountdownCellProps) {
  const t = useT();
  // The minute clock decides whether this countdown needs the second clock, so
  // only one in its last two minutes moves every second.
  const minute = useNow("minute");
  const now = useNow(target == null ? "minute" : granularityFor(target - minute));

  if (target == null) {
    return <span className="text-muted-foreground">—</span>;
  }

  const remaining = target - now;
  const magnitude = Math.abs(remaining);
  const amount =
    magnitude < SECONDS_SHOWN_WITHIN && magnitude < 60
      ? t("{0}s", Math.floor(magnitude))
      : formatDurationFromSeconds(magnitude);
  const late = remaining < 0;
  const tone = late
    ? "danger"
    : remaining <= dangerWithinMinutes * 60
      ? "danger"
      : remaining <= warnWithinMinutes * 60
        ? "warning"
        : "neutral";

  return (
    <span
      data-tone={tone}
      className={cn(
        "tabular-nums",
        tone === "danger" && "text-danger-foreground",
        tone === "warning" && "text-warning-foreground",
      )}
    >
      {late ? t("{0} late", amount) : t("in {0}", amount)}
    </span>
  );
}
