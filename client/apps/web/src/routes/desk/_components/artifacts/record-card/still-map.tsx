import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { mapTiles, OSM_COPYRIGHT_URL } from "./map-tiles";

/** The map's side, in CSS pixels; the card's square holds it. */
export const MAP_SIZE = 112;
/** Close enough to read the town and the road, far enough to place them. */
const ZOOM = 11;

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
}: {
  latitude: number;
  longitude: number;
  /** The place is where the record is now, and the dot says so. */
  live: boolean;
  label: string;
}) {
  const t = useT();
  const tiles = useMemo(
    () => mapTiles({ latitude, longitude, zoom: ZOOM, size: MAP_SIZE }),
    [latitude, longitude],
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
