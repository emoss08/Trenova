import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCountUp } from "./use-count-up";

function mockReducedMotion(reduced: boolean) {
  vi.stubGlobal(
    "matchMedia",
    vi.fn((query: string) => ({
      matches: reduced && query.includes("reduce"),
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  );
}

describe("useCountUp", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["requestAnimationFrame", "cancelAnimationFrame", "performance"] });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("starts at the value it is first given", () => {
    mockReducedMotion(false);
    const { result } = renderHook(() => useCountUp(42));

    expect(result.current).toBe(42);
  });

  it("eases to a new value and lands on it exactly", () => {
    mockReducedMotion(false);
    const { result, rerender } = renderHook(({ value }) => useCountUp(value, 900), {
      initialProps: { value: 0 },
    });

    rerender({ value: 100 });
    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(result.current).toBeGreaterThan(0);
    expect(result.current).toBeLessThan(100);

    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current).toBe(100);
  });

  it("jumps straight to the new value when the person asked for less motion", () => {
    mockReducedMotion(true);
    const { result, rerender } = renderHook(({ value }) => useCountUp(value), {
      initialProps: { value: 0 },
    });

    rerender({ value: 100 });

    expect(result.current).toBe(100);
  });
});
