/** A letter page's width at 100%: 8.5 inches at 96 pixels an inch. */
export const ACTUAL_PAGE_WIDTH = 816;
export const ZOOM_STEPS = [0.5, 0.75, 1, 1.25, 1.5, 2, 2.5, 3] as const;
const MIN_SCALE = 0.25;
const MAX_SCALE = 4;
const MIN_SHEET_WIDTH = 160;

/** How the column sizes its pages: to the viewer's width, to its height, or at a scale. */
export type PageZoom = { mode: "width" } | { mode: "page" } | { mode: "scale"; scale: number };

export const FIT_WIDTH: PageZoom = { mode: "width" };

/** The room the column keeps around a page: its gutters, and above and below it. */
export type ZoomFrame = {
  width: number;
  height: number;
  gutter: number;
  /** Height the column spends on a page that is not the page: its label, the toolbar's room. */
  chrome: number;
};

/** A page's drawn width under a zoom, for a page whose box is `aspect` wide over tall. */
export function sheetWidthFor(zoom: PageZoom, frame: ZoomFrame, aspect: number): number {
  const width =
    zoom.mode === "width"
      ? frame.width - frame.gutter
      : zoom.mode === "page"
        ? (frame.height - frame.chrome) * aspect
        : ACTUAL_PAGE_WIDTH * zoom.scale;
  return Math.max(MIN_SHEET_WIDTH, Math.round(width));
}

/** The next step up or down from a page drawn at `width`, whatever mode drew it. */
export function stepZoom(width: number, direction: 1 | -1): PageZoom {
  const scale = width / ACTUAL_PAGE_WIDTH;
  const next =
    direction > 0
      ? ZOOM_STEPS.find((step) => step > scale + 0.01)
      : [...ZOOM_STEPS].reverse().find((step) => step < scale - 0.01);
  const edge = direction > 0 ? ZOOM_STEPS[ZOOM_STEPS.length - 1] : ZOOM_STEPS[0];
  return { mode: "scale", scale: next ?? edge };
}

/** A pinch or Ctrl + scroll from a page drawn at `width`: a tenth larger or smaller a notch. */
export function wheelZoom(width: number, deltaY: number): PageZoom {
  const scale = (width / ACTUAL_PAGE_WIDTH) * (deltaY < 0 ? 1.1 : 1 / 1.1);
  return { mode: "scale", scale: Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale)) };
}
