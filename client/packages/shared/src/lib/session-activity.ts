const STORAGE_KEY = "trenova:session-activity";
const WRITE_INTERVAL_MS = 5_000;

/**
 * When the person last did something in any tab of the app, in epoch milliseconds. Every
 * tab reads the same moment, so a tab left in the background is not signed out from under
 * someone working in another one. The moment is shared through local storage, which other
 * tabs hear about through the `storage` event; the tab that acts keeps its own copy, so it
 * stays exact when storage is unavailable and between the throttled writes.
 */
export type SessionActivityClock = {
  last: () => number;
  record: (at: number, options?: { force?: boolean }) => void;
  subscribe: (listener: () => void) => () => void;
};

type ClockEnvironment = {
  storage: () => Storage | null;
  target: Pick<Window, "addEventListener" | "removeEventListener"> | null;
};

export function createSessionActivityClock({
  storage,
  target,
}: ClockEnvironment): SessionActivityClock {
  let local = 0;
  let written = 0;

  const read = (): number => {
    try {
      const raw = storage()?.getItem(STORAGE_KEY);
      const value = raw == null ? Number.NaN : Number(raw);
      return Number.isFinite(value) ? value : 0;
    } catch {
      return 0;
    }
  };

  const write = (at: number) => {
    try {
      storage()?.setItem(STORAGE_KEY, String(at));
      written = at;
    } catch {
      // Storage refused (private mode, quota): this tab still keeps its own copy.
    }
  };

  return {
    last: () => Math.max(local, read()),
    record: (at, { force = false } = {}) => {
      local = Math.max(local, at);
      if (force || at - written >= WRITE_INTERVAL_MS) {
        write(at);
      }
    },
    subscribe: (listener) => {
      if (!target) {
        return () => {};
      }
      const onStorage = (event: StorageEvent) => {
        if (event.key === STORAGE_KEY) {
          listener();
        }
      };
      target.addEventListener("storage", onStorage);
      return () => target.removeEventListener("storage", onStorage);
    },
  };
}

const browserStorage = (): Storage | null =>
  typeof window === "undefined" ? null : window.localStorage;

export const sessionActivityClock = createSessionActivityClock({
  storage: browserStorage,
  target: typeof window === "undefined" ? null : window,
});
