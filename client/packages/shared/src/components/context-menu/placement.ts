import type { ContextMenuEntry } from "./menu-spec";

/**
 * Where a menu goes and how much of it is drawn.
 *
 * A menu opens at the pointer and must stay whole on screen: near the right
 * edge it opens to the left of the pointer instead, near the bottom it opens
 * upward, and when neither fits it is pushed back inside a margin from every
 * edge. A menu taller than the window scrolls rather than running off it. A
 * submenu opens beside its row, on whichever side has room, level with the
 * row. All of it is arithmetic on rectangles, kept apart from the component
 * so it can be checked without a browser.
 */
export type Point = { x: number; y: number };
export type Size = { width: number; height: number };
export type Rect = { top: number; left: number; right: number; bottom: number };

export type Placement = {
  x: number;
  y: number;
  /** The tallest the menu may be before it scrolls. */
  maxHeight: number;
};

export type SubmenuPlacement = Placement & { side: "right" | "left" };

/** The gap kept between a menu and the window's edge. */
export const MENU_MARGIN = 8;
/** A submenu is capped shorter than the window so a long one scrolls inside a comfortable height. */
export const SUBMENU_MAX_HEIGHT = 340;
/** The menu's own padding, so a submenu's first row lines up with the row that opened it. */
export const MENU_PADDING = 4;

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(value, max));
}

export function placeMenu(
  point: Point,
  size: Size,
  viewport: Size,
  margin = MENU_MARGIN,
): Placement {
  const maxHeight = Math.max(0, viewport.height - margin * 2);
  const height = Math.min(size.height, maxHeight);
  const width = Math.min(size.width, Math.max(0, viewport.width - margin * 2));

  const fitsRight = point.x + width <= viewport.width - margin;
  const x = clamp(fitsRight ? point.x : point.x - width, margin, viewport.width - margin - width);

  const fitsBelow = point.y + height <= viewport.height - margin;
  const y = clamp(
    fitsBelow ? point.y : point.y - height,
    margin,
    viewport.height - margin - height,
  );

  return { x, y, maxHeight };
}

export function placeSubmenu(
  anchor: Rect,
  size: Size,
  viewport: Size,
  margin = MENU_MARGIN,
  cap = SUBMENU_MAX_HEIGHT,
): SubmenuPlacement {
  const maxHeight = Math.max(0, Math.min(cap, viewport.height - margin * 2));
  const height = Math.min(size.height, maxHeight);
  const width = Math.min(size.width, Math.max(0, viewport.width - margin * 2));

  const roomRight = viewport.width - margin - anchor.right;
  const roomLeft = anchor.left - margin;
  let side: "right" | "left";
  let x: number;
  if (width <= roomRight) {
    side = "right";
    x = anchor.right;
  } else if (width <= roomLeft) {
    side = "left";
    x = anchor.left - width;
  } else {
    side = roomRight >= roomLeft ? "right" : "left";
    x = clamp(
      side === "right" ? anchor.right : anchor.left - width,
      margin,
      viewport.width - margin - width,
    );
  }

  const y = clamp(anchor.top - MENU_PADDING, margin, viewport.height - margin - height);

  return { x, y, maxHeight, side };
}

/** Row heights, in pixels, so a long menu can draw only the rows in view. */
export const ITEM_HEIGHT = 28;
export const SEPARATOR_HEIGHT = 9;
export const HEADER_HEIGHT = 26;
/** Menus longer than this draw only the rows in view. */
export const VIRTUALIZE_AFTER = 30;

export function entryHeight(entry: ContextMenuEntry): number {
  switch (entry.type) {
    case "separator":
      return SEPARATOR_HEIGHT;
    case "header":
      return HEADER_HEIGHT;
    default:
      return ITEM_HEIGHT;
  }
}

/** Where each row starts, plus the total height as the last element. */
export function entryOffsets(entries: readonly ContextMenuEntry[]): number[] {
  const offsets = [0];
  for (const entry of entries) {
    offsets.push(offsets[offsets.length - 1] + entryHeight(entry));
  }

  return offsets;
}

function rowAt(offsets: readonly number[], position: number): number {
  let low = 0;
  let high = offsets.length - 2;
  while (low < high) {
    const middle = Math.ceil((low + high) / 2);
    if (offsets[middle] <= position) {
      low = middle;
    } else {
      high = middle - 1;
    }
  }

  return Math.max(0, low);
}

/** The rows in view, with a few either side, as a half-open range. */
export function visibleRange(
  offsets: readonly number[],
  scrollTop: number,
  viewportHeight: number,
  overscan = 4,
): { start: number; end: number } {
  const count = offsets.length - 1;
  if (count <= 0) {
    return { start: 0, end: 0 };
  }
  const first = rowAt(offsets, Math.max(0, scrollTop));
  const last = rowAt(offsets, Math.max(0, scrollTop + viewportHeight - 1));

  return { start: Math.max(0, first - overscan), end: Math.min(count, last + 1 + overscan) };
}

/** The scroll position that brings a row fully into view, or the current one when it is. */
export function scrollToReveal(
  offsets: readonly number[],
  index: number,
  scrollTop: number,
  viewportHeight: number,
): number {
  const top = offsets[index];
  const bottom = offsets[index + 1];
  if (top < scrollTop) {
    return top;
  }
  if (bottom > scrollTop + viewportHeight) {
    return bottom - viewportHeight;
  }

  return scrollTop;
}
