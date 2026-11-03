import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect, useRef } from "react";

/* The Maps API rasterises a literal colour, so a token is read off the document
   when the line is styled; this grey stands in only if the token is missing. */
const FALLBACK_COLOR = "#7a7a7a"; // design-tokens-ignore: Maps API needs a literal

function resolveCssVar(name: string): string {
  if (typeof window === "undefined") return FALLBACK_COLOR;
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || FALLBACK_COLOR;
}

type MapPolylineProps = {
  path: google.maps.LatLngLiteral[];
  /** A colour token, such as `--danger`. */
  colorVar: string;
  weight: number;
  opacity: number;
  dashed?: boolean;
  geodesic?: boolean;
  zIndex?: number;
  onClick?: () => void;
};

/**
 * A line on the map. The line is created once per path and restyled in place,
 * so hovering and focusing never tear it down and redraw it.
 */
export function MapPolyline({
  path,
  colorVar,
  weight,
  opacity,
  dashed = false,
  geodesic = true,
  zIndex = 5,
  onClick,
}: MapPolylineProps) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");
  const lineRef = useRef<google.maps.Polyline | null>(null);
  const clickable = !!onClick;
  const click = useLatestCallback(() => onClick?.());
  const pathKey = path.map((p) => [p.lat.toFixed(6), p.lng.toFixed(6)].join(",")).join(";");

  useEffect(() => {
    if (!map || !mapsLib) return;
    const line = new mapsLib.Polyline({ path, map, geodesic });
    const listener = line.addListener("click", () => click());
    lineRef.current = line;
    return () => {
      listener.remove();
      line.setMap(null);
      lineRef.current = null;
    };
    // The path is keyed by its coordinates, not by the array's identity.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [map, mapsLib, pathKey, geodesic, click]);

  useEffect(() => {
    const line = lineRef.current;
    if (!line) return;
    const color = resolveCssVar(colorVar);
    line.setOptions({
      strokeColor: color,
      strokeOpacity: dashed ? 0 : opacity,
      strokeWeight: weight,
      icons: dashed
        ? [
            {
              icon: {
                path: "M 0,-1 0,1",
                strokeColor: color,
                strokeOpacity: opacity,
                strokeWeight: weight,
                scale: 2,
              },
              offset: "0",
              repeat: "9px",
            },
          ]
        : [],
      zIndex,
      clickable,
    });
  }, [map, mapsLib, pathKey, colorVar, weight, opacity, dashed, zIndex, clickable]);

  return null;
}
