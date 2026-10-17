import type { CSSProperties } from "react";

/** A region of the page as fractions of its width and height. */
export type PageBox = { x: number; y: number; w: number; h: number };

/** Where the drawn page sits in the viewer that scrolls it, in pixels. */
export type PageFrame = {
  pageLeft: number;
  pageTop: number;
  pageWidth: number;
  pageHeight: number;
  viewWidth: number;
  viewHeight: number;
};

/** A letter page's width over its height: the frame a page is drawn in before its size is known. */
export const LETTER_ASPECT = 8.5 / 11;

/** How wide the page is drawn, as a multiple of the viewer's width; 1 fits it to the width. */
export const ZOOM_STEPS = [1, 1.25, 1.5, 2, 2.5, 3] as const;

export type Zoom = (typeof ZOOM_STEPS)[number];

/**
 * How wide a page must be drawn for its body text to read: a letter page set in 9 to 10 point
 * type reaches about 11px at this width.
 */
export const READABLE_PAGE_WIDTH = 600;

/** A fraction of the page as a CSS percentage, kept to the four places the server rounds to. */
export function pagePercent(fraction: number): string {
  return `${Math.round(fraction * 10000) / 100}%`;
}

/**
 * A box placed over the drawn page. It is positioned in percentages of an overlay that is
 * exactly the image's own box, so it stays on its text at any width, zoom or pixel ratio.
 */
export function boxStyle(box: PageBox): CSSProperties {
  return {
    left: pagePercent(box.x),
    top: pagePercent(box.y),
    width: pagePercent(box.w),
    height: pagePercent(box.h),
  };
}

/** A page's width over its height, from a size when it is known, else a letter page's. */
export function pageAspect(size: { width: number; height: number } | null | undefined): number {
  return size && size.width > 0 && size.height > 0 ? size.width / size.height : LETTER_ASPECT;
}

/** The scroll that puts a box in the middle of the viewer, as far as the page allows. */
export function scrollToBox(box: PageBox, frame: PageFrame): { left: number; top: number } {
  const centreX = frame.pageLeft + (box.x + box.w / 2) * frame.pageWidth;
  const centreY = frame.pageTop + (box.y + box.h / 2) * frame.pageHeight;
  return {
    left: Math.max(0, Math.round(centreX - frame.viewWidth / 2)),
    top: Math.max(0, Math.round(centreY - frame.viewHeight / 2)),
  };
}

/** The next zoom step in a direction, staying on the first or last step at either end. */
export function stepZoom(zoom: Zoom, direction: 1 | -1): Zoom {
  const at = ZOOM_STEPS.indexOf(zoom);
  return ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, at + direction))];
}

/**
 * How wide the sheet is drawn: the viewer's width at Fit, but never narrower than a readable
 * page, so on a phone the page scrolls sideways rather than shrinking past reading; each zoom
 * step multiplies that.
 */
export function sheetWidth(zoom: Zoom): string {
  return `calc(max(100%, ${READABLE_PAGE_WIDTH}px) * ${zoom})`;
}

/** The margin a snippet keeps around what it shows, so the label beside a value reads with it. */
const SNIPPET_MARGIN = { x: 0.05, y: 0.016 };

/** The part of the page a snippet shows: every box given, with a margin of the text around them. */
export function snippetRegion(boxes: readonly PageBox[]): PageBox {
  const left = Math.min(...boxes.map((box) => box.x));
  const top = Math.min(...boxes.map((box) => box.y));
  const right = Math.max(...boxes.map((box) => box.x + box.w));
  const bottom = Math.max(...boxes.map((box) => box.y + box.h));
  const x = Math.max(0, left - SNIPPET_MARGIN.x);
  const y = Math.max(0, top - SNIPPET_MARGIN.y);
  return {
    x,
    y,
    w: Math.min(1, right + SNIPPET_MARGIN.x) - x,
    h: Math.min(1, bottom + SNIPPET_MARGIN.y) - y,
  };
}

/** A box as fractions of a region of its page, for drawing it over that region's snippet. */
export function boxInRegion(box: PageBox, region: PageBox): PageBox {
  return {
    x: (box.x - region.x) / region.w,
    y: (box.y - region.y) / region.h,
    w: box.w / region.w,
    h: box.h / region.h,
  };
}

/** The snippet's width over its height: the region's own shape on a page of the given shape. */
export function regionAspect(region: PageBox, aspect: number): number {
  return (region.w / region.h) * aspect;
}

/**
 * The whole page placed so only the region shows through its snippet's frame: the page drawn
 * as many times wider and taller than the frame as the region is smaller than the page, and
 * moved up and left by where the region starts.
 */
export function regionImageStyle(region: PageBox): CSSProperties {
  return {
    width: pagePercent(1 / region.w),
    height: pagePercent(1 / region.h),
    left: pagePercent(-region.x / region.w),
    top: pagePercent(-region.y / region.h),
  };
}
