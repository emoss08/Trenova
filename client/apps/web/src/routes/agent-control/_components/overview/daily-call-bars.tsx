import type { AIUsageDay } from "@/lib/graphql/ai-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatISODateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

/** The shortest a bar draws, so a quiet day still reads as a day. */
const MIN_BAR_PX = 3;
/** What the busiest day adds to the shortest bar. */
const RANGE_PX = 17;
/** The least of a bar the failed part takes, in percent, so one failure is visible. */
const MIN_FAILED_PERCENT = 20;

/**
 * A week of model calls as small bars, oldest first and today last, each topped with the
 * share that failed.
 */
export function DailyCallBars({ days }: { days: readonly AIUsageDay[] }) {
  const t = useT();
  const busiest = Math.max(1, ...days.map((day) => day.calls));

  return (
    <span className="fbars">
      {days.map((day, index) => (
        <i
          key={day.day}
          title={t(
            "{0}: {1} calls, {2} failed",
            formatISODateMedium(day.day),
            day.calls.toLocaleString(),
            day.failed.toLocaleString(),
          )}
          className={cn(index === days.length - 1 && "now")}
          style={{ height: MIN_BAR_PX + (day.calls / busiest) * RANGE_PX }}
        >
          {day.failed > 0 && (
            <u
              style={{
                height: `${Math.max(MIN_FAILED_PERCENT, (day.failed / Math.max(day.calls, 1)) * 100)}%`,
              }}
            />
          )}
        </i>
      ))}
    </span>
  );
}
