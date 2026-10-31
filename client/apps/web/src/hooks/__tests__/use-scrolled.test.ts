import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useScrolled } from "../use-scrolled";

// jsdom lays nothing out, so the scroll position is set as the browser would report it.
function scrollTo(node: HTMLDivElement, top: number) {
  Object.defineProperty(node, "scrollTop", { value: top, configurable: true });
  node.dispatchEvent(new Event("scroll"));
}

describe("useScrolled", () => {
  it("says when the area has left its top, and when it is back", () => {
    const { result } = renderHook(() => useScrolled());
    const node = document.createElement("div");
    act(() => result.current.ref(node));
    expect(result.current.scrolled).toBe(false);

    act(() => scrollTo(node, 40));
    expect(result.current.scrolled).toBe(true);

    act(() => scrollTo(node, 0));
    expect(result.current.scrolled).toBe(false);
  });

  it("stops following an area it was taken off", () => {
    const { result } = renderHook(() => useScrolled());
    const first = document.createElement("div");
    const second = document.createElement("div");
    act(() => result.current.ref(first));
    act(() => result.current.ref(second));

    act(() => scrollTo(first, 40));
    expect(result.current.scrolled).toBe(false);

    act(() => scrollTo(second, 40));
    expect(result.current.scrolled).toBe(true);
  });
});
