import { calendarDayStart } from "@/lib/calendar-days";
import { useEffect, useState } from "react";

/** A little past midnight, so the new day has begun on every clock that reads it. */
const PAST_MIDNIGHT_MS = 1000;

type CalendarDay = { timezone: string; start: number };

function nowSeconds(): number {
  return Math.floor(Date.now() / 1000);
}

function dayIn(timezone: string): CalendarDay {
  return { timezone, start: calendarDayStart(nowSeconds(), 0, timezone) };
}

/**
 * When the reader's calendar day began, in Unix seconds, moving on at their
 * midnight. A tab left asleep over midnight reads the day again as it is
 * shown, since a timer does not run while the machine sleeps. Readers render
 * again only when the day changes.
 */
export function useCalendarDay(timezone: string): number {
  const [day, setDay] = useState(() => dayIn(timezone));
  let current = day;
  if (day.timezone !== timezone) {
    current = dayIn(timezone);
    setDay(current);
  }
  const { start } = current;

  useEffect(() => {
    const check = () =>
      setDay((previous) => {
        const next = dayIn(timezone);
        return next.timezone === previous.timezone && next.start === previous.start
          ? previous
          : next;
      });
    const tomorrow = calendarDayStart(start, -1, timezone);
    const timer = window.setTimeout(
      check,
      Math.max(0, (tomorrow - nowSeconds()) * 1000) + PAST_MIDNIGHT_MS,
    );
    const onShown = () => {
      if (document.visibilityState === "visible") {
        check();
      }
    };
    document.addEventListener("visibilitychange", onShown);

    return () => {
      window.clearTimeout(timer);
      document.removeEventListener("visibilitychange", onShown);
    };
  }, [start, timezone]);

  return start;
}
