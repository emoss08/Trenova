import type { SessionActivityClock } from "@trenova/shared/lib/session-activity";
import type { SessionIdlePolicy } from "@trenova/shared/types/platform";

/**
 * Where a signed-in tab stands against the idle timeout. `deadline` is the epoch
 * millisecond the session will be signed out; it is 0 while the timeout is off.
 */
export type IdlePhase = "active" | "warning" | "timedOut";

export type IdleSnapshot = Readonly<{ phase: IdlePhase; deadline: number }>;

export type IdleMonitor = {
  subscribe: (listener: () => void) => () => void;
  getSnapshot: () => IdleSnapshot;
  stayActive: () => void;
};

type IdleTarget = Pick<Document, "addEventListener" | "removeEventListener">;

export type IdleMonitorOptions = {
  policy: SessionIdlePolicy;
  clock: SessionActivityClock;
  onTimeout: () => void;
  onKeepAlive: () => void;
  target?: IdleTarget;
  now?: () => number;
};

/**
 * What counts as the person being there. Moving the pointer or scrolling a page they are
 * reading counts as much as typing; a request the app makes on its own does not.
 */
export const IDLE_ACTIVITY_EVENTS = [
  "pointerdown",
  "pointermove",
  "keydown",
  "wheel",
  "touchstart",
  "scroll",
] as const;

const RECHECK_EVENTS = ["visibilitychange", "focus"] as const;

const LISTENER_OPTIONS: AddEventListenerOptions = { capture: true, passive: true };

const OFF: IdleSnapshot = Object.freeze({ phase: "active", deadline: 0 });

/**
 * Watches for the person going quiet and signs them out when they have been away for the
 * policy's idle timeout, opening a warning `idleWarningSeconds` before then. Once the
 * warning is up only `stayActive` (the dialog's button) keeps the session: a pointer
 * drifting across the screen is not someone choosing to stay. While they work it tells
 * the server at most once per `keepAliveSeconds`, so the server's own idle timeout, which
 * counts requests rather than keystrokes, never ends a session someone is using.
 *
 * It runs only while subscribed, and reads the deadline from the wall clock on every
 * check, so a laptop that slept through the deadline signs out the moment it wakes.
 */
export function createIdleMonitor({
  policy,
  clock,
  onTimeout,
  onKeepAlive,
  target = typeof document === "undefined" ? undefined : document,
  now = Date.now,
}: IdleMonitorOptions): IdleMonitor {
  const enabled = policy.idleTimeoutSeconds > 0 && target !== undefined;
  const timeoutMs = policy.idleTimeoutSeconds * 1000;
  const warningMs = policy.idleWarningSeconds * 1000;
  const keepAliveMs = policy.keepAliveSeconds * 1000;

  const listeners = new Set<() => void>();
  let snapshot: IdleSnapshot = OFF;
  let timer: ReturnType<typeof setTimeout> | null = null;
  let lastKeepAliveAt = 0;
  let stopWatching: (() => void) | null = null;

  const publish = (next: IdleSnapshot) => {
    if (next.phase === snapshot.phase && next.deadline === snapshot.deadline) {
      return;
    }
    snapshot = next;
    for (const listener of listeners) {
      listener();
    }
  };

  const clearTimer = () => {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  };

  const keepAlive = (at: number, force: boolean) => {
    if (force || at - lastKeepAliveAt >= keepAliveMs) {
      lastKeepAliveAt = at;
      onKeepAlive();
    }
  };

  const evaluate = () => {
    clearTimer();
    if (snapshot.phase === "timedOut") {
      return;
    }

    const at = now();
    const deadline = clock.last() + timeoutMs;
    if (at >= deadline) {
      stopWatching?.();
      stopWatching = null;
      publish({ phase: "timedOut", deadline });
      onTimeout();
      return;
    }

    const warnAt = deadline - warningMs;
    const warning = at >= warnAt;
    publish({ phase: warning ? "warning" : "active", deadline });
    timer = setTimeout(evaluate, (warning ? deadline : warnAt) - at);
  };

  const onActivity = () => {
    if (snapshot.phase !== "active") {
      return;
    }
    const at = now();
    clock.record(at);
    keepAlive(at, false);
  };

  const start = () => {
    if (!enabled || !target) {
      return;
    }

    clock.record(now(), { force: true });

    for (const type of IDLE_ACTIVITY_EVENTS) {
      target.addEventListener(type, onActivity, LISTENER_OPTIONS);
    }
    for (const type of RECHECK_EVENTS) {
      target.addEventListener(type, evaluate, LISTENER_OPTIONS);
    }
    const stopClock = clock.subscribe(evaluate);

    stopWatching = () => {
      for (const type of IDLE_ACTIVITY_EVENTS) {
        target.removeEventListener(type, onActivity, LISTENER_OPTIONS);
      }
      for (const type of RECHECK_EVENTS) {
        target.removeEventListener(type, evaluate, LISTENER_OPTIONS);
      }
      stopClock();
    };

    evaluate();
  };

  const stop = () => {
    clearTimer();
    stopWatching?.();
    stopWatching = null;
  };

  return {
    subscribe: (listener) => {
      listeners.add(listener);
      if (listeners.size === 1) {
        start();
      }
      return () => {
        listeners.delete(listener);
        if (listeners.size === 0) {
          stop();
        }
      };
    },
    getSnapshot: () => snapshot,
    stayActive: () => {
      if (!enabled || snapshot.phase === "timedOut") {
        return;
      }
      const at = now();
      clock.record(at, { force: true });
      keepAlive(at, true);
      evaluate();
    },
  };
}
