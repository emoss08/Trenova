import { describe, expect, it } from "vitest";
import type { ContextMenuEntry } from "../menu-spec";
import {
  HEADER_HEIGHT,
  ITEM_HEIGHT,
  MENU_MARGIN,
  MENU_PADDING,
  SEPARATOR_HEIGHT,
  SUBMENU_MAX_HEIGHT,
  entryOffsets,
  placeMenu,
  placeSubmenu,
  scrollToReveal,
  visibleRange,
} from "../placement";

const screen = { width: 1440, height: 900 };
const menu = { width: 240, height: 320 };

describe("placeMenu", () => {
  it("opens at the pointer when the menu fits there", () => {
    expect(placeMenu({ x: 300, y: 200 }, menu, screen)).toEqual({
      x: 300,
      y: 200,
      maxHeight: screen.height - MENU_MARGIN * 2,
    });
  });

  it("opens to the left of the pointer near the right edge", () => {
    const placed = placeMenu({ x: 1300, y: 200 }, menu, screen);
    expect(placed.x).toBe(1300 - menu.width);
    expect(placed.y).toBe(200);
  });

  it("opens upward near the bottom edge", () => {
    const placed = placeMenu({ x: 300, y: 800 }, menu, screen);
    expect(placed.y).toBe(800 - menu.height);
  });

  it("flips both ways in the bottom-right corner", () => {
    expect(placeMenu({ x: 1430, y: 890 }, menu, screen)).toMatchObject({
      x: 1430 - menu.width,
      y: 890 - menu.height,
    });
  });

  it("is pushed back inside the margin when neither side of the pointer fits", () => {
    const narrow = { width: 300, height: 400 };
    const placed = placeMenu({ x: 150, y: 250 }, { width: 260, height: 300 }, narrow);
    expect(placed.x).toBeGreaterThanOrEqual(MENU_MARGIN);
    expect(placed.x + 260).toBeLessThanOrEqual(narrow.width - MENU_MARGIN);
    expect(placed.y).toBeGreaterThanOrEqual(MENU_MARGIN);
    expect(placed.y + 300).toBeLessThanOrEqual(narrow.height - MENU_MARGIN);
  });

  it("caps a menu taller than the window, which then scrolls, and keeps it on screen", () => {
    const placed = placeMenu({ x: 100, y: 600 }, { width: 240, height: 2000 }, screen);
    expect(placed.maxHeight).toBe(screen.height - MENU_MARGIN * 2);
    expect(placed.y).toBe(MENU_MARGIN);
  });
});

describe("placeSubmenu", () => {
  const row = { top: 300, bottom: 328, left: 400, right: 640 };

  it("opens to the right of its row, level with it", () => {
    expect(placeSubmenu(row, { width: 220, height: 200 }, screen)).toEqual({
      x: 640,
      y: 300 - MENU_PADDING,
      maxHeight: SUBMENU_MAX_HEIGHT,
      side: "right",
    });
  });

  it("opens to the left when the right edge is too close", () => {
    const nearEdge = { top: 300, bottom: 328, left: 1100, right: 1340 };
    const placed = placeSubmenu(nearEdge, { width: 220, height: 200 }, screen);
    expect(placed.side).toBe("left");
    expect(placed.x).toBe(1100 - 220);
  });

  it("slides up so a submenu opened near the bottom ends above the margin", () => {
    const low = { top: 860, bottom: 888, left: 400, right: 640 };
    const placed = placeSubmenu(low, { width: 220, height: 300 }, screen);
    expect(placed.y + 300).toBe(screen.height - MENU_MARGIN);
  });

  it("caps a long submenu so it scrolls, and keeps it inside a short window", () => {
    const placed = placeSubmenu(row, { width: 220, height: 1200 }, { width: 1440, height: 300 });
    expect(placed.maxHeight).toBe(300 - MENU_MARGIN * 2);
    expect(placed.y).toBeGreaterThanOrEqual(MENU_MARGIN);
    expect(placed.y + placed.maxHeight).toBeLessThanOrEqual(300 - MENU_MARGIN);
  });

  it("takes the roomier side, clamped, when neither side fits", () => {
    const tight = { width: 500, height: 900 };
    const middle = { top: 100, bottom: 128, left: 150, right: 350 };
    const placed = placeSubmenu(middle, { width: 220, height: 200 }, tight);
    expect(placed.x).toBeGreaterThanOrEqual(MENU_MARGIN);
    expect(placed.x + 220).toBeLessThanOrEqual(tight.width - MENU_MARGIN);
  });
});

describe("windowing a long menu", () => {
  const item: ContextMenuEntry = { label: "Person" };
  const entries: ContextMenuEntry[] = [
    { type: "header", label: "Assign to" },
    ...Array.from({ length: 98 }, (_, index) => ({ label: `Person ${index}` })),
    { type: "separator" },
    item,
  ];

  it("measures each kind of row at its own height", () => {
    const offsets = entryOffsets(entries);
    expect(offsets[1]).toBe(HEADER_HEIGHT);
    expect(offsets[2]).toBe(HEADER_HEIGHT + ITEM_HEIGHT);
    expect(offsets.at(-1)).toBe(HEADER_HEIGHT + 99 * ITEM_HEIGHT + SEPARATOR_HEIGHT);
  });

  it("draws only the rows in view, plus a few either side", () => {
    const offsets = entryOffsets(entries);
    const top = { start: 0, end: 0 };
    Object.assign(top, visibleRange(offsets, 0, 280, 2));
    expect(top).toEqual({ start: 0, end: 13 });

    const scrolled = visibleRange(offsets, HEADER_HEIGHT + 50 * ITEM_HEIGHT, 280, 2);
    expect(scrolled.start).toBe(49);
    expect(scrolled.end).toBe(63);

    const bottom = visibleRange(offsets, offsets.at(-1)! - 280, 280, 2);
    expect(bottom.end).toBe(entries.length);
  });

  it("scrolls only as far as it takes to show the active row", () => {
    const offsets = entryOffsets(entries);
    expect(scrollToReveal(offsets, 3, 0, 280)).toBe(0);
    expect(scrollToReveal(offsets, 20, 0, 280)).toBe(offsets[21] - 280);
    expect(scrollToReveal(offsets, 1, 400, 280)).toBe(offsets[1]);
  });
});
