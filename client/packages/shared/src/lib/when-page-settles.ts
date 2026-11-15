const IDLE_TIMEOUT_MS = 10_000;
const NO_IDLE_CALLBACK_DELAY_MS = 2_000;

/** Runs the task once the page's load event has fired, at once when it already has. */
export function afterPageLoad(task: () => void): void {
  if (document.readyState === "complete") task();
  else window.addEventListener("load", task, { once: true });
}

/**
 * Runs the task when the browser is idle, and at the latest after `timeoutMs`; after a short
 * delay where the browser cannot say when it is idle.
 */
export function whenBrowserIdle(task: () => void, timeoutMs: number = IDLE_TIMEOUT_MS): void {
  if (typeof window.requestIdleCallback === "function") {
    window.requestIdleCallback(task, { timeout: timeoutMs });
  } else {
    window.setTimeout(task, Math.min(timeoutMs, NO_IDLE_CALLBACK_DELAY_MS));
  }
}

/** Runs work that must never compete with first paint: after load, then at idle time. */
export function whenPageSettles(task: () => void): void {
  afterPageLoad(() => whenBrowserIdle(task));
}
