import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import {
  mapTiles,
  metersPerPixel,
  OSM_COPYRIGHT_URL,
  projectOnMap,
  type MapOutline,
} from "./map-tiles";

/** The map's side, in CSS pixels; the card's square holds it. */
export const MAP_SIZE = 112;
/** Close enough to read the town and the road, far enough to place them. */
export const MAP_ZOOM = 11;
/** An outline smaller than this many pixels across is left to the dot. */
const MIN_OUTLINE_PX = 6;

/**
 * A place as a small still map of OpenStreetMap's tiles with a dot on the
 * spot. It is a picture, not a map to work in: nothing on it can be dragged,
 * zoomed or clicked but the credit OpenStreetMap asks for. Its own chunk,
 * mounted by `CardWhere` once the card's square comes into view; each tile is
 * fetched by the browser as it is needed.
 */
export function StillMap({
  latitude,
  longitude,
  live,
  label,
  zoom = MAP_ZOOM,
  outline = null,
}: {
  latitude: number;
  longitude: number;
  /** The place is where the record is now, and the dot says so. */
  live: boolean;
  label: string;
  /** How close the map reads; the town by default. */
  zoom?: number;
  /** A fence drawn round the point, such as a location's geofence. */
  outline?: MapOutline | null;
}) {
  const t = useT();
  const tiles = useMemo(
    () => mapTiles({ latitude, longitude, zoom, size: MAP_SIZE }),
    [latitude, longitude, zoom],
  );
  const fence = useMemo(
    () => (outline ? fenceShape({ latitude, longitude }, zoom, outline) : null),
    [latitude, longitude, zoom, outline],
  );
  return (
    <div className={cn("dk-rec-map", live && "dk-live")} role="img" aria-label={label}>
      <div className="dk-rec-map-tiles" aria-hidden>
        {tiles.map((tile) => (
          <img
            key={tile.key}
            src={tile.url}
            alt=""
            width={tile.size}
            height={tile.size}
            loading="lazy"
            decoding="async"
            draggable={false}
            referrerPolicy="strict-origin-when-cross-origin"
            style={{ left: tile.left, top: tile.top }}
          />
        ))}
      </div>
      {fence && (
        <svg
          className="dk-rec-map-fence"
          viewBox={`0 0 ${MAP_SIZE} ${MAP_SIZE}`}
          width={MAP_SIZE}
          height={MAP_SIZE}
          aria-hidden
        >
          {fence.kind === "circle" ? (
            <circle cx={MAP_SIZE / 2} cy={MAP_SIZE / 2} r={fence.radius} />
          ) : (
            <polygon points={fence.points} />
          )}
        </svg>
      )}
      <i className="dk-rec-map-dot" aria-hidden />
      <a
        className="dk-rec-map-credit"
        href={OSM_COPYRIGHT_URL}
        target="_blank"
        rel="noreferrer noopener"
      >
        {t("© OpenStreetMap")}
      </a>
    </div>
  );
}

type FenceShape = { kind: "circle"; radius: number } | { kind: "polygon"; points: string };

/** The outline in the map's pixels, or null when it is too small to see beside the dot. */
function fenceShape(
  center: { latitude: number; longitude: number },
  zoom: number,
  outline: MapOutline,
): FenceShape | null {
  if (outline.kind === "circle") {
    const radius = outline.radiusMeters / metersPerPixel(center.latitude, Math.round(zoom));
    return Number.isFinite(radius) && radius * 2 >= MIN_OUTLINE_PX
      ? { kind: "circle", radius }
      : null;
  }
  if (outline.vertices.length < 3) return null;
  const pixels = projectOnMap(center, zoom, MAP_SIZE, outline.vertices);
  const xs = pixels.map((pixel) => pixel.x);
  const ys = pixels.map((pixel) => pixel.y);
  const span = Math.max(Math.max(...xs) - Math.min(...xs), Math.max(...ys) - Math.min(...ys));
  if (!Number.isFinite(span) || span < MIN_OUTLINE_PX) return null;
  return {
    kind: "polygon",
    points: pixels.map((pixel) => `${pixel.x.toFixed(1)},${pixel.y.toFixed(1)}`).join(" "),
  };
}
