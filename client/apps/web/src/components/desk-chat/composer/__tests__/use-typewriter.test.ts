import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useTypewriter } from "../use-typewriter";

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

/**
 * Lets `ms` pass in 10ms slices, each in its own act: the typewriter schedules its
 * next keystroke from an effect, which runs only once an act has flushed, and its
 * shortest delay (18ms) is longer than a slice.
 */
function runFor(ms: number) {
  for (let passed = 0; passed < ms; passed += 10) {
    act(() => {
      vi.advanceTimersByTime(10);
    });
  }
}

describe("useTypewriter", () => {
  it("types every line once, then rests on the last one with nothing scheduled", () => {
    const lines = ["Rate a load", "Find a truck"];
    const { result } = renderHook(() => useTypewriter(lines, true));

    // One pass takes under four seconds: the first line typed, held and erased, the
    // second typed. Ten seconds is long enough for a looping typewriter to start over.
    runFor(10_000);

    expect(result.current.index).toBe(1);
    expect(result.current.text).toBe("Find a truck");
    expect(result.current.done).toBe(true);
    expect(result.current.resting).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("is not resting while an earlier line is still on screen", () => {
    const lines = ["Rate a load", "Find a truck"];
    const { result } = renderHook(() => useTypewriter(lines, true));

    // "Rate a load" is typed out and held, not yet erased.
    runFor(38 * "Rate a load".length + 100);

    expect(result.current.index).toBe(0);
    expect(result.current.text).toBe("Rate a load");
    expect(result.current.done).toBe(true);
    expect(result.current.resting).toBe(false);
  });

  it("rests on a single line once it is typed", () => {
    const { result } = renderHook(() => useTypewriter(["Ask anything"], true));

    runFor(10_000);

    expect(result.current.text).toBe("Ask anything");
    expect(result.current.resting).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("types nothing while inactive", () => {
    const { result } = renderHook(() => useTypewriter(["Rate a load"], false));

    runFor(10_000);

    expect(result.current.text).toBe("");
    expect(vi.getTimerCount()).toBe(0);
  });
});
