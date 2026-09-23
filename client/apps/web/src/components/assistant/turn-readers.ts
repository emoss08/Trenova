/**
 * Every open reader of a reply, so signing out can let go of all of them at
 * once.
 *
 * A reader reattaches when its stream drops, because a dropped connection is
 * not a lost reply. Signing out stops the person's replies on the server and
 * ends the session under them, and a reader left running would read that as
 * a dropped connection and knock on a closed door until its allowance ran
 * out — then report a failure, and refetch the thread, against a session that
 * no longer exists. Releasing the readers first is what makes sign-out quiet.
 *
 * Releasing only lets go of the reader. It never stops a turn: that is the
 * server's to do at sign-out, and a person closing a palette or leaving a
 * page has not asked for their answer to be thrown away.
 */
const readers = new Set<AbortController>();

/** Tracks a reader until it is released or lets go of its own accord. */
export function registerTurnReader(controller: AbortController): () => void {
  if (controller.signal.aborted) {
    return () => undefined;
  }
  readers.add(controller);
  const forget = () => {
    readers.delete(controller);
  };
  controller.signal.addEventListener("abort", forget, { once: true });

  return () => {
    controller.signal.removeEventListener("abort", forget);
    forget();
  };
}

/** Lets go of every open reader. Returns how many there were. */
export function releaseTurnReaders(): number {
  const open = [...readers];
  readers.clear();
  for (const controller of open) {
    controller.abort();
  }

  return open.length;
}

/** How many readers are open, for tests and diagnostics. */
export function openTurnReaderCount(): number {
  return readers.size;
}
