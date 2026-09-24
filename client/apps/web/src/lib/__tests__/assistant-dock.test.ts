import { describe, expect, it } from "vitest";
import {
  clampPanelSize,
  dockPositionClass,
  isAssistantDock,
  isPanelSize,
  MIN_PANEL_SIZE,
  nearestDock,
  PANEL_RESIZE_STEP,
  panelSizeStyle,
  resizePanel,
  resizePanelByKey,
} from "../assistant-dock";

const viewport = { width: 1600, height: 900 };

/**
 * A dragged launcher lands in the corner of the quadrant it was let go in,
 * so a drop anywhere near a corner means that corner, and nowhere in between.
 */
describe("nearestDock", () => {
  it.each([
    [{ x: 1500, y: 850 }, "bottom-right"],
    [{ x: 100, y: 850 }, "bottom-left"],
    [{ x: 1500, y: 60 }, "top-right"],
    [{ x: 100, y: 60 }, "top-left"],
    [{ x: 801, y: 451 }, "bottom-right"],
    [{ x: 799, y: 449 }, "top-left"],
  ] as const)("drops %o in the %s corner", (point, dock) => {
    expect(nearestDock(point, viewport)).toBe(dock);
  });
});

/**
 * The panel is anchored in its corner and grows from the opposite one, so
 * pulling the handle away from the anchor always makes it bigger.
 */
describe("resizePanel", () => {
  const start = { width: 400, height: 500 };

  it("grows a bottom-right panel when pulled up and left", () => {
    expect(resizePanel("bottom-right", start, { x: -50, y: -30 })).toEqual({
      width: 450,
      height: 530,
    });
  });

  it("grows a bottom-left panel when pulled up and right", () => {
    expect(resizePanel("bottom-left", start, { x: 50, y: -30 })).toEqual({
      width: 450,
      height: 530,
    });
  });

  it("grows a top-right panel when pulled down and left", () => {
    expect(resizePanel("top-right", start, { x: -50, y: 30 })).toEqual({
      width: 450,
      height: 530,
    });
  });

  it("grows a top-left panel when pulled down and right", () => {
    expect(resizePanel("top-left", start, { x: 50, y: 30 })).toEqual({
      width: 450,
      height: 530,
    });
  });

  it("shrinks a panel pulled toward its anchor", () => {
    expect(resizePanel("bottom-right", start, { x: 40, y: 20 })).toEqual({
      width: 360,
      height: 480,
    });
  });
});

describe("resizePanelByKey", () => {
  const start = { width: 400, height: 500 };

  // The arrows move the handle, the same as a drag: for a panel in the
  // bottom right the handle is top-left, so left widens and up heightens.
  it("moves the handle the way the arrow points", () => {
    expect(resizePanelByKey("bottom-right", start, "ArrowLeft")).toEqual({
      width: 400 + PANEL_RESIZE_STEP,
      height: 500,
    });
    expect(resizePanelByKey("bottom-right", start, "ArrowUp")).toEqual({
      width: 400,
      height: 500 + PANEL_RESIZE_STEP,
    });
    expect(resizePanelByKey("top-left", start, "ArrowRight")).toEqual({
      width: 400 + PANEL_RESIZE_STEP,
      height: 500,
    });
    expect(resizePanelByKey("top-left", start, "ArrowDown")).toEqual({
      width: 400,
      height: 500 + PANEL_RESIZE_STEP,
    });
  });

  it("ignores keys that are not arrows", () => {
    expect(resizePanelByKey("bottom-right", start, "Enter")).toBeNull();
  });
});

describe("clampPanelSize", () => {
  it("never goes below the smallest usable panel", () => {
    expect(clampPanelSize({ width: 10, height: 10 }, viewport)).toEqual(MIN_PANEL_SIZE);
  });

  it("never outgrows the window", () => {
    expect(clampPanelSize({ width: 5000, height: 5000 }, viewport)).toEqual({
      width: viewport.width - 32,
      height: viewport.height - 72,
    });
  });

  // A window smaller than the minimum still gets a usable panel rather than
  // a negative one; the inline style caps it to the window again.
  it("keeps the minimum in a window smaller than the minimum", () => {
    expect(clampPanelSize({ width: 400, height: 500 }, { width: 200, height: 200 })).toEqual(
      MIN_PANEL_SIZE,
    );
  });

  it("rounds a size a drag left fractional", () => {
    expect(clampPanelSize({ width: 400.6, height: 500.2 }, viewport)).toEqual({
      width: 401,
      height: 500,
    });
  });
});

describe("placement classes", () => {
  // The top corners sit below the app header, never over it.
  it("keeps the top corners below the header", () => {
    expect(dockPositionClass("top-right", "launcher")).toContain("top-14");
    expect(dockPositionClass("top-left", "panel")).toContain("top-14");
    expect(dockPositionClass("top-left", "tab")).toContain("top-14");
  });

  it("puts the edge tab flush against the edge", () => {
    expect(dockPositionClass("bottom-right", "tab")).toContain("right-0");
    expect(dockPositionClass("bottom-left", "tab")).toContain("left-0");
  });

  it("leaves room for the header in a top corner's height", () => {
    expect(panelSizeStyle("top-right", { width: 380, height: 560 }).height).toBe(
      "min(560px, calc(100dvh - 4.5rem))",
    );
    expect(panelSizeStyle("bottom-right", { width: 380, height: 560 }).height).toBe(
      "min(560px, calc(100dvh - 2rem))",
    );
  });
});

describe("saved values", () => {
  it("accepts only the four corners", () => {
    expect(isAssistantDock("bottom-left")).toBe(true);
    expect(isAssistantDock("center")).toBe(false);
    expect(isAssistantDock(undefined)).toBe(false);
  });

  it("accepts only a size made of two finite numbers", () => {
    expect(isPanelSize({ width: 400, height: 500 })).toBe(true);
    expect(isPanelSize({ width: "400", height: 500 })).toBe(false);
    expect(isPanelSize({ width: Number.NaN, height: 500 })).toBe(false);
    expect(isPanelSize({ width: 400 })).toBe(false);
    expect(isPanelSize(null)).toBe(false);
  });
});
