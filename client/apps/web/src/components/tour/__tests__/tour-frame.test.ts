import { afterEach, describe, expect, it } from "vitest";
import { clipOf, findTourTarget, outgrowsItsClip, visibleBox } from "../tour-frame";

type Rect = { x: number; y: number; width: number; height: number };

function boxed(element: Element, rect: Rect): void {
  element.getBoundingClientRect = () => DOMRect.fromRect(rect);
}

function element(parent: Element, rect: Rect, style = ""): HTMLElement {
  const node = document.createElement("div");
  node.setAttribute("style", style);
  boxed(node, rect);
  parent.append(node);
  return node;
}

/** A long list inside a box that scrolls, under a chart: the areas of the freight outlook. */
function scrollingList() {
  const page = element(document.body, { x: 0, y: 0, width: 1000, height: 800 });
  const scroller = element(page, { x: 20, y: 400, width: 900, height: 300 }, "overflow: auto");
  const list = element(scroller, { x: 20, y: 180, width: 900, height: 2000 });
  list.setAttribute("data-tour", "areas");
  return { page, scroller, list };
}

describe("tour frame", () => {
  afterEach(() => {
    document.body.replaceChildren();
  });

  it("lights only the part of a long list that shows through the box it scrolls in", () => {
    const { list } = scrollingList();

    expect(clipOf(list)).toEqual({ x: 20, y: 400, width: 900, height: 300 });
    expect(visibleBox(list)).toEqual({ x: 20, y: 400, width: 900, height: 300 });
    expect(outgrowsItsClip(list)).toBe(true);
  });

  it("lights a part that fits as it stands", () => {
    const { page } = scrollingList();
    const figures = element(page, { x: 20, y: 60, width: 900, height: 80 });

    expect(visibleBox(figures)).toEqual({ x: 20, y: 60, width: 900, height: 80 });
    expect(outgrowsItsClip(figures)).toBe(false);
  });

  it("cuts a part down by every box that clips it, not only the nearest", () => {
    const outer = element(
      document.body,
      { x: 0, y: 0, width: 44, height: 800 },
      "overflow: hidden",
    );
    const inner = element(outer, { x: 0, y: 100, width: 400, height: 500 }, "overflow: auto");
    const section = element(inner, { x: 0, y: 150, width: 400, height: 200 });

    expect(visibleBox(section)).toEqual({ x: 0, y: 150, width: 44, height: 200 });
  });

  it("has nothing to light for a part scrolled out of its box or off the window", () => {
    const { scroller } = scrollingList();
    const above = element(scroller, { x: 20, y: 100, width: 900, height: 200 });
    const below = element(document.body, { x: 0, y: 5000, width: 900, height: 200 });

    expect(visibleBox(above)).toBeNull();
    expect(visibleBox(below)).toBeNull();
  });

  it("finds a part by its name", () => {
    const { list } = scrollingList();

    expect(findTourTarget("areas")).toBe(list);
    expect(findTourTarget("moves")).toBeNull();
  });

  it("leaves out a part inside something switched off, as a folded panel's contents are", () => {
    const { scroller, list } = scrollingList();

    scroller.setAttribute("inert", "");
    expect(findTourTarget("areas")).toBeNull();

    scroller.removeAttribute("inert");
    expect(findTourTarget("areas")).toBe(list);
  });
});
