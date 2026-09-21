import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCalendarNow } from "../use-calendar-now";

const HOUR_MS = 60 * 60 * 1000;

describe("useCalendarNow", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("holds its reading while the reader's day is the same", () => {
    vi.setSystemTime(new Date("2026-09-21T10:00:00Z"));
    const { result } = renderHook(() => useCalendarNow("UTC"));
    const opened = result.current;

    act(() => {
      vi.advanceTimersByTime(3 * HOUR_MS);
    });

    expect(result.current).toBe(opened);
  });

  it("moves on once midnight passes in the reader's zone", () => {
    vi.setSystemTime(new Date("2026-09-21T23:59:30Z"));
    const { result } = renderHook(() => useCalendarNow("UTC"));
    const opened = result.current;

    act(() => {
      vi.advanceTimersByTime(2 * 60 * 1000);
    });

    expect(result.current).toBeGreaterThan(opened);
    expect(new Date(result.current * 1000).getUTCDate()).toBe(22);
  });

  it("counts midnight by the reader's zone, not the browser's", () => {
    // 04:30 UTC is still 21 September in Chicago (UTC-5): no new day yet.
    vi.setSystemTime(new Date("2026-09-22T04:30:00Z"));
    const { result } = renderHook(() => useCalendarNow("America/Chicago"));
    const opened = result.current;

    act(() => {
      vi.advanceTimersByTime(20 * 60 * 1000);
    });
    expect(result.current).toBe(opened);

    act(() => {
      vi.advanceTimersByTime(15 * 60 * 1000);
    });
    expect(result.current).toBeGreaterThan(opened);
  });
});
