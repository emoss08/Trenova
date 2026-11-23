import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatHoursMinutes } from "@/lib/hours-format";
import { shortWeekday, weekBars } from "./schedule-model";

export type WeekPatternProps = {
  daysOfWeek: string;
  startMinute: number;
  durationMinutes: number;
  /** Days (Sunday-based) the worker said they can't work, drawn in warning where the shift lands. */
  clashes?: readonly number[];
  className?: string;
};

/** One week of a shift pattern, Monday first: each day's hours as a bar across the day. */
export function WeekPattern({
  daysOfWeek,
  startMinute,
  durationMinutes,
  clashes = [],
  className,
}: WeekPatternProps) {
  const t = useT();
  const week = weekBars(daysOfWeek, startMinute, durationMinutes);
  return (
    <div className={cn("grid grid-cols-7 gap-1", className)}>
      {week.map((entry, index) => {
        const clash = clashes.includes(entry.day);
        return (
          <div
            key={entry.day}
            className="flex min-w-0 animate-rise flex-col gap-1 motion-reduce:animate-none"
            style={{ animationDelay: `${index * 30}ms` }}
          >
            <span className="text-2xs text-muted-foreground">{shortWeekday(entry.day)}</span>
            <span className="relative h-6 overflow-hidden rounded-xs border border-border-subtle bg-card">
              {entry.bars.map((bar) => (
                <span
                  key={bar.startPercent}
                  className={cn("absolute inset-y-0", clash ? "bg-warning" : "bg-ink")}
                  style={{ left: `${bar.startPercent}%`, width: `${bar.widthPercent}%` }}
                />
              ))}
            </span>
            <span className="truncate text-2xs text-muted-foreground tabular-nums">
              {entry.minutes > 0 ? formatHoursMinutes(t, entry.minutes * 60_000) : t("Off")}
            </span>
          </div>
        );
      })}
    </div>
  );
}
