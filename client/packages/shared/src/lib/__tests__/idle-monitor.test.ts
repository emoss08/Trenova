import { createIdleMonitor, type IdleMonitor } from "@trenova/shared/lib/idle-monitor";
import {
  createSessionActivityClock,
  type SessionActivityClock,
} from "@trenova/shared/lib/session-activity";
import type { SessionIdlePolicy } from "@trenova/shared/types/platform";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const POLICY: SessionIdlePolicy = {
  idleTimeoutSeconds: 30 * 60,
  idleWarningSeconds: 2 * 60,
  keepAliveSeconds: 60,
};

const START = Date.UTC(2026, 9, 16, 14, 0, 0);
const MINUTE = 60_000;

function memoryStorage(): Storage {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => values.delete(key),
    setItem: (key, value) => values.set(key, value),
  };
}

type Harness = {
  monitor: IdleMonitor;
  page: EventTarget;
  windowTarget: EventTarget;
  storage: Storage;
  clock: SessionActivityClock;
  onTimeout: ReturnType<typeof vi.fn>;
  onKeepAlive: ReturnType<typeof vi.fn>;
  unsubscribe: () => void;
  changes: ReturnType<typeof vi.fn>;
};

function setup(policy: SessionIdlePolicy = POLICY, storage = memoryStorage()): Harness {
  const page = new EventTarget();
  const windowTarget = new EventTarget();
  const clock = createSessionActivityClock({ storage: () => storage, target: windowTarget });
  const onTimeout = vi.fn();
  const onKeepAlive = vi.fn();
  const monitor = createIdleMonitor({
    policy,
    clock,
    onTimeout,
    onKeepAlive,
    target: page as unknown as Document,
  });
  const changes = vi.fn();
  const unsubscribe = monitor.subscribe(changes);
  return {
    monitor,
    page,
    windowTarget,
    storage,
    clock,
    onTimeout,
    onKeepAlive,
    unsubscribe,
    changes,
  };
}

function act(page: EventTarget, type = "pointerdown") {
  page.dispatchEvent(new Event(type));
}

