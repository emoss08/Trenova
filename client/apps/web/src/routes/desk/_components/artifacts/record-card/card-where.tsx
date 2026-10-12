import { useInView } from "@/hooks/use-in-view";
import { describeApiError } from "@/lib/api-error-message";
import { lazyComponent } from "@/lib/lazy-component";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Suspense, type ReactNode } from "react";
import type { MapOutline } from "./map-tiles";
import { CardFacts, CardIcon, CardSectionError, CardSectionWait } from "./record-card";

/** How far below the screen the map starts reading, so it is drawn as it scrolls in. */
const MAP_READ_AHEAD = "240px 0px";

const StillMap = lazyComponent(
  () => import("./still-map"),
  (module) => module.StillMap,
);

/** A place on the map. */
export type MapPoint = { latitude: number; longitude: number };

/** A point when both of its coordinates are numbers, or null. */
export function mapPoint(
  latitude: number | null | undefined,
  longitude: number | null | undefined,
): MapPoint | null {
  return typeof latitude === "number" &&
    typeof longitude === "number" &&
    Number.isFinite(latitude) &&
    Number.isFinite(longitude)
    ? { latitude, longitude }
    : null;
}

/**
 * Where a record is: a small still map of the place with a dot on it, beside
 * the place's name, how the record stands there and what else is known about
 * it, with whatever a card lists under it (the location a move left it at).
 * The map's code and tiles are read only once its square comes within reach
 * of the screen; a record with no place shows a bare pin in the same square,
 * so nothing moves when the place is known.
 */
export function CardWhere({
  point,
  live = false,
  pending = false,
  place,
  placeMissing,
  line,
  lineOff = false,
  facts,
  mapLabel,
  noMapLabel,
  error,
  errorText,
  zoom,
  outline,
  children,
}: {
  /** Where to put the dot; null when the place has no coordinates. */
  point: MapPoint | null;
  /** How close the map reads; the town when not given. */
  zoom?: number;
  /** A fence drawn round the dot, such as a location's geofence. */
  outline?: MapOutline | null;
  /** The place is where the record is now, not where it was. */
  live?: boolean;
  /** The place is still being read. */
  pending?: boolean;
  /** The place's name, or null when there is none. */
  place: string | null;
  /** What stands for the place when it has no name. */
  placeMissing: string;
  /** How the record stands there: "Moving 61 mph S", "Placed by its last move". */
  line: string;
  /** The line says nothing is known, and reads muted. */
  lineOff?: boolean;
  facts?: readonly string[];
  mapLabel: string;
  noMapLabel: string;
  error?: unknown;
  /** What the section says when the place could not be read. */
  errorText?: string;
  children?: ReactNode;
}) {
  const t = useT();
  const [square, near] = useInView<HTMLDivElement>({ rootMargin: MAP_READ_AHEAD, once: true });
  const shown = facts?.filter(Boolean) ?? [];

  return (
    <>
      <div className={cn("dk-rec-where", live && "dk-live")}>
        <div ref={square} className="dk-rec-scope">
          {point && near ? (
            <Suspense fallback={null}>
              <StillMap.Component
                latitude={point.latitude}
                longitude={point.longitude}
                live={live}
                label={mapLabel}
                zoom={zoom}
                outline={outline}
              />
            </Suspense>
          ) : (
            !point &&
            !pending && (
              <span className="dk-rec-nomap" role="img" aria-label={noMapLabel}>
                <CardIcon name="pin" size={18} stroke={1.6} />
              </span>
            )
          )}
        </div>
        <div className="dk-rec-at">
          {pending ? (
            <CardSectionWait rows={2} />
          ) : (
            <>
              <b className={cn(!place && "dk-rec-unknown")}>{place ?? placeMissing}</b>
              <span className={cn("dk-rec-motion", lineOff && "dk-off")}>
                <i aria-hidden />
                {line}
              </span>
              {shown.length > 0 && (
                <span className="dk-rec-geo">
                  <CardFacts>{shown}</CardFacts>
                </span>
              )}
            </>
          )}
        </div>
      </div>
      {error !== null && error !== undefined && (
        <CardSectionError>
          {describeApiError(error, errorText ?? t("The position could not be read."))}
        </CardSectionError>
      )}
      {children}
    </>
  );
}
