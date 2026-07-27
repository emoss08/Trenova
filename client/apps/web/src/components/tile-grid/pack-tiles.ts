/**
 * The minimum a tile-grid item must expose. Position is deliberately absent:
 * items are an ordered list that flows into the grid, so moving one is a
 * reorder rather than a coordinate edit.
 */
export type GridItem = {
  id: string;
  w: number;
  h: number;
};

export type PackedGridItem<T extends GridItem> = T & {
  x: number;
  y: number;
};

export type PackTilesOptions = {
  columns: number;
  defaultW?: number;
  defaultH?: number;
};

/**
 * Flows an ordered list left-to-right into a fixed-column grid and derives each
 * item's x/y. Callers that persist coordinates can store the result; callers
 * that persist order alone can use it purely for rendering.
 */
export function packTiles<T extends GridItem>(
  items: T[],
  { columns, defaultW = 4, defaultH = 4 }: PackTilesOptions,
): PackedGridItem<T>[] {
  let x = 0;
  let y = 0;
  let rowHeight = 0;

  return items.map((item) => {
    const w = Math.min(Math.max(item.w || defaultW, 1), columns);
    const h = Math.max(item.h || defaultH, 1);

    if (x + w > columns) {
      x = 0;
      y += rowHeight || h;
      rowHeight = 0;
    }

    const placed = { ...item, w, h, x, y };
    x += w;
    rowHeight = Math.max(rowHeight, h);
    return placed;
  });
}

/**
 * Names a new item uniquely within its list. The counter starts past the
 * current length and steps forward until it finds a free name, so removing an
 * item never causes the next one to collide with a survivor.
 */
export function nextItemId(items: { id: string }[], prefix: string): string {
  let suffix = items.length + 1;
  while (items.some((item) => item.id === `${prefix}${suffix}`)) suffix += 1;
  return `${prefix}${suffix}`;
}