describe("createIdleMonitor", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(START);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts active with the deadline one idle timeout from now", () => {
    const { monitor } = setup();

    expect(monitor.getSnapshot()).toEqual({ phase: "active", deadline: START + 30 * MINUTE });
  });

  it("opens the warning at the warning lead and signs out at the deadline", () => {
    const { monitor, onTimeout, changes } = setup();

    vi.advanceTimersByTime(28 * MINUTE - 1);
    expect(monitor.getSnapshot().phase).toBe("active");

    vi.advanceTimersByTime(1);
    expect(monitor.getSnapshot()).toEqual({ phase: "warning", deadline: START + 30 * MINUTE });
    expect(onTimeout).not.toHaveBeenCalled();

    vi.advanceTimersByTime(2 * MINUTE);
    expect(monitor.getSnapshot()).toEqual({ phase: "timedOut", deadline: START + 30 * MINUTE });
    expect(onTimeout).toHaveBeenCalledTimes(1);
    expect(changes).toHaveBeenCalledTimes(3);
  });

  it("pushes the deadline back for every kind of activity", () => {
    for (const type of ["pointerdown", "pointermove", "keydown", "wheel", "touchstart", "scroll"]) {
      vi.setSystemTime(START);
      const { monitor, page, unsubscribe } = setup();

      vi.advanceTimersByTime(20 * MINUTE);
      act(page, type);
      vi.advanceTimersByTime(8 * MINUTE);

      expect(monitor.getSnapshot(), type).toEqual({
        phase: "active",
        deadline: START + 50 * MINUTE,
      });
      unsubscribe();
    }
  });

  it("does not let passing activity dismiss the warning", () => {
    const { monitor, page, onTimeout } = setup();

    vi.advanceTimersByTime(29 * MINUTE);
    act(page, "pointermove");
    act(page, "keydown");

    expect(monitor.getSnapshot().phase).toBe("warning");
    vi.advanceTimersByTime(MINUTE);
    expect(onTimeout).toHaveBeenCalledTimes(1);
  });

  it("keeps the session when the person chooses to stay", () => {
    const { monitor, onTimeout, onKeepAlive } = setup();

    vi.advanceTimersByTime(29 * MINUTE);
    monitor.stayActive();

    expect(monitor.getSnapshot()).toEqual({ phase: "active", deadline: START + 59 * MINUTE });
    expect(onKeepAlive).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(29 * MINUTE);
    expect(onTimeout).not.toHaveBeenCalled();
  });

  it("tells the server at most once per keepalive interval while the person works", () => {
    const { page, onKeepAlive } = setup();

    act(page);
    expect(onKeepAlive).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(30_000);
    act(page);
    act(page, "keydown");
    expect(onKeepAlive).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(30_000);
    act(page);
    expect(onKeepAlive).toHaveBeenCalledTimes(2);
  });

  it("does not tell the server anything while nobody is there", () => {
    const { onKeepAlive } = setup();

    vi.advanceTimersByTime(30 * MINUTE);

    expect(onKeepAlive).not.toHaveBeenCalled();
  });

  it("follows activity in another tab, closing a warning that is open here", () => {
    const { monitor, storage, windowTarget, onTimeout } = setup();

    vi.advanceTimersByTime(29 * MINUTE);
    expect(monitor.getSnapshot().phase).toBe("warning");

    storage.setItem("trenova:session-activity", String(Date.now()));
    windowTarget.dispatchEvent(
      Object.assign(new Event("storage"), { key: "trenova:session-activity" }),
    );

    expect(monitor.getSnapshot()).toEqual({ phase: "active", deadline: START + 59 * MINUTE });
    vi.advanceTimersByTime(MINUTE);
    expect(onTimeout).not.toHaveBeenCalled();
  });

  it("ignores storage events for other keys", () => {
    const { monitor, storage, windowTarget } = setup();

    vi.advanceTimersByTime(29 * MINUTE);
    storage.setItem("trenova:session-activity", String(Date.now()));
    windowTarget.dispatchEvent(Object.assign(new Event("storage"), { key: "trenova.locale" }));

    expect(monitor.getSnapshot().phase).toBe("warning");
  });

  it("starts from now, not from activity a previous session left in storage", () => {
    const storage = memoryStorage();
    storage.setItem("trenova:session-activity", String(START - 6 * 60 * MINUTE));

    const { monitor, onTimeout } = setup(POLICY, storage);

    expect(monitor.getSnapshot()).toEqual({ phase: "active", deadline: START + 30 * MINUTE });
    expect(onTimeout).not.toHaveBeenCalled();
  });

  it("signs out as soon as it is checked after sleeping through the deadline", () => {
    const { monitor, page, onTimeout } = setup();

    vi.setSystemTime(START + 45 * MINUTE);
    page.dispatchEvent(new Event("visibilitychange"));

    expect(monitor.getSnapshot().phase).toBe("timedOut");
    expect(onTimeout).toHaveBeenCalledTimes(1);
  });

  it("signs out once, and ignores the dialog after it has", () => {
    const { monitor, page, onTimeout, onKeepAlive } = setup();

    vi.advanceTimersByTime(30 * MINUTE);
    page.dispatchEvent(new Event("focus"));
    act(page);
    monitor.stayActive();
    vi.advanceTimersByTime(60 * MINUTE);

    expect(onTimeout).toHaveBeenCalledTimes(1);
    expect(onKeepAlive).not.toHaveBeenCalled();
    expect(monitor.getSnapshot().phase).toBe("timedOut");
  });

  it("stops every timer and listener once nothing subscribes", () => {
    const { page, unsubscribe, onTimeout, onKeepAlive } = setup();

    unsubscribe();
    act(page);
    vi.advanceTimersByTime(60 * MINUTE);

    expect(onTimeout).not.toHaveBeenCalled();
    expect(onKeepAlive).not.toHaveBeenCalled();
  });

  it("does nothing while the idle timeout is off", () => {
    const { monitor, page, onTimeout, onKeepAlive } = setup({
      idleTimeoutSeconds: 0,
      idleWarningSeconds: 0,
      keepAliveSeconds: 0,
    });

    act(page);
    monitor.stayActive();
    vi.advanceTimersByTime(24 * 60 * MINUTE);

    expect(monitor.getSnapshot()).toEqual({ phase: "active", deadline: 0 });
    expect(onTimeout).not.toHaveBeenCalled();
    expect(onKeepAlive).not.toHaveBeenCalled();
  });

  it("keeps working when storage refuses every read and write", () => {
    const broken = memoryStorage();
    broken.getItem = () => {
      throw new DOMException("denied", "SecurityError");
    };
    broken.setItem = () => {
      throw new DOMException("denied", "SecurityError");
    };
    const { monitor, page, onTimeout } = setup(POLICY, broken);

    vi.advanceTimersByTime(20 * MINUTE);
    act(page);
    vi.advanceTimersByTime(29 * MINUTE);

    expect(monitor.getSnapshot()).toEqual({ phase: "warning", deadline: START + 50 * MINUTE });
    vi.advanceTimersByTime(MINUTE);
    expect(onTimeout).toHaveBeenCalledTimes(1);
  });
});

describe("createSessionActivityClock", () => {
  it("shares a moment one tab records with another reading the same storage", () => {
    const storage = memoryStorage();
    const first = createSessionActivityClock({ storage: () => storage, target: null });
    const second = createSessionActivityClock({ storage: () => storage, target: null });

    first.record(START, { force: true });

    expect(second.last()).toBe(START);
  });

  it("writes to storage at most every five seconds unless forced", () => {
    const storage = memoryStorage();
    const clock = createSessionActivityClock({ storage: () => storage, target: null });

    clock.record(START, { force: true });
    clock.record(START + 4_000);
    expect(storage.getItem("trenova:session-activity")).toBe(String(START));
    expect(clock.last()).toBe(START + 4_000);

    clock.record(START + 5_000);
    expect(storage.getItem("trenova:session-activity")).toBe(String(START + 5_000));
  });

  it("never moves its own moment backwards", () => {
    const clock = createSessionActivityClock({ storage: () => null, target: null });

    clock.record(START + 10_000);
    clock.record(START, { force: true });

    expect(clock.last()).toBe(START + 10_000);
  });

  it("reads a value that is not a number as no activity", () => {
    const storage = memoryStorage();
    storage.setItem("trenova:session-activity", "yesterday");
    const clock = createSessionActivityClock({ storage: () => storage, target: null });

    expect(clock.last()).toBe(0);
  });
});
