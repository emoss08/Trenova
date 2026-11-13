import { describe, expect, it } from "vitest";
import { narrowTileSpan } from "../sortable-tile";

describe("narrowTileSpan", () => {
  it("gives a tile at most half the grid wide half of it, so two still sit side by side", () => {
    expect([2, 3, 4, 6].map((w) => narrowTileSpan(w, 12))).toEqual([6, 6, 6, 6]);
  });

  it("gives a tile wider than half the whole row", () => {
    expect([7, 8, 9, 12].map((w) => narrowTileSpan(w, 12))).toEqual([12, 12, 12, 12]);
  });

  it("follows the grid it is in rather than assuming twelve columns", () => {
    expect(narrowTileSpan(2, 8)).toBe(4);
    expect(narrowTileSpan(5, 8)).toBe(8);
  });
});
