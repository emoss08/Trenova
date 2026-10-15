import { renderHook } from "@testing-library/react";
import { StrictMode, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useObjectUrl } from "./use-object-url";

describe("useObjectUrl", () => {
  let made: string[];
  let revoked: Set<string>;

  beforeEach(() => {
    made = [];
    revoked = new Set();
    vi.spyOn(URL, "createObjectURL").mockImplementation(() => {
      const url = `blob:test/${made.length + 1}`;
      made.push(url);
      return url;
    });
    vi.spyOn(URL, "revokeObjectURL").mockImplementation((url: string) => {
      revoked.add(url);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("is null without a blob", () => {
    const { result } = renderHook(() => useObjectUrl(null));
    expect(result.current).toBeNull();
    expect(made).toEqual([]);
  });

  // React's development double mount runs the effect, its cleanup, and the effect again.
  // The URL in use afterwards must be one that has not been revoked.
  it("hands back a live URL under StrictMode's double mount", () => {
    const blob = new Blob(["%PDF"], { type: "application/pdf" });
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(StrictMode, null, children);
    const { result } = renderHook(() => useObjectUrl(blob), { wrapper });

    expect(result.current).not.toBeNull();
    expect(revoked.has(result.current as string)).toBe(false);
  });

  it("revokes the old URL when the blob changes, and the last one on unmount", () => {
    const first = new Blob(["a"]);
    const second = new Blob(["b"]);
    const { result, rerender, unmount } = renderHook(({ blob }) => useObjectUrl(blob), {
      initialProps: { blob: first },
    });
    const firstUrl = result.current as string;

    rerender({ blob: second });
    const secondUrl = result.current as string;
    expect(secondUrl).not.toBe(firstUrl);
    expect(revoked.has(firstUrl)).toBe(true);
    expect(revoked.has(secondUrl)).toBe(false);

    unmount();
    expect(revoked.has(secondUrl)).toBe(true);
  });
});
