import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useSharedItems, useSharedMap } from "../use-keyed-sharing";

type Row = { id: string; text: string };

const byId = (row: Row) => row.id;

/**
 * Derived rows rebuilt on every change keep their objects across renders
 * while they are equal, so memoized components drawn from them are skipped.
 */
describe("useSharedItems", () => {
  it("keeps the rows that were there when more arrive in front", () => {
    const first: Row[] = [{ id: "b", text: "two" }];
    const { result, rerender } = renderHook(({ rows }) => useSharedItems(rows, byId), {
      initialProps: { rows: first },
    });
    const drawn = result.current;

    rerender({
      rows: [
        { id: "a", text: "one" },
        { id: "b", text: "two" },
      ],
    });

    expect(result.current).toHaveLength(2);
    expect(result.current[1]).toBe(drawn[0]);
  });

  it("keeps the whole list when it was rebuilt equal", () => {
    const { result, rerender } = renderHook(({ rows }) => useSharedItems(rows, byId), {
      initialProps: { rows: [{ id: "a", text: "one" }] },
    });
    const drawn = result.current;

    rerender({ rows: [{ id: "a", text: "one" }] });

    expect(result.current).toBe(drawn);
  });
});

describe("useSharedMap", () => {
  it("keeps each value that was rebuilt equal", () => {
    const { result, rerender } = renderHook(({ map }) => useSharedMap(map), {
      initialProps: { map: new Map([["m2", ["lookup"]]]) },
    });
    const steps = result.current.get("m2");

    rerender({
      map: new Map([
        ["m1", ["older"]],
        ["m2", ["lookup"]],
      ]),
    });

    expect(result.current.get("m2")).toBe(steps);
  });
});
