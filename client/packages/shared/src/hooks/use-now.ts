import { useSyncExternalStore } from "react";

/** How often a reader needs the time to move: once a second or once a minute. */
export type ClockGranularity = "second" | "minute";

type Clock = {
  stepMs: number;
  now: number;
  listeners: Set<() => void>;
  timer: number | null;
};

const clocks: Record<ClockGranularity, Clock> = {
  second: { stepMs: 1000, now: 0, listeners: new Set(), timer: null },
  minute: { stepMs: 60_000, now: 0, listeners: new Set(), timer: null },
};

function read(clock: Clock): number {
  return Math.floor(Date.now() / clock.stepMs) * (clock.stepMs / 1000);
}

function tick(clock: Clock) {
  const next = read(clock);
  if (next !== clock.now) {
    clock.now = next;
    for (const listener of clock.listeners) listener();
  }
}

function start(clock: Clock) {
  clock.now = read(clock);
  const untilBoundary = clock.stepMs - (Date.now() % clock.stepMs);
  clock.timer = window.setTimeout(() => {
    tick(clock);
    clock.timer = window.setInterval(() => tick(clock), clock.stepMs);
  }, untilBoundary);
}

function stop(clock: Clock) {
  if (clock.timer !== null) {
    window.clearTimeout(clock.timer);
    window.clearInterval(clock.timer);
    clock.timer = null;
  }
}

const subscribers: Record<ClockGranularity, (listener: () => void) => () => void> = {
  second: (listener) => subscribe(clocks.second, listener),
  minute: (listener) => subscribe(clocks.minute, listener),
};

const snapshots: Record<ClockGranularity, () => number> = {
  second: () => snapshot(clocks.second),
  minute: () => snapshot(clocks.minute),
};

function subscribe(clock: Clock, listener: () => void): () => void {
  clock.listeners.add(listener);
  if (clock.listeners.size === 1) start(clock);
  return () => {
    clock.listeners.delete(listener);
    if (clock.listeners.size === 0) stop(clock);
  };
}

function snapshot(clock: Clock): number {
  if (clock.listeners.size === 0) clock.now = read(clock);
  return clock.now;
}

/**
 * The current time in Unix seconds, moving on the second or minute boundary. Every
 * reader of one granularity shares one timer, which runs only while something
 * reads it, so a page of countdowns costs one timer and redraws only the readers.
 */
export function useNow(granularity: ClockGranularity = "minute"): number {
  return useSyncExternalStore(subscribers[granularity], snapshots[granularity]);
}
