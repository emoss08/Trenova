import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useListEntrance } from "../use-list-entrance";

type Props = { view: string; hasRows: boolean; shown: boolean };

const settle = () => act(() => void vi.advanceTimersByTime(480));

describe("useListEntrance, when a list's rows have finished rising in", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("is unsettled while the rows rise and settled once they have", () => {
    const { result } = renderHook(() => useListEntrance("Everything", true));

    expect(result.current).toBe(false);
    act(() => void vi.advanceTimersByTime(479));
    expect(result.current).toBe(false);
    act(() => void vi.advanceTimersByTime(1));
    expect(result.current).toBe(true);
  });

  it("waits for rows: a list still being read has nothing to settle", () => {
    const { result, rerender } = renderHook(
      (props: Props) => useListEntrance(props.view, props.hasRows, props.shown),
      { initialProps: { view: "Everything", hasRows: false, shown: true } },
    );

    settle();
    expect(result.current).toBe(false);

    rerender({ view: "Everything", hasRows: true, shown: true });
    settle();
    expect(result.current).toBe(true);
  });

  it("rises again for another view", () => {
    const { result, rerender } = renderHook(
      (props: Props) => useListEntrance(props.view, props.hasRows, props.shown),
      { initialProps: { view: "Everything", hasRows: true, shown: true } },
    );
    settle();
    expect(result.current).toBe(true);

    rerender({ view: "Watchtower", hasRows: true, shown: true });

    expect(result.current).toBe(false);
    settle();
    expect(result.current).toBe(true);
  });

  it("rises again when a view kept behind another comes back in front", () => {
    const { result, rerender } = renderHook(
      (props: Props) => useListEntrance(props.view, props.hasRows, props.shown),
      { initialProps: { view: "Decisions", hasRows: true, shown: true } },
    );
    settle();
    expect(result.current).toBe(true);

    rerender({ view: "Decisions", hasRows: true, shown: false });
    settle();
    expect(result.current).toBe(true);

    rerender({ view: "Decisions", hasRows: true, shown: true });

    expect(result.current).toBe(false);
    settle();
    expect(result.current).toBe(true);
  });
});
