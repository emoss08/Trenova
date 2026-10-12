import { afterEach, describe, expect, it } from "vitest";
import { scrollParent } from "@/lib/dom";

function box(overflowY: string, parent: HTMLElement, sizes?: { scroll: number; client: number }) {
  const element = document.createElement("div");
  element.style.overflowY = overflowY;
  if (sizes) {
    Object.defineProperty(element, "scrollHeight", { value: sizes.scroll });
    Object.defineProperty(element, "clientHeight", { value: sizes.client });
  }
  parent.append(element);
  return element;
}

afterEach(() => {
  document.body.replaceChildren();
});

describe("scrollParent", () => {
  it("finds the nearest ancestor that scrolls, passing those that do not", () => {
    const pane = box("auto", document.body);
    const card = box("visible", pane);
    const row = box("visible", card);

    expect(scrollParent(row)).toBe(pane);
    expect(scrollParent(card)).toBe(pane);
  });

  it("does not count the element itself, and stops short of the page", () => {
    const pane = box("scroll", document.body);

    expect(scrollParent(pane)).toBeNull();
    expect(scrollParent(null)).toBeNull();
  });

  it("passes an ancestor with nothing to scroll yet when asked for one that overflows", () => {
    const pane = box("auto", document.body, { scroll: 900, client: 400 });
    const code = box("auto", pane, { scroll: 40, client: 40 });
    const line = box("visible", code);

    expect(scrollParent(line)).toBe(code);
    expect(scrollParent(line, { overflowing: true })).toBe(pane);
  });
});
