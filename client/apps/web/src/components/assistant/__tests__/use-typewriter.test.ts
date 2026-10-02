import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useTypewriter } from "@trenova/shared/hooks/use-typewriter";

const OPTIONS = { typeMs: 10, holdMs: 100, eraseMs: 5, gapMs: 20 };

/**
 * Each phase arms the next timeout from inside a render, so the clock is
 * moved a few milliseconds at a time with a flush between, as a browser
 * would, rather than in one jump that would leave later timeouts unarmed.
 */
function pass(ms: number) {
  for (let elapsed = 0; elapsed < ms; elapsed += 5) {
    act(() => {
      vi.advanceTimersByTime(5);
    });
  }
}

describe("useTypewriter", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("writes a line a character at a time, holds it, takes it back and writes the next", () => {
    const { result } = renderHook(() => useTypewriter(["ab", "cd"], OPTIONS));
    expect(result.current.text).toBe("");
    expect(result.current.complete).toBe(false);

    pass(OPTIONS.gapMs);
    expect(result.current.text).toBe("a");
    pass(OPTIONS.typeMs);
    expect(result.current.text).toBe("ab");
    expect(result.current.phase).toBe("holding");
    expect(result.current.complete).toBe(true);

    pass(OPTIONS.holdMs);
    expect(result.current.phase).toBe("erasing");
    pass(OPTIONS.eraseMs * 2);
    expect(result.current.text).toBe("");
    expect(result.current.index).toBe(1);

    pass(OPTIONS.gapMs + OPTIONS.typeMs);
    expect(result.current.text).toBe("cd");
  });

  it("holds a single line once it is written rather than erasing it", () => {
    const { result } = renderHook(() => useTypewriter(["ab"], OPTIONS));
    pass(OPTIONS.gapMs + OPTIONS.typeMs + OPTIONS.holdMs * 3);
    expect(result.current.text).toBe("ab");
    expect(result.current.complete).toBe(true);
  });

  it("starts over from the first line when the lines change", () => {
    const { result, rerender } = renderHook(({ lines }) => useTypewriter(lines, OPTIONS), {
      initialProps: { lines: ["ab", "cd"] },
    });
    pass(OPTIONS.gapMs + OPTIONS.typeMs);
    expect(result.current.text).toBe("ab");

    rerender({ lines: ["xy"] });
    expect(result.current.text).toBe("");
    pass(OPTIONS.gapMs + OPTIONS.typeMs);
    expect(result.current.text).toBe("xy");
  });

  it("shows whole lines when asked to be instant, and stands still when paused", () => {
    const { result, rerender } = renderHook(
      ({ paused }) => useTypewriter(["ab", "cd"], { ...OPTIONS, instant: true, paused }),
      { initialProps: { paused: false } },
    );
    pass(OPTIONS.gapMs);
    expect(result.current.text).toBe("ab");
    expect(result.current.complete).toBe(true);

    rerender({ paused: true });
    pass(OPTIONS.holdMs * 4);
    expect(result.current.text).toBe("ab");
  });
});
