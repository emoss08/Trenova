import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect } from "react";

export function TrafficLayer() {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");

  useEffect(() => {
    if (!map || !mapsLib) return;

    const layer = new mapsLib.TrafficLayer();
    layer.setMap(map);

    return () => {
      layer.setMap(null);
    };
  }, [map, mapsLib]);

  return null;
}
