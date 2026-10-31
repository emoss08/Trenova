/**
 * Whether the page can be reloaded without losing work. Anything that holds work a full page
 * load would throw away — a form with unsaved edits, popout windows the page closes when it
 * unloads — adds a guard while it does; code that reloads on its own initiative (moving onto
 * a new release) asks first and waits when the answer is no. A reload the person asks for
 * (refresh, closing the tab) is not this registry's business: beforeunload handles those.
 */

type ReloadGuard = () => boolean;

const guards = new Set<ReloadGuard>();

/**
 * Registers a guard; while it returns true the page holds work a reload would lose. Returns
 * the function that removes it.
 */
export function addReloadGuard(isHoldingWork: ReloadGuard): () => void {
  guards.add(isHoldingWork);
  return () => {
    guards.delete(isHoldingWork);
  };
}

/** True when no guard holds work. */
export function reloadIsSafe(): boolean {
  for (const isHoldingWork of guards) {
    if (isHoldingWork()) return false;
  }
  return true;
}

let leavingRegardless = false;

/**
 * Loads `destination` without stopping for the work the page holds: the session that work
 * belonged to has ended or someone else holds it now, so nothing on the page can be saved,
 * and on a shared computer the next person must not be asked whether to keep the last one's
 * screen. The browser's leave-page prompt stands aside for this one page load. A reload the
 * person chooses is not this: it goes through the prompt like any other.
 */
export function leaveRegardlessOfWork(destination: string): void {
  leavingRegardless = true;
  window.location.assign(destination);
}

/** Whether the page is being left on purpose, past any unsaved work. */
export function isLeavingRegardlessOfWork(): boolean {
  return leavingRegardless;
}
