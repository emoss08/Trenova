import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { RAIL_OPEN_STORAGE_KEY, useStudioRail } from "../use-studio-rail";

describe("useStudioRail", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("starts open beside the form on a first visit", () => {
    const { result } = renderHook(() => useStudioRail(false));
    expect(result.current[0]).toBe(true);
  });

  it("stays closed on the next visit after the person closed it beside the form", () => {
    const first = renderHook(() => useStudioRail(false));
    act(() => first.result.current[1](false));
    expect(first.result.current[0]).toBe(false);
    expect(window.localStorage.getItem(RAIL_OPEN_STORAGE_KEY)).toBe("false");
    first.unmount();

    const next = renderHook(() => useStudioRail(false));
    expect(next.result.current[0]).toBe(false);
  });

  it("starts closed over the form on a narrow screen even when it was left open", () => {
    window.localStorage.setItem(RAIL_OPEN_STORAGE_KEY, "true");
    const { result } = renderHook(() => useStudioRail(true));
    expect(result.current[0]).toBe(false);
  });

  it("opens over the form without changing what the wide screen remembers", () => {
    window.localStorage.setItem(RAIL_OPEN_STORAGE_KEY, "false");
    const { result, rerender } = renderHook(({ stacked }) => useStudioRail(stacked), {
      initialProps: { stacked: true },
    });

    act(() => result.current[1](true));
    expect(result.current[0]).toBe(true);
    expect(window.localStorage.getItem(RAIL_OPEN_STORAGE_KEY)).toBe("false");

    rerender({ stacked: false });
    expect(result.current[0]).toBe(false);
  });
});
