import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useSettledValue } from "./use-settled-value";

describe("useSettledValue", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("follows a change at once when the value had been still", () => {
    const { result, rerender } = renderHook(({ value }) => useSettledValue(value, 200), {
      initialProps: { value: "a" },
    });
    expect(result.current).toBe("a");

    act(() => {
      vi.advanceTimersByTime(500);
    });
    rerender({ value: "b" });
    act(() => {
      vi.advanceTimersByTime(0);
    });
    expect(result.current).toBe("b");
  });

  it("passes over values that follow each other quickly and settles on the last", () => {
    const { result, rerender } = renderHook(({ value }) => useSettledValue(value, 200), {
      initialProps: { value: "a" },
    });
    act(() => {
      vi.advanceTimersByTime(500);
    });

    rerender({ value: "b" });
    act(() => {
      vi.advanceTimersByTime(50);
    });
    rerender({ value: "c" });
    act(() => {
      vi.advanceTimersByTime(50);
    });
    rerender({ value: "d" });
    act(() => {
      vi.advanceTimersByTime(199);
    });
    expect(result.current).toBe("b");

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(result.current).toBe("d");
  });
});
