import { describe, expect, it } from "vitest";
import {
  LETTER_ASPECT,
  ZOOM_STEPS,
  boxInRegion,
  boxStyle,
  pageAspect,
  pagePercent,
  regionAspect,
  regionImageStyle,
  scrollToBox,
  sheetWidth,
  snippetRegion,
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

describe("snippetRegion", () => {
  it("frames a box with a margin of the text around it", () => {
    const region = snippetRegion([BOX]);

    expect(region.x).toBeCloseTo(0.57, 6);
    expect(region.y).toBeCloseTo(0.069, 6);
    expect(region.w).toBeCloseTo(0.2125, 6);
    expect(region.h).toBeCloseTo(0.046, 6);
  });

  it("frames every box given, however far apart", () => {
    const region = snippetRegion([
      { x: 0.1, y: 0.3, w: 0.2, h: 0.02 },
      { x: 0.5, y: 0.4, w: 0.1, h: 0.02 },
    ]);

    expect(region.x).toBeCloseTo(0.05, 6);
    expect(region.y).toBeCloseTo(0.284, 6);
    expect(region.x + region.w).toBeCloseTo(0.65, 6);
    expect(region.y + region.h).toBeCloseTo(0.436, 6);
  });

  it("stops at the edges of the page", () => {
    const region = snippetRegion([{ x: 0.01, y: 0.005, w: 0.98, h: 0.99 }]);

    expect(region).toEqual({ x: 0, y: 0, w: 1, h: 1 });
  });
});

describe("boxInRegion", () => {
  it("places a box as fractions of the region it is drawn over", () => {
    const box = boxInRegion(
      { x: 0.3, y: 0.45, w: 0.2, h: 0.05 },
      { x: 0.2, y: 0.4, w: 0.4, h: 0.2 },
    );

    expect(box.x).toBeCloseTo(0.25, 6);
    expect(box.y).toBeCloseTo(0.25, 6);
    expect(box.w).toBeCloseTo(0.5, 6);
    expect(box.h).toBeCloseTo(0.25, 6);
  });

  it("fills the whole snippet when the region is the box itself", () => {
    expect(boxInRegion(BOX, BOX)).toEqual({ x: 0, y: 0, w: 1, h: 1 });
  });
});

describe("regionAspect", () => {
  it("keeps the region's own shape on a page of the given shape", () => {
    expect(regionAspect({ x: 0, y: 0, w: 1, h: 1 }, LETTER_ASPECT)).toBeCloseTo(LETTER_ASPECT, 6);
    expect(regionAspect({ x: 0.1, y: 0.1, w: 0.5, h: 0.1 }, LETTER_ASPECT)).toBeCloseTo(
      5 * LETTER_ASPECT,
      6,
    );
  });
});

describe("regionImageStyle", () => {
  it("draws the whole page so only the region shows through the frame", () => {
    expect(regionImageStyle({ x: 0.25, y: 0.1, w: 0.5, h: 0.2 })).toEqual({
      width: "200%",
      height: "500%",
      left: "-50%",
      top: "-50%",
    });
  });

  it("draws the page as it is when the region is the whole page", () => {
    expect(regionImageStyle({ x: 0, y: 0, w: 1, h: 1 })).toEqual({
      width: "100%",
      height: "100%",
      left: "0%",
      top: "0%",
    });
  });
});
