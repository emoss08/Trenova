import { calendarDayOf } from "@/lib/calendar-days";
import { useEffect, useState } from "react";

const DAY_CHECK_MS = 60_000;

/**
 * The current instant, in seconds, re-read only when the reader's calendar
 * day changes.
 *
 * Day markers are relative to today. A thread left open over midnight kept
 * labelling yesterday's messages "Today", because the clock was read once
 * when the thread mounted. A clock that ticked every minute would fix that by
 * re-rendering every row sixty times an hour for nothing; this one checks
 * every minute and changes only when the date does.
 */
export function useCalendarNow(timezone: string): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  useEffect(() => {
    const id = setInterval(() => {
      const current = Math.floor(Date.now() / 1000);
      setNow((previous) =>
        calendarDayOf(previous, timezone) === calendarDayOf(current, timezone) ? previous : current,
      );
    }, DAY_CHECK_MS);
    return () => clearInterval(id);
  }, [timezone]);

  return now;
}
