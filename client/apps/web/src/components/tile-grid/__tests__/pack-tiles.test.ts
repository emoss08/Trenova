import { describe, expect, it } from "vitest";
import { nextItemId, packTiles, type GridItem } from "../pack-tiles";

function item(id: string, w: number, h: number): GridItem {
  return { id, w, h };
}

describe("packTiles", () => {
  it("lays a row out left to right", () => {
    const packed = packTiles([item("a", 4, 2), item("b", 4, 2), item("c", 4, 2)], { columns: 12 });

    expect(packed.map((t) => [t.x, t.y])).toEqual([
      [0, 0],
      [4, 0],
      [8, 0],
    ]);
  });

  it("wraps to the next row when an item will not fit", () => {
    const packed = packTiles([item("a", 8, 3), item("b", 6, 2)], { columns: 12 });

    expect(packed[0]).toMatchObject({ x: 0, y: 0 });
    expect(packed[1]).toMatchObject({ x: 0, y: 3 });
  });

  it("advances the row by the tallest item in it", () => {
    const packed = packTiles([item("a", 6, 5), item("b", 6, 2), item("c", 12, 2)], { columns: 12 });

    expect(packed[2]).toMatchObject({ x: 0, y: 5 });
  });

  it("clamps a width wider than the grid", () => {
    const packed = packTiles([item("a", 99, 2)], { columns: 12 });

    expect(packed[0].w).toBe(12);
    expect(packed[0].x).toBe(0);
  });

  it("substitutes defaults for a zero span", () => {
    const packed = packTiles([{ id: "a", w: 0, h: 0 }], {
      columns: 12,
      defaultW: 5,
      defaultH: 3,
    });

    expect(packed[0]).toMatchObject({ w: 5, h: 3 });
  });

  it("keeps every other field on the item", () => {
    const packed = packTiles([{ id: "a", w: 4, h: 2, kind: "chart" }], { columns: 12 });

    expect(packed[0].kind).toBe("chart");
  });

  it("never places an item past the last column", () => {
    const packed = packTiles([item("a", 5, 2), item("b", 5, 2), item("c", 5, 2), item("d", 5, 2)], {
      columns: 12,
    });

    for (const tile of packed) {
      expect(tile.x + tile.w).toBeLessThanOrEqual(12);
    }
  });
});

describe("nextItemId", () => {
  it("names the first item from the prefix", () => {
    expect(nextItemId([], "tile_")).toBe("tile_1");
  });

  it("continues past the current length", () => {
    expect(nextItemId([item("tile_1", 4, 2), item("tile_2", 4, 2)], "tile_")).toBe("tile_3");
  });

  // Removing an item shortens the list, so the length-based guess can collide
  // with a survivor; the counter has to step past it.
  it("skips a name a survivor already holds", () => {
    expect(nextItemId([item("tile_1", 4, 2), item("tile_3", 4, 2)], "tile_")).toBe("tile_4");
  });
});
