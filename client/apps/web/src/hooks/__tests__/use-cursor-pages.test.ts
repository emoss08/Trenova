import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  canReachPage,
  initialCursorPages,
  moveToPage,
  recordCursorPage,
  useCursorPages,
} from "../use-cursor-pages";

describe("recordCursorPage", () => {
  it("learns where the next page starts and keeps the first count", () => {
    const first = recordCursorPage(initialCursorPages("scope"), {
      pageIndex: 0,
      endCursor: "c25",
      hasNextPage: true,
      totalCount: 60,
    });
    expect(first.cursors).toEqual({ 0: null, 1: "c25" });
    expect(first.totalCount).toBe(60);

    const second = recordCursorPage(first, {
      pageIndex: 1,
      endCursor: "c50",
      hasNextPage: true,
      totalCount: null,
    });
    expect(second.cursors).toEqual({ 0: null, 1: "c25", 2: "c50" });
    expect(second.totalCount).toBe(60);
  });

  // The last page has an end cursor too; it is not a way to a page after it.
  it("does not open a page after the last one", () => {
    const state = recordCursorPage(initialCursorPages("scope"), {
      pageIndex: 0,
      endCursor: "c7",
      hasNextPage: false,
      totalCount: 7,
    });
    expect(canReachPage(state, 1)).toBe(false);
    expect(moveToPage(state, 1)).toBe(state);
  });

  it("returns the same state when nothing new was learned", () => {
    const state = recordCursorPage(initialCursorPages("scope"), {
      pageIndex: 0,
      endCursor: "c25",
      hasNextPage: true,
      totalCount: 60,
    });
    expect(
      recordCursorPage(state, {
        pageIndex: 0,
        endCursor: "c25",
        hasNextPage: true,
        totalCount: null,
      }),
    ).toBe(state);
  });
});

describe("useCursorPages", () => {
  it("walks pages it has learned and starts over when the scope changes", () => {
    const { result, rerender } = renderHook(({ scope }) => useCursorPages(scope), {
      initialProps: { scope: "all" },
    });
    expect(result.current.pageIndex).toBe(0);
    expect(result.current.after).toBeNull();

    act(() => result.current.goToPage(1));
    expect(result.current.pageIndex).toBe(0);

    act(() =>
      result.current.recordPage({
        pageIndex: 0,
        endCursor: "c25",
        hasNextPage: true,
        totalCount: 30,
      }),
    );
    act(() => result.current.goToPage(1));
    expect(result.current.pageIndex).toBe(1);
    expect(result.current.after).toBe("c25");
    expect(result.current.totalCount).toBe(30);

    rerender({ scope: "email" });
    expect(result.current.pageIndex).toBe(0);
    expect(result.current.after).toBeNull();
    expect(result.current.totalCount).toBeNull();
  });
});
