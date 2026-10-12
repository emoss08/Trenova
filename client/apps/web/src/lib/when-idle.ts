/**
 * Runs `task` once the browser has nothing better to do, or after `timeout`
 * at the latest. Where idle callbacks are missing (Safari) it runs after
 * `fallbackDelay` instead. Returns what cancels it, for an effect's cleanup.
 */
export function whenIdle(
  task: () => void,
  { timeout = 4000, fallbackDelay = 2000 }: { timeout?: number; fallbackDelay?: number } = {},
): () => void {
  if (typeof window.requestIdleCallback === "function") {
    const handle = window.requestIdleCallback(task, { timeout });
    return () => window.cancelIdleCallback(handle);
  }
  const handle = window.setTimeout(task, fallbackDelay);
  return () => window.clearTimeout(handle);
}
