import { describe, expect, it } from "vitest";
import {
  mapTiles,
  metersPerPixel,
  OSM_TILE_URL,
  outlineReach,
  projectOnMap,
  zoomToFit,
} from "../map-tiles";

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

describe("metersPerPixel", () => {
  it("reads the ground a pixel covers, halving with each zoom and narrowing toward the poles", () => {
    expect(metersPerPixel(0, 0)).toBeCloseTo(156_543.03, 1);
    expect(metersPerPixel(0, 11)).toBeCloseTo(156_543.03 / 2048, 3);
    expect(metersPerPixel(60, 11)).toBeCloseTo(metersPerPixel(0, 11) / 2, 3);
  });
});

describe("projectOnMap", () => {
  const center = { latitude: 32.7801, longitude: -96.8005 };

  it("puts the centre in the middle and a point east of it to the right, north of it above", () => {
    const [middle, east, north] = projectOnMap(center, 14, 112, [
      center,
      { latitude: center.latitude, longitude: center.longitude + 0.002 },
      { latitude: center.latitude + 0.002, longitude: center.longitude },
    ]);
    expect(middle).toEqual({ x: 56, y: 56 });
    expect(east?.x).toBeGreaterThan(56);
    expect(east?.y).toBeCloseTo(56, 6);
    expect(north?.y).toBeLessThan(56);
    expect(north?.x).toBeCloseTo(56, 6);
  });

  it("places a point across the antimeridian on the near side", () => {
    const [west] = projectOnMap({ latitude: 0, longitude: 179.999 }, 10, 112, [
      { latitude: 0, longitude: -179.999 },
    ]);
    expect(west?.x).toBeGreaterThan(56);
    expect(west?.x).toBeLessThan(80);
  });
});

describe("zoomToFit", () => {
  const center = { latitude: 32.7801, longitude: -96.8005 };

  it("reads as close as the fence allows, keeping it inside the map with room round it", () => {
    const outline = { kind: "circle", radiusMeters: 250 } as const;
    const zoom = zoomToFit({ center, outline, size: 112, min: 11, max: 17 });

    expect(outlineReach(center, zoom, outline) * 2).toBeLessThanOrEqual(112 * 0.72);
    expect(outlineReach(center, zoom + 1, outline) * 2).toBeGreaterThan(112 * 0.72);
  });

  it("measures a drawn fence by its farthest corner", () => {
    const outline = {
      kind: "polygon",
      vertices: [
        { latitude: 32.781, longitude: -96.802 },
        { latitude: 32.781, longitude: -96.799 },
        { latitude: 32.779, longitude: -96.799 },
      ],
    } as const;
    const zoom = zoomToFit({ center, outline, size: 112, min: 11, max: 17 });

    expect(zoom).toBeGreaterThan(11);
    expect(zoom).toBeLessThan(17);
    expect(outlineReach(center, zoom, outline) * 2).toBeLessThanOrEqual(112 * 0.72);
  });

  it("stops at the farthest zoom it is allowed when the fence is wider still", () => {
    const outline = { kind: "circle", radiusMeters: 80_000 } as const;
    expect(zoomToFit({ center, outline, size: 112, min: 11, max: 17 })).toBe(11);
  });
});
