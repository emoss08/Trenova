import { useEffect, useState } from "react";

/** The longest a browser timer can be set for; a longer snooze is waited out in steps. */
const MAX_TIMER_MS = 2_147_483_647;

function nowSeconds(): number {
  return Math.floor(Date.now() / 1000);
}

/**
 * Now, in seconds, moved forward when a snooze ends so the case comes back
 * on screen at its time without polling. `onWake` runs then too, to read the
 * case's new state from the server.
 */
export function useSnoozeClock(wakeAt: number | null | undefined, onWake?: () => void): number {
  const [now, setNow] = useState(nowSeconds);

  useEffect(() => {
    if (!wakeAt) {
      return;
    }
    const delay = (wakeAt - nowSeconds()) * 1000;
    if (delay <= 0) {
      return;
    }
    const timer = window.setTimeout(
      () => {
        const at = nowSeconds();
        setNow(at);
        if (at >= wakeAt) {
          onWake?.();
        }
      },
      Math.min(delay + 250, MAX_TIMER_MS),
    );

    return () => window.clearTimeout(timer);
  }, [onWake, wakeAt]);

  return now;
}

/** The soonest of several snoozes still to end, or null when none is. */
export function nextWake(snoozes: Iterable<number | null | undefined>, now: number): number | null {
  let soonest: number | null = null;
  for (const until of snoozes) {
    if (until && until > now && (soonest === null || until < soonest)) {
      soonest = until;
    }
  }

  return soonest;
}
