/** OpenStreetMap's standard tiles. Their use needs the attribution the map draws. */
export const OSM_TILE_URL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
export const OSM_COPYRIGHT_URL = "https://www.openstreetmap.org/copyright";

/** The Web Mercator edge: no tile is drawn past it. */
const MAX_LATITUDE = 85.051_128_78;
const TILE_SIZE = 256;

/** The Earth's circumference at the equator, in metres, as Web Mercator reads it. */
const EARTH_CIRCUMFERENCE = 40_075_016.686;

/** Where a point falls on the Web Mercator world, 0 to 1 across and down. */
function worldPoint(latitude: number, longitude: number): { x: number; y: number } {
  const lat = Math.max(-MAX_LATITUDE, Math.min(MAX_LATITUDE, latitude));
  const sin = Math.sin((lat * Math.PI) / 180);
  return {
    x: ((((longitude + 180) / 360) % 1) + 1) % 1,
    y: 0.5 - Math.log((1 + sin) / (1 - sin)) / (4 * Math.PI),
  };
}

/** The ground under one CSS pixel of a still map read at `zoom`, in metres, at a latitude. */
export function metersPerPixel(latitude: number, zoom: number): number {
  const lat = Math.max(-MAX_LATITUDE, Math.min(MAX_LATITUDE, latitude));
  return (EARTH_CIRCUMFERENCE * Math.cos((lat * Math.PI) / 180)) / (TILE_SIZE * 2 ** zoom);
}

/** A point on a still map, in CSS pixels from its top left corner. */
export type MapPixel = { x: number; y: number };

/**
 * Where points fall on a square map of `size` CSS pixels centred on a point and
 * read at `zoom`, the same way `mapTiles` lays its tiles, so an outline drawn
 * over them sits on the ground it marks. A point across the antimeridian from
 * the centre is placed on the near side.
 */
export function projectOnMap(
  center: { latitude: number; longitude: number },
  zoom: number,
  size: number,
  points: readonly { latitude: number; longitude: number }[],
): MapPixel[] {
  const scale = TILE_SIZE * 2 ** Math.round(zoom);
  const middle = worldPoint(center.latitude, center.longitude);
  return points.map((point) => {
    const at = worldPoint(point.latitude, point.longitude);
    const across = ((at.x - middle.x + 1.5) % 1) - 0.5;
    return { x: size / 2 + across * scale, y: size / 2 + (at.y - middle.y) * scale };
  });
}

/** What a still map can outline round its point: a location's geofence. */
export type MapOutline =
  | { kind: "circle"; radiusMeters: number }
  | { kind: "polygon"; vertices: readonly { latitude: number; longitude: number }[] };

/** How far the outline reaches from the centre at `zoom`, in CSS pixels. */
export function outlineReach(
  center: { latitude: number; longitude: number },
  zoom: number,
  outline: MapOutline,
): number {
  if (outline.kind === "circle") {
    return outline.radiusMeters / metersPerPixel(center.latitude, Math.round(zoom));
  }
  return projectOnMap(center, zoom, 0, outline.vertices).reduce(
    (reach, pixel) => Math.max(reach, Math.abs(pixel.x), Math.abs(pixel.y)),
    0,
  );
}

/**
 * The closest zoom, from `max` down to `min`, at which an outline round the
 * centre stays inside `share` of the map, so the whole fence shows with room
 * round it; `min` when even that is too close.
 */
export function zoomToFit({
  center,
  outline,
  size,
  min,
  max,
  share = 0.72,
}: {
  center: { latitude: number; longitude: number };
  outline: MapOutline;
  size: number;
  min: number;
  max: number;
  share?: number;
}): number {
  for (let zoom = max; zoom > min; zoom--) {
    if (outlineReach(center, zoom, outline) * 2 <= size * share) return zoom;
  }
  return min;
}

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
  const world = worldPoint(latitude, longitude);
  const pointX = world.x * count * drawn;
  const pointY = world.y * count * drawn;
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
