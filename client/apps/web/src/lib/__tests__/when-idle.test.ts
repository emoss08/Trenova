import { afterEach, describe, expect, it, vi } from "vitest";
import { whenIdle } from "../when-idle";

type IdleWindow = Window & {
  requestIdleCallback?: Window["requestIdleCallback"];
  cancelIdleCallback?: Window["cancelIdleCallback"];
};

const idleWindow = window as IdleWindow;
const original = {
  request: idleWindow.requestIdleCallback,
  cancel: idleWindow.cancelIdleCallback,
};

afterEach(() => {
  idleWindow.requestIdleCallback = original.request;
  idleWindow.cancelIdleCallback = original.cancel;
  vi.useRealTimers();
});

/** Work that can wait runs when the browser is idle, and not at all once cancelled. */
describe("whenIdle", () => {
  it("waits for the browser to be idle, at most the timeout", () => {
    const request = vi.fn<Window["requestIdleCallback"]>(() => 7);
    const cancel = vi.fn<Window["cancelIdleCallback"]>();
    idleWindow.requestIdleCallback = request;
    idleWindow.cancelIdleCallback = cancel;
    const task = vi.fn();

    const stop = whenIdle(task, { timeout: 1500 });

    expect(request).toHaveBeenCalledWith(task, { timeout: 1500 });
    stop();
    expect(cancel).toHaveBeenCalledWith(7);
  });

  it("falls back to a delay where idle callbacks are missing", () => {
    vi.useFakeTimers();
    idleWindow.requestIdleCallback = undefined;
    const task = vi.fn();

    whenIdle(task, { fallbackDelay: 500 });
    vi.advanceTimersByTime(499);
    expect(task).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(task).toHaveBeenCalledTimes(1);
  });

  it("never runs a task cancelled before its delay", () => {
    vi.useFakeTimers();
    idleWindow.requestIdleCallback = undefined;
    const task = vi.fn();

    const stop = whenIdle(task, { fallbackDelay: 500 });
    stop();
    vi.advanceTimersByTime(1000);
    expect(task).not.toHaveBeenCalled();
  });
});
