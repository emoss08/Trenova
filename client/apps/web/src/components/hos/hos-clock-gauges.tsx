import { RingGauge, type RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

const HOUR_MS = 3_600_000;

/**
 * Regulatory ceilings, used only to size the gauges. Remaining values themselves always
 * come from the telematics feed, which may carry a shorter limit for a given ruleset.
 */
export const HOS_LIMITS_MS = {
  drive: 11 * HOUR_MS,
  shift: 14 * HOUR_MS,
  cycle: 70 * HOUR_MS,
} as const;

export type HosClockSeverity = "normal" | "warning" | "critical";

/**
 * Severity is read off the hours left, not the fraction of the limit left: an hour of
 * drive time is the same problem whether the ruleset allows 11 or 14.
 */
export function hosClockSeverity(remainingMs: number): HosClockSeverity {
  if (remainingMs < HOUR_MS) return "critical";
  if (remainingMs < 2 * HOUR_MS) return "warning";
  return "normal";
}

const SEVERITY_TONE: Record<HosClockSeverity, RingGaugeTone> = {
  normal: "brand",
  warning: "warning",
  critical: "critical",
};

const SEVERITY_TEXT: Record<HosClockSeverity, string> = {
  normal: "text-foreground",
  warning: "text-warning",
  critical: "text-destructive",
};

export function HosClockGauge({
  label,
  remainingMs,
  limitMs,
  severity,
  size = 24,
  strokeWidth = 3,
  className,
}: {
  label: string;
  remainingMs: number;
  limitMs: number;
  severity?: HosClockSeverity;
  size?: number;
  strokeWidth?: number;
  className?: string;
}) {
  const clamped = Math.max(0, remainingMs);
  const tone = severity ?? hosClockSeverity(clamped);
  const limitHours = Math.round(limitMs / HOUR_MS);

  return (
    <Tooltip>
      <TooltipTrigger
        render={<div className={cn("flex min-w-0 items-center gap-1.5", className)} />}
      >
        <RingGauge
          value={limitMs > 0 ? clamped / limitMs : 0}
          size={size}
          strokeWidth={strokeWidth}
          tone={SEVERITY_TONE[tone]}
          aria-label={`${label} time remaining`}
        />
        <div className="flex min-w-0 flex-col">
          <span className="text-muted-foreground text-[8.5px] leading-tight font-semibold tracking-wide uppercase">
            {label}
          </span>
          <span className={cn("text-[10.5px] leading-tight font-semibold tabular-nums", SEVERITY_TEXT[tone])}>
            {formatClockDurationMs(clamped)}
          </span>
        </div>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        {label}: {formatClockDurationMs(clamped)} of {limitHours}h remaining
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * The drive / shift / cycle trio a dispatcher reads before committing a load. An absent or
 * stale feed is missing information, not zero hours — drawing empty gauges would read as
 * "out of hours" and quietly remove a usable driver from consideration.
 */
export function HosClockGauges({
  driveRemainingMs,
  shiftRemainingMs,
  cycleRemainingMs,
  driveLimitMs = HOS_LIMITS_MS.drive,
  shiftLimitMs = HOS_LIMITS_MS.shift,
  cycleLimitMs = HOS_LIMITS_MS.cycle,
  hasViolation = false,
  hasFeed = true,
  isStale = false,
  size,
  strokeWidth,
  className,
}: {
  driveRemainingMs: number;
  shiftRemainingMs: number;
  cycleRemainingMs: number;
  driveLimitMs?: number;
  shiftLimitMs?: number;
  cycleLimitMs?: number;
  hasViolation?: boolean;
  hasFeed?: boolean;
  isStale?: boolean;
  size?: number;
  strokeWidth?: number;
  className?: string;
}) {
  if (!hasFeed) {
    return (
      <p className={cn("text-muted-foreground text-[10px]", className)}>No hours-of-service feed</p>
    );
  }

  if (isStale) {
    return (
      <p className={cn("text-warning text-[10px]", className)}>Hours-of-service data is stale</p>
    );
  }

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <HosClockGauge
        label="Drive"
        remainingMs={driveRemainingMs}
        limitMs={driveLimitMs}
        severity={hasViolation ? "critical" : undefined}
        size={size}
        strokeWidth={strokeWidth}
        className="flex-1"
      />
      <HosClockGauge
        label="Shift"
        remainingMs={shiftRemainingMs}
        limitMs={shiftLimitMs}
        severity={hasViolation ? "critical" : undefined}
        size={size}
        strokeWidth={strokeWidth}
        className="flex-1"
      />
      <HosClockGauge
        label="Cycle"
        remainingMs={cycleRemainingMs}
        limitMs={cycleLimitMs}
        size={size}
        strokeWidth={strokeWidth}
        className="flex-1"
      />
    </div>
  );
}
