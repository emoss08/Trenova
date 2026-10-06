import {
  TYPEWRITER_SETTLE_MS,
  TYPEWRITER_THINK_MS,
  typewriterDelay,
  useTypewriter,
} from "@/hooks/use-typewriter";
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const motion = vi.hoisted(() => ({ reduce: false }));

vi.mock("motion/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("motion/react")>()),
  useReducedMotion: () => motion.reduce,
}));

describe("typewriterDelay", () => {
  it("pauses longest after a sentence, then a clause, then a line", () => {
    expect(typewriterDelay("a", 0)).toBe(12);
    expect(typewriterDelay("a", 1)).toBe(30);
    expect(typewriterDelay(".", 0)).toBe(272);
    expect(typewriterDelay("?", 0)).toBe(272);
    expect(typewriterDelay(",", 0)).toBe(122);
    expect(typewriterDelay("—", 0)).toBe(122);
    expect(typewriterDelay("\n", 0)).toBe(212);
  });
});

describe("useTypewriter", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(Math, "random").mockReturnValue(0);
    motion.reduce = false;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("thinks, types at the documented cadence, then reports done once", () => {
    const onDone = vi.fn();
    const { result } = renderHook(() => useTypewriter("Hi. Go", { animate: true, onDone }));

    expect(result.current).toEqual({ revealed: 0, thinking: true, typing: false });
    act(() => {
      vi.advanceTimersByTime(TYPEWRITER_THINK_MS);
    });
    expect(result.current).toEqual({ revealed: 1, thinking: false, typing: true });

    act(() => {
      vi.advanceTimersByTime(12);
    });
    expect(result.current.revealed).toBe(2);
    act(() => {
      vi.advanceTimersByTime(12);
    });
    expect(result.current.revealed).toBe(3);

    act(() => {
      vi.advanceTimersByTime(271);
    });
    expect(result.current.revealed).toBe(3);
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(result.current.revealed).toBe(4);

    act(() => {
      vi.advanceTimersByTime(24);
    });
    expect(result.current).toEqual({ revealed: 6, thinking: false, typing: false });
    expect(onDone).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(TYPEWRITER_SETTLE_MS);
    });
    expect(onDone).toHaveBeenCalledTimes(1);
  });

  it("shows a line it is not asked to animate whole, and still reports done", () => {
    const onDone = vi.fn();
    const { result } = renderHook(() =>
      useTypewriter("Already typed.", { animate: false, onDone }),
    );

    expect(result.current).toEqual({ revealed: 14, thinking: false, typing: false });
    expect(onDone).toHaveBeenCalled();
  });

  it("shows the line at once under reduced motion", () => {
    motion.reduce = true;
    const onDone = vi.fn();
    const { result } = renderHook(() => useTypewriter("Hello.", { animate: true, onDone }));

    expect(result.current).toEqual({ revealed: 6, thinking: false, typing: false });
    expect(onDone).toHaveBeenCalled();
  });

  it("never types again after it has started whole", () => {
    const { result, rerender } = renderHook(
      ({ animate }) => useTypewriter("Typed once.", { animate }),
      { initialProps: { animate: false } },
    );
    rerender({ animate: true });
    act(() => {
      vi.advanceTimersByTime(TYPEWRITER_THINK_MS);
    });

    expect(result.current).toEqual({ revealed: 11, thinking: false, typing: false });
  });
});
