import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { usePendingValue } from "./use-pending-value";

describe("usePendingValue", () => {
  it("follows the value while nothing is pending", () => {
    const { result, rerender } = renderHook(({ value }) => usePendingValue(value), {
      initialProps: { value: "a" },
    });
    expect(result.current[0]).toBe("a");

    rerender({ value: "b" });

    expect(result.current[0]).toBe("b");
  });

  it("shows what was set while the value it stands for has not changed", () => {
    const { result, rerender } = renderHook(({ value }) => usePendingValue(value), {
      initialProps: { value: false },
    });

    act(() => result.current[1](true));
    rerender({ value: false });

    expect(result.current[0]).toBe(true);
  });

  it("gives way to whatever the value becomes next", () => {
    const { result, rerender } = renderHook(({ value }) => usePendingValue(value), {
      initialProps: { value: "Late loads" },
    });

    act(() => result.current[1]("detention claims"));
    rerender({ value: "Detention claims" });

    expect(result.current[0]).toBe("Detention claims");
  });

  it("shows the old value again when the change is refused after it landed", () => {
    const { result, rerender } = renderHook(({ value }) => usePendingValue(value), {
      initialProps: { value: false },
    });

    act(() => result.current[1](true));
    rerender({ value: true });
    rerender({ value: false });

    expect(result.current[0]).toBe(false);
  });

  it("holds a pending value equal to nothing at all", () => {
    const { result } = renderHook(() => usePendingValue<string | null>("a"));

    act(() => result.current[1](null));

    expect(result.current[0]).toBeNull();
  });
});
