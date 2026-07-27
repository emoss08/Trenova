import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { HOS_LIMITS_MS } from "./dispatch-vocabulary";

type ClockProps = {
  label: string;
  remainingMs: number;
  limitMs: number;
};

/**
 * A driver's clocks have to be readable at a glance in a dense list, but the exact figure
 * still has to be reachable — a dispatcher committing a load needs the number, not an
 * impression of it. Hence bars with the value on hover.
 */
function Clock({ label, remainingMs, limitMs }: ClockProps) {
  const ratio = limitMs > 0 ? Math.min(Math.max(remainingMs / limitMs, 0), 1) : 0;
  const percent = Math.round(ratio * 100);

  const tone =
    ratio <= 0.1
      ? "bg-red-500"
      : ratio <= 0.3
        ? "bg-amber-500"
        : "bg-green-500 dark:bg-green-400";

  return (
    <Tooltip>
      <TooltipTrigger
        render={<div className="flex flex-1 flex-col gap-0.5" aria-label={`${label} clock`} />}
      >
        <div className="h-1 w-full overflow-hidden rounded-full bg-muted">
          <div
            className={cn("h-full rounded-full transition-[width]", tone)}
            style={{ width: `${percent}%` }}
          />
        </div>
        <span className="text-[9px] uppercase tracking-wide text-muted-foreground">{label}</span>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        {label}: {formatClockDurationMs(remainingMs)} remaining
      </TooltipContent>
    </Tooltip>
  );
}

export function HosClockBars({
  driveRemainingMs,
  shiftRemainingMs,
  cycleRemainingMs,
  isStale,
  hasFeed,
  className,
}: {
  driveRemainingMs: number;
  shiftRemainingMs: number;
  cycleRemainingMs: number;
  isStale: boolean;
  hasFeed: boolean;
  className?: string;
}) {
  // An absent or stale feed is missing information, not zero hours. Drawing empty bars
  // would read as "out of hours" and quietly remove a usable driver from consideration.
  if (!hasFeed) {
    return (
      <p className={cn("text-[10px] text-muted-foreground", className)}>No hours-of-service feed</p>
    );
  }

  if (isStale) {
    return (
      <p className={cn("text-[10px] text-amber-600 dark:text-amber-400", className)}>
        Hours-of-service data is stale
      </p>
    );
  }

  return (
    <div className={cn("flex items-end gap-2", className)}>
      <Clock label="Drive" remainingMs={driveRemainingMs} limitMs={HOS_LIMITS_MS.drive} />
      <Clock label="Shift" remainingMs={shiftRemainingMs} limitMs={HOS_LIMITS_MS.shift} />
      <Clock label="Cycle" remainingMs={cycleRemainingMs} limitMs={HOS_LIMITS_MS.cycle} />
    </div>
  );
}
