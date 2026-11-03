/**
 * Results a screen is showing as they happen.
 *
 * A notice that announces a result (a finished billing transfer) is news for a person who is
 * somewhere else. While a screen watches that result, its notice waits; once the screen has
 * shown the result, the notice is dropped, since the person has just seen it; if the screen
 * closes before showing it, the notice is shown after all. The notice still reaches the bell
 * either way: this only decides whether it also interrupts.
 *
 * Kept to plain module state with no imports, because the realtime connection in the shell
 * reads it and must not pull in the screens that write it.
 */

type Watch = {
  watchers: number;
  seen: boolean;
  deferred: (() => void) | null;
};

const watches = new Map<string, Watch>();

/** Marks a result as on screen until the returned release is called. */
export function watchResult(key: string): () => void {
  const watch = watches.get(key) ?? { watchers: 0, seen: false, deferred: null };
  watch.watchers += 1;
  watches.set(key, watch);

  let released = false;
  return () => {
    if (released) return;
    released = true;
    watch.watchers -= 1;
    if (watch.watchers > 0) return;

    watches.delete(key);
    const deferred = watch.seen ? null : watch.deferred;
    watch.deferred = null;
    deferred?.();
  };
}

/** Records that the watching screen has shown the result, so its notice need not interrupt. */
export function markResultSeen(key: string) {
  const watch = watches.get(key);
  if (!watch) return;
  watch.seen = true;
  watch.deferred = null;
}

/**
 * Announces a result now when nothing is showing it, holds the announcement while a screen
 * is, and drops it when that screen has already shown the result.
 */
export function announceUnlessWatched(key: string, announce: () => void) {
  const watch = watches.get(key);
  if (!watch) {
    announce();
    return;
  }
  if (watch.seen) return;
  watch.deferred = announce;
}

export function billingTransferResultKey(runId: string): string {
  return `billing-transfer-run:${runId}`;
}
