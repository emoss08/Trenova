import type { AIUsageDay } from "@/lib/graphql/ai-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatISODateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

/** The shortest a bar draws, so a quiet day still reads as a day. */
const MIN_BAR_PX = 3;
const MAX_BAR_PX = 20;
/** The least of a bar the failed part takes, so one failure is visible. */
const MIN_FAILED_SHARE = 0.2;

/**
 * A week of model calls as small bars, oldest first and today last, each topped with the
 * share that failed.
 */
export function DailyCallBars({ days }: { days: readonly AIUsageDay[] }) {
  const t = useT();
  const busiest = Math.max(1, ...days.map((day) => day.calls));

  return (
    <span className="inline-flex h-5 items-end gap-0.5" aria-hidden>
      {days.map((day, index) => {
        const height = MIN_BAR_PX + (day.calls / busiest) * (MAX_BAR_PX - MIN_BAR_PX);
        const failedShare =
          day.failed > 0 ? Math.max(MIN_FAILED_SHARE, day.failed / Math.max(day.calls, 1)) : 0;
        return (
          <span
            key={day.day}
            title={t(
              "{0}: {1} calls, {2} failed",
              formatISODateMedium(day.day),
              day.calls.toLocaleString(),
              day.failed.toLocaleString(),
            )}
            className={cn(
              "relative flex w-1.5 flex-col overflow-hidden rounded-sm",
              index === days.length - 1 ? "bg-foreground/60" : "bg-foreground/25",
            )}
            style={{ height }}
          >
            {failedShare > 0 && (
              <span className="w-full bg-danger" style={{ height: `${failedShare * 100}%` }} />
            )}
          </span>
        );
      })}
    </span>
  );
}
