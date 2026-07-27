import type { DispatchBoardDriver } from "@/lib/graphql/dispatch-console";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";

const HOUR_SECONDS = 3600;

export type TimelineZoom = 4 | 12 | 48;

type Span = { leftPercent: number; widthPercent: number };

/**
 * Clamps a block to the visible window and expresses it as percentages, so a commitment
 * that starts before the window or ends after it still renders as a partial bar rather
 * than disappearing.
 */
function spanWithin(
  start: number,
  end: number,
  windowStart: number,
  windowEnd: number,
): Span | null {
  if (windowEnd <= windowStart) return null;

  const clampedStart = Math.max(start, windowStart);
  const clampedEnd = Math.min(end, windowEnd);
  if (clampedEnd <= clampedStart) return null;

  const total = windowEnd - windowStart;
  return {
    leftPercent: ((clampedStart - windowStart) / total) * 100,
    widthPercent: ((clampedEnd - clampedStart) / total) * 100,
  };
}

function hourTicks(windowStart: number, windowEnd: number, zoom: TimelineZoom): number[] {
  const step = zoom <= 4 ? HOUR_SECONDS : zoom <= 12 ? 2 * HOUR_SECONDS : 6 * HOUR_SECONDS;
  const ticks: number[] = [];
  const first = Math.ceil(windowStart / step) * step;
  for (let tick = first; tick < windowEnd; tick += step) {
    ticks.push(tick);
  }
  return ticks;
}

function tickLabel(unixSeconds: number): string {
  return new Date(unixSeconds * 1000).toLocaleTimeString(undefined, {
    hour: "numeric",
    hour12: true,
  });
}

function DriverLane({
  driver,
  windowStart,
  windowEnd,
  onSelectMove,
}: {
  driver: DispatchBoardDriver;
  windowStart: number;
  windowEnd: number;
  onSelectMove: (moveId: string) => void;
}) {
  // Time off is drawn beneath commitments: a driver can be scheduled off and still hold
  // work that has not been moved yet, and hiding the conflict would be the wrong call.
  const timeOffSpans = driver.timeOff
    .map((pto) => ({ pto, span: spanWithin(pto.startDate, pto.endDate, windowStart, windowEnd) }))
    .filter((item) => item.span !== null);

  const commitmentSpans = driver.commitments
    .map((commitment) => ({
      commitment,
      span: spanWithin(commitment.windowStart, commitment.windowEnd, windowStart, windowEnd),
    }))
    .filter((item) => item.span !== null);

  const availableSpan = spanWithin(
    Math.max(driver.projectedTimeAvailable, windowStart),
    windowEnd,
    windowStart,
    windowEnd,
  );

  return (
    <div className="flex items-center gap-2 border-b border-border/60 py-1 last:border-b-0">
      <div className="w-40 shrink-0 truncate text-[11px]">
        {driver.firstName} {driver.lastName}
        {driver.tractorCode ? (
          <span className="ml-1 font-mono text-[9px] text-muted-foreground">
            {driver.tractorCode}
          </span>
        ) : null}
      </div>

      <div className="relative h-6 flex-1 overflow-hidden rounded bg-muted/40">
        {availableSpan ? (
          <div
            className="absolute inset-y-0 bg-green-500/10"
            style={{
              left: `${availableSpan.leftPercent}%`,
              width: `${availableSpan.widthPercent}%`,
            }}
            title="Projected available"
          />
        ) : null}

        {timeOffSpans.map(({ pto, span }, index) => (
          <div
            key={`pto-${index}`}
            className="absolute inset-y-0 bg-purple-500/25"
            style={{ left: `${span!.leftPercent}%`, width: `${span!.widthPercent}%` }}
            title={`${pto.type} time off`}
          />
        ))}

        {commitmentSpans.map(({ commitment, span }) => (
          <button
            key={commitment.moveId}
            type="button"
            onClick={() => onSelectMove(commitment.moveId)}
            className={cn(
              "absolute inset-y-0.5 flex items-center overflow-hidden rounded px-1 text-[9px]",
              "bg-brand/80 text-white hover:bg-brand",
            )}
            style={{ left: `${span!.leftPercent}%`, width: `${span!.widthPercent}%` }}
            title={`${commitment.proNumber} → ${commitment.destinationCity}, ${commitment.destinationState}`}
          >
            <span className="truncate font-mono">{commitment.proNumber}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

/**
 * The planning board — the view McLeod and Trimble users call a dispatch board. Committed
 * work, approved time off, and projected availability on one time axis is how relay and
 * multi-move planning actually gets done.
 */
export function PlanningBoard({
  drivers,
  windowStart,
  zoom,
  onZoomChange,
  onSelectMove,
}: {
  drivers: readonly DispatchBoardDriver[];
  windowStart: number;
  zoom: TimelineZoom;
  onZoomChange: (zoom: TimelineZoom) => void;
  onSelectMove: (moveId: string) => void;
}) {
  const windowEnd = windowStart + zoom * HOUR_SECONDS;
  const ticks = useMemo(
    () => hourTicks(windowStart, windowEnd, zoom),
    [windowStart, windowEnd, zoom],
  );

  return (
    <section className="flex max-h-72 min-h-0 flex-col gap-2 rounded-md border border-border bg-background p-2">
      <header className="flex items-center justify-between gap-2">
        <h2 className="text-xs font-semibold uppercase tracking-wide">Planning board</h2>
        <div className="flex items-center gap-1">
          {([4, 12, 48] as const).map((option) => (
            <button
              key={option}
              type="button"
              onClick={() => onZoomChange(option)}
              className={cn(
                "rounded px-1.5 py-0.5 text-[10px] transition-colors",
                zoom === option
                  ? "bg-brand text-white"
                  : "bg-muted text-muted-foreground hover:bg-muted/70",
              )}
            >
              {option}h
            </button>
          ))}
        </div>
      </header>

      <div className="flex items-center gap-2">
        <div className="w-40 shrink-0" />
        <div className="relative h-4 flex-1">
          {ticks.map((tick) => (
            <span
              key={tick}
              className="absolute -translate-x-1/2 text-[9px] text-muted-foreground"
              style={{ left: `${((tick - windowStart) / (windowEnd - windowStart)) * 100}%` }}
            >
              {tickLabel(tick)}
            </span>
          ))}
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="pr-2">
          {drivers.map((driver) => (
            <DriverLane
              key={driver.workerId}
              driver={driver}
              windowStart={windowStart}
              windowEnd={windowEnd}
              onSelectMove={onSelectMove}
            />
          ))}
          {drivers.length === 0 ? (
            <p className="py-6 text-center text-xs text-muted-foreground">
              No drivers to plan.
            </p>
          ) : null}
        </div>
      </ScrollArea>
    </section>
  );
}
