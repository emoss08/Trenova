import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useOpenFirstItem } from "../use-open-first-item";

describe("useOpenFirstItem", () => {
  it("opens the first item the queue lists when none is open", () => {
    const open = vi.fn();
    const { result } = renderHook(() => useOpenFirstItem(null, open));

    expect(result.current.firstItemId).toBeUndefined();
    expect(open).not.toHaveBeenCalled();

    act(() => result.current.reportFirstItem("bqi_first"));

    expect(open).toHaveBeenCalledExactlyOnceWith("bqi_first");
  });

  it("leaves an open item open when the list answers", () => {
    const open = vi.fn();
    const { result } = renderHook(() => useOpenFirstItem("bqi_open", open));

    act(() => result.current.reportFirstItem("bqi_first"));

    expect(open).not.toHaveBeenCalled();
  });

  it("opens the first item again once the open one is closed", () => {
    const open = vi.fn();
    const { result, rerender } = renderHook(
      ({ selected }: { selected: string | null }) => useOpenFirstItem(selected, open),
      { initialProps: { selected: "bqi_open" as string | null } },
    );
    act(() => result.current.reportFirstItem("bqi_first"));

    rerender({ selected: null });

    expect(open).toHaveBeenCalledExactlyOnceWith("bqi_first");
  });

  it("opens nothing and says so when the queue lists no item", () => {
    const open = vi.fn();
    const { result } = renderHook(() => useOpenFirstItem(null, open));

    act(() => result.current.reportFirstItem(null));

    expect(result.current.firstItemId).toBeNull();
    expect(open).not.toHaveBeenCalled();
  });
});
