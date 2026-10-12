/** OpenStreetMap's standard tiles. Their use needs the attribution the map draws. */
export const OSM_TILE_URL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
export const OSM_COPYRIGHT_URL = "https://www.openstreetmap.org/copyright";

/** The Web Mercator edge: no tile is drawn past it. */
const MAX_LATITUDE = 85.051_128_78;
const TILE_SIZE = 256;

/** One tile of a small still map, placed in the map's own box. */
export type MapTile = { key: string; url: string; left: number; top: number; size: number };

/**
 * The tiles that cover a square map of `size` CSS pixels centred on a point,
 * each placed so the point falls in the middle. Tiles are read one zoom level
 * deeper than they are drawn and shown at half their size, so a map on a
 * high-density screen stays sharp. The map wraps round the antimeridian and
 * stops at the poles' edge, where there is nothing to draw.
 */
export function mapTiles({
  latitude,
  longitude,
  zoom,
  size,
  template = OSM_TILE_URL,
}: {
  latitude: number;
  longitude: number;
  /** The zoom the map reads as, 0 (the world) to 18 (a street). */
  zoom: number;
  size: number;
  template?: string;
}): MapTile[] {
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude) || size <= 0) return [];
  const level = Math.min(19, Math.max(1, Math.round(zoom) + 1));
  const drawn = TILE_SIZE / 2;
  const count = 2 ** level;
  const lat = Math.max(-MAX_LATITUDE, Math.min(MAX_LATITUDE, latitude));
  const sin = Math.sin((lat * Math.PI) / 180);
  const worldX = ((((longitude + 180) / 360) % 1) + 1) % 1;
  const worldY = 0.5 - Math.log((1 + sin) / (1 - sin)) / (4 * Math.PI);
  const pointX = worldX * count * drawn;
  const pointY = worldY * count * drawn;
  const originX = pointX - size / 2;
  const originY = pointY - size / 2;

  const tiles: MapTile[] = [];
  for (
    let row = Math.floor(originY / drawn);
    row <= Math.floor((originY + size - 1) / drawn);
    row++
  ) {
    if (row < 0 || row >= count) continue;
    for (
      let column = Math.floor(originX / drawn);
      column <= Math.floor((originX + size - 1) / drawn);
      column++
    ) {
      const x = ((column % count) + count) % count;
      tiles.push({
        key: `${level}/${column}/${row}`,
        url: template
          .replace("{z}", String(level))
          .replace("{x}", String(x))
          .replace("{y}", String(row)),
        left: Math.round(column * drawn - originX),
        top: Math.round(row * drawn - originY),
        size: drawn,
      });
    }
  }
  return tiles;
}
