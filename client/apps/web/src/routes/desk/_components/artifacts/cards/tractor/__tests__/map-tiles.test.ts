import { describe, expect, it } from "vitest";
import { mapTiles, OSM_TILE_URL } from "../map-tiles";

describe("mapTiles", () => {
  it("reads tiles a level deeper and draws them at half size, the point in the middle", () => {
    const tiles = mapTiles({ latitude: 0, longitude: 0, zoom: 1, size: 112 });

    // At level 2 the equator and the prime meridian meet at the corner of four tiles.
    expect(tiles.map((tile) => tile.key)).toEqual(["2/1/1", "2/2/1", "2/1/2", "2/2/2"]);
    expect(tiles.every((tile) => tile.size === 128)).toBe(true);
    expect(tiles[0]).toMatchObject({ left: -72, top: -72 });
    expect(tiles[3]).toMatchObject({ left: 56, top: 56 });
    expect(tiles[0]?.url).toBe("https://tile.openstreetmap.org/2/1/1.png");
  });

  it("covers the whole square and nothing outside it", () => {
    const tiles = mapTiles({ latitude: 32.3932, longitude: -96.8461, zoom: 11, size: 112 });

    expect(tiles.length).toBeGreaterThanOrEqual(1);
    expect(tiles.length).toBeLessThanOrEqual(4);
    for (const tile of tiles) {
      expect(tile.left).toBeLessThan(112);
      expect(tile.top).toBeLessThan(112);
      expect(tile.left + tile.size).toBeGreaterThan(0);
      expect(tile.top + tile.size).toBeGreaterThan(0);
      expect(tile.url).toMatch(/^https:\/\/tile\.openstreetmap\.org\/12\/\d+\/\d+\.png$/u);
    }
    const left = Math.min(...tiles.map((tile) => tile.left));
    const right = Math.max(...tiles.map((tile) => tile.left + tile.size));
    const top = Math.min(...tiles.map((tile) => tile.top));
    const bottom = Math.max(...tiles.map((tile) => tile.top + tile.size));
    expect(left).toBeLessThanOrEqual(0);
    expect(top).toBeLessThanOrEqual(0);
    expect(right).toBeGreaterThanOrEqual(112);
    expect(bottom).toBeGreaterThanOrEqual(112);
  });

  it("places Dallas on the tile OpenStreetMap files it under", () => {
    const tiles = mapTiles({ latitude: 32.7767, longitude: -96.797, zoom: 11, size: 2 });

    expect(tiles.map((tile) => tile.key)).toEqual(["12/946/1652"]);
  });

  it("wraps across the antimeridian rather than asking for a tile that does not exist", () => {
    const tiles = mapTiles({ latitude: 0, longitude: 179.99, zoom: 2, size: 112 });

    const columns = tiles.map((tile) => Number(tile.url.split("/").at(-2)));
    expect(columns).toContain(0);
    expect(columns).toContain(7);
    expect(Math.max(...columns)).toBeLessThan(8);
  });

  it("draws nothing past the poles' edge", () => {
    const tiles = mapTiles({ latitude: 89.9, longitude: 10, zoom: 3, size: 112 });

    for (const tile of tiles) {
      expect(Number(tile.url.split("/").at(-1)?.replace(".png", ""))).toBeGreaterThanOrEqual(0);
    }
    expect(tiles.every((tile) => tile.top >= -tile.size)).toBe(true);
  });

  it("draws nothing for a position that is not a number", () => {
    expect(mapTiles({ latitude: Number.NaN, longitude: 0, zoom: 11, size: 112 })).toEqual([]);
    expect(mapTiles({ latitude: 0, longitude: 0, zoom: 11, size: 0 })).toEqual([]);
  });

  it("fills another tile server's template", () => {
    const tiles = mapTiles({
      latitude: 0,
      longitude: 0,
      zoom: 1,
      size: 2,
      template: "https://tiles.example.com/{z}/{x}/{y}@2x.png",
    });
    expect(tiles[0]?.url).toMatch(/^https:\/\/tiles\.example\.com\/2\/\d\/\d@2x\.png$/u);
    expect(OSM_TILE_URL).toContain("{z}/{x}/{y}");
  });
});
