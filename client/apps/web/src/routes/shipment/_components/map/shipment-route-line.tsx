import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect } from "react";
import type { PinTone } from "./shipment-pin";

const TONE_HEX_FALLBACK: Record<PinTone, string> = {
  brand: "#3060f4",
  destructive: "#dc2f2a",
  success: "#10b981",
  warning: "#eab308",
  muted: "#7a7a7a",
};

function resolveCssVar(name: string, fallback: string): string {
  if (typeof window === "undefined") return fallback;
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || fallback;
}

type Props = {
  origin: google.maps.LatLngLiteral;
  destination: google.maps.LatLngLiteral;
  tone: PinTone;
  highlighted?: boolean;
  dimmed?: boolean;
  dashed?: boolean;
};

export function ShipmentRouteLine({
  origin,
  destination,
  tone,
  highlighted,
  dimmed,
  dashed,
}: Props) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");

  useEffect(() => {
    if (!map || !mapsLib) return;

    const cssVarName = `--${tone === "destructive" ? "destructive" : tone === "muted" ? "muted-foreground" : tone}`;
    const color = resolveCssVar(cssVarName, TONE_HEX_FALLBACK[tone]);

    const baseOpacity = highlighted ? 0.95 : 0.75;
    const opacity = dimmed ? 0.18 : baseOpacity;
    const weight = highlighted ? 4.5 : 2.5;

    const dashSymbol: google.maps.Symbol = {
      path: "M 0,-1 0,1",
      strokeColor: color,
      strokeOpacity: opacity,
      strokeWeight: weight,
      scale: 3,
    };

    const line = new mapsLib.Polyline({
      path: [origin, destination],
      map,
      geodesic: true,
      strokeColor: color,
      strokeOpacity: dashed ? 0 : opacity,
      strokeWeight: weight,
      icons: dashed ? [{ icon: dashSymbol, offset: "0", repeat: "10px" }] : undefined,
      zIndex: highlighted ? 90 : 5,
      clickable: false,
    });

    return () => {
      line.setMap(null);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    map,
    mapsLib,
    origin.lat,
    origin.lng,
    destination.lat,
    destination.lng,
    tone,
    highlighted,
    dimmed,
    dashed,
  ]);

  return null;
}
