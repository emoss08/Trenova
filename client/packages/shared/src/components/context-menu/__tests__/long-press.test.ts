import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LONG_PRESS_MS, LONG_PRESS_TOLERANCE, createLongPress } from "../long-press";

const touch = (x = 100, y = 100) => ({ pointerType: "touch", clientX: x, clientY: y, button: 0 });

describe("long press", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("opens where the finger came down once it has been held still long enough", () => {
    const onLongPress = vi.fn();
    const press = createLongPress({ onLongPress });

    press.down(touch(40, 60));
    vi.advanceTimersByTime(LONG_PRESS_MS - 1);
    expect(onLongPress).not.toHaveBeenCalled();
    press.move(touch(43, 62));
    vi.advanceTimersByTime(1);

    expect(onLongPress).toHaveBeenCalledWith({ x: 40, y: 60 });
  });

  it("is a tap, not a press, when the finger lifts early", () => {
    const onLongPress = vi.fn();
    const press = createLongPress({ onLongPress });

    press.down(touch());
    vi.advanceTimersByTime(LONG_PRESS_MS / 2);
    press.up();
    vi.advanceTimersByTime(LONG_PRESS_MS);

    expect(onLongPress).not.toHaveBeenCalled();
    expect(press.consumeClick()).toBe(false);
  });

  it("gives way to a scroll or a swipe that moves past the tolerance", () => {
    const onLongPress = vi.fn();
    const press = createLongPress({ onLongPress });

    press.down(touch(100, 100));
    press.move(touch(100, 100 + LONG_PRESS_TOLERANCE + 1));
    vi.advanceTimersByTime(LONG_PRESS_MS * 2);

    expect(onLongPress).not.toHaveBeenCalled();
  });

  it("leaves a mouse to its right button", () => {
    const onLongPress = vi.fn();
    const press = createLongPress({ onLongPress });

    press.down({ pointerType: "mouse", clientX: 0, clientY: 0, button: 0 });
    vi.advanceTimersByTime(LONG_PRESS_MS * 2);

    expect(onLongPress).not.toHaveBeenCalled();
  });

  it("swallows the one click the lifted finger sends, and only that one", () => {
    const press = createLongPress({ onLongPress: vi.fn() });

    press.down(touch());
    vi.advanceTimersByTime(LONG_PRESS_MS);
    press.up();

    expect(press.justFired()).toBe(true);
    expect(press.consumeClick()).toBe(true);
    expect(press.consumeClick()).toBe(false);
  });

  it("starts over when a second finger lands before the first fires", () => {
    const onLongPress = vi.fn();
    const press = createLongPress({ onLongPress });

    press.down(touch(10, 10));
    vi.advanceTimersByTime(LONG_PRESS_MS - 100);
    press.down(touch(200, 200));
    vi.advanceTimersByTime(100);
    expect(onLongPress).not.toHaveBeenCalled();
    vi.advanceTimersByTime(LONG_PRESS_MS);

    expect(onLongPress).toHaveBeenCalledTimes(1);
    expect(onLongPress).toHaveBeenCalledWith({ x: 200, y: 200 });
  });
});
