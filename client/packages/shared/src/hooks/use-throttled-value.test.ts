import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useThrottledValue } from "./use-throttled-value";

describe("useThrottledValue", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function render() {
    return renderHook(({ value }) => useThrottledValue(value, 120), {
      initialProps: { value: 0 },
    });
  }

  it("answers the first change at once", () => {
    const { result, rerender } = render();
    rerender({ value: 1 });
    act(() => {
      vi.advanceTimersByTime(0);
    });
    expect(result.current).toBe(1);
  });

  it("keeps answering while the value moves, at most once an interval, without waiting for it to stop", () => {
    const { result, rerender } = render();
    rerender({ value: 1 });
    act(() => {
      vi.advanceTimersByTime(0);
    });

    for (const value of [2, 3, 4]) {
      act(() => {
        vi.advanceTimersByTime(30);
      });
      rerender({ value });
    }
    expect(result.current).toBe(1);

    act(() => {
      vi.advanceTimersByTime(30);
    });
    expect(result.current).toBe(4);

    rerender({ value: 5 });
    act(() => {
      vi.advanceTimersByTime(60);
    });
    expect(result.current).toBe(4);
    act(() => {
      vi.advanceTimersByTime(60);
    });
    expect(result.current).toBe(5);
  });
});
