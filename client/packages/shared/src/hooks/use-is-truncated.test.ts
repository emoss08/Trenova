import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useIsTruncated } from "./use-is-truncated";

function element(scrollWidth: number, clientWidth: number) {
  const node = document.createElement("span");
  Object.defineProperty(node, "scrollWidth", { configurable: true, value: scrollWidth });
  Object.defineProperty(node, "clientWidth", { configurable: true, value: clientWidth });
  return node;
}

describe("useIsTruncated", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reports text wider than its element as truncated", () => {
    const { result } = renderHook(() => useIsTruncated<HTMLSpanElement>());

    act(() => {
      result.current[0](element(400, 200));
    });

    expect(result.current[1]).toBe(true);
  });

  it("reports text that fits its element as not truncated", () => {
    const { result } = renderHook(() => useIsTruncated<HTMLSpanElement>());

    act(() => {
      result.current[0](element(200, 200));
    });

    expect(result.current[1]).toBe(false);
  });

  it("follows the element as it is resized", () => {
    let notify: () => void = () => {};
    const disconnect = vi.fn();
    vi.stubGlobal(
      "ResizeObserver",
      class {
        constructor(callback: () => void) {
          notify = callback;
        }
        observe() {}
        disconnect = disconnect;
      },
    );
    const node = element(400, 200);
    const { result } = renderHook(() => useIsTruncated<HTMLSpanElement>());

    act(() => {
      result.current[0](node);
    });
    expect(result.current[1]).toBe(true);

    Object.defineProperty(node, "clientWidth", { configurable: true, value: 500 });
    act(() => notify());

    expect(result.current[1]).toBe(false);
  });
});
