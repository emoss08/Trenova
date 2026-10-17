import { describe, expect, it } from "vitest";
import {
  LETTER_ASPECT,
  ZOOM_STEPS,
  boxStyle,
  pageAspect,
  pagePercent,
  scrollToBox,
  sheetWidth,
  stepZoom,
} from "../page-geometry";

const BOX = { x: 0.62, y: 0.085, w: 0.1125, h: 0.014 };

describe("pagePercent", () => {
  it("writes a fraction as a percentage without float noise", () => {
    expect(pagePercent(0.14)).toBe("14%");
    expect(pagePercent(0.0183)).toBe("1.83%");
    expect(pagePercent(0)).toBe("0%");
  });
});

describe("boxStyle", () => {
  it("places a box in percentages of the drawn page, whatever size it is drawn at", () => {
    expect(boxStyle(BOX)).toEqual({ left: "62%", top: "8.5%", width: "11.25%", height: "1.4%" });
  });

  it("lands on the same pixels of the page at every rendered size", () => {
    for (const width of [320, 637, 1275, 2550]) {
      const height = width / LETTER_ASPECT;
      const style = boxStyle(BOX);
      const left = (parseFloat(String(style.left)) / 100) * width;
      const top = (parseFloat(String(style.top)) / 100) * height;
      expect(left / width).toBeCloseTo(BOX.x, 6);
      expect(top / height).toBeCloseTo(BOX.y, 6);
    }
  });
});

describe("pageAspect", () => {
  it("frames a page in its own shape, and a letter page when its size is not known", () => {
    expect(pageAspect({ width: 1650, height: 1275 })).toBeCloseTo(1650 / 1275, 6);
    expect(pageAspect({ width: 0, height: 0 })).toBe(LETTER_ASPECT);
    expect(pageAspect(undefined)).toBe(LETTER_ASPECT);
  });
});

describe("sheetWidth", () => {
  it("fits the viewer but never draws a page narrower than it reads, and zooms from there", () => {
    expect(sheetWidth(1)).toBe("calc(max(100%, 600px) * 1)");
    expect(sheetWidth(2)).toBe("calc(max(100%, 600px) * 2)");
  });
});

describe("scrollToBox", () => {
  const frame = {
    pageLeft: 12,
    pageTop: 12,
    pageWidth: 1000,
    pageHeight: 1294,
    viewWidth: 500,
    viewHeight: 400,
  };

  it("centres the box in the viewer", () => {
    expect(scrollToBox({ x: 0.5, y: 0.5, w: 0.2, h: 0.02 }, frame)).toEqual({
      left: 362,
      top: 472,
    });
  });

  it("centres the same box at twice the size, as a zoom step draws it", () => {
    expect(
      scrollToBox(
        { x: 0.5, y: 0.5, w: 0.2, h: 0.02 },
        { ...frame, pageWidth: 2000, pageHeight: 2588 },
      ),
    ).toEqual({ left: 962, top: 1132 });
  });

  it("never scrolls before the start of the page", () => {
    expect(scrollToBox({ x: 0.02, y: 0.01, w: 0.1, h: 0.02 }, frame)).toEqual({ left: 0, top: 0 });
  });
});

describe("stepZoom", () => {
  it("steps through the zoom levels and stays at either end", () => {
    expect(stepZoom(1, 1)).toBe(1.25);
    expect(stepZoom(2, -1)).toBe(1.5);
    expect(stepZoom(1, -1)).toBe(1);
    expect(stepZoom(ZOOM_STEPS[ZOOM_STEPS.length - 1], 1)).toBe(3);
  });
});
