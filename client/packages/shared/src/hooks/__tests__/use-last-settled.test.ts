import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useLastSettled } from "../use-last-settled";

describe("useLastSettled", () => {
  it("keeps the value from the last time it settled while a new one is on its way", () => {
    const { result, rerender } = renderHook(
      ({ value, settled }) => useLastSettled(value, settled),
      { initialProps: { value: 25, settled: true } },
    );

    rerender({ value: 100, settled: false });
    expect(result.current).toBe(25);

    rerender({ value: 100, settled: true });
    expect(result.current).toBe(100);
  });

  it("takes the new value in the same render it settles", () => {
    const seen: number[] = [];
    const { rerender } = renderHook(
      ({ value, settled }) => {
        const kept = useLastSettled(value, settled);
        seen.push(kept);
        return kept;
      },
      { initialProps: { value: 1, settled: true } },
    );

    rerender({ value: 2, settled: true });

    expect(seen.at(-1)).toBe(2);
    expect(seen).not.toContain(undefined);
  });

  it("keeps the value it has when a new one is the same by `same`", () => {
    const first = ["open"];
    const { result, rerender } = renderHook(
      ({ value }) =>
        useLastSettled(
          value,
          true,
          (a, b) => a.length === b.length && a.every((k, i) => k === b[i]),
        ),
      { initialProps: { value: first } },
    );

    rerender({ value: ["open"] });

    expect(result.current).toBe(first);
  });
});
