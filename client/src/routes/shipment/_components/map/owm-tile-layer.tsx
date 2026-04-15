import { useMap } from "@vis.gl/react-google-maps";
import { useEffect, useRef } from "react";

export type OWMLayerId = "wind_new" | "clouds_new" | "temp_new" | "pressure_new" | "precipitation_new";

export function OWMTileLayer({
  layerId,
  apiKey,
}: {
  layerId: OWMLayerId;
  apiKey: string;
}) {
  const map = useMap();
  const overlayRef = useRef<google.maps.ImageMapType | null>(null);

  useEffect(() => {
    if (!map || !apiKey) return;

    const overlay = new google.maps.ImageMapType({
      getTileUrl: (coord, zoom) =>
        `https://tile.openweathermap.org/map/${layerId}/${zoom}/${coord.x}/${coord.y}.png?appid=${apiKey}`,
      tileSize: new google.maps.Size(256, 256),
      opacity: 0.7,
    });

    map.overlayMapTypes.push(overlay);
    overlayRef.current = overlay;

    return () => {
      if (overlayRef.current && map) {
        const types = map.overlayMapTypes;
        for (let i = 0; i < types.getLength(); i++) {
          if (types.getAt(i) === overlayRef.current) {
            types.removeAt(i);
            break;
          }
        }
        overlayRef.current = null;
      }
    };
  }, [map, apiKey, layerId]);

  return null;
}
