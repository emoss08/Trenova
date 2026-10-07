import { useCallback, useInsertionEffect, useRef } from "react";

/**
 * A function whose identity never changes but which always runs the latest
 * `callback`. Hand it to a memoized child in place of a handler that closes
 * over fast-changing state, so the child is not redrawn every time that state
 * moves. It is for event handlers: calling it during render reads the
 * callback from the previous commit.
 */
export function useLatestCallback<TArgs extends unknown[], TResult>(
  callback: (...args: TArgs) => TResult,
): (...args: TArgs) => TResult {
  const ref = useRef(callback);
  useInsertionEffect(() => {
    ref.current = callback;
  });
  return useCallback((...args: TArgs) => ref.current(...args), []);
}
