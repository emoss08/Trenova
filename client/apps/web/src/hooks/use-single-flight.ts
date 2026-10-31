import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useCallback, useRef, useState } from "react";

/**
 * Runs an async action at most once at a time. A call made while one is still running is ignored
 * (a double click, a key held down, a menu reopened before the first request lands), and
 * `pending` stays true until its promise settles. The lock is taken synchronously, so it holds
 * before the next render shows anything as busy. The action reports its own failure (a mutation's
 * `onError`), so a rejection here only releases the lock.
 */
export function useSingleFlight<Args extends unknown[]>(
  action: (...args: Args) => Promise<unknown>,
): { run: (...args: Args) => Promise<void>; pending: boolean } {
  const inFlight = useRef(false);
  const [pending, setPending] = useState(false);
  const latest = useLatestCallback(action);

  const run = useCallback(
    async (...args: Args) => {
      if (inFlight.current) return;
      inFlight.current = true;
      setPending(true);
      try {
        await latest(...args);
      } catch {
        // The action has already told the person what went wrong.
      } finally {
        inFlight.current = false;
        setPending(false);
      }
    },
    [latest],
  );

  return { run, pending };
}
