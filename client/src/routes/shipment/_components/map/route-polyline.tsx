import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect } from "react";

const BLUE = "#3b82f6";

export function RoutePolyline({
  path,
}: {
  path: { lat: number; lng: number }[];
}) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");

  useEffect(() => {
    if (!map || !mapsLib || path.length === 0) return;

    const polyline = new mapsLib.Polyline({
      path,
      map,
      strokeColor: BLUE,
      strokeWeight: 3,
      strokeOpacity: 0,
      icons: [
        {
          icon: {
            path: "M 0,-1 0,1",
            strokeOpacity: 0.8,
            strokeColor: BLUE,
            strokeWeight: 3,
            scale: 3,
          },
          offset: "0",
          repeat: "16px",
        },
      ],
    });

    return () => {
      polyline.setMap(null);
    };
  }, [map, mapsLib, path]);

  return null;
}
