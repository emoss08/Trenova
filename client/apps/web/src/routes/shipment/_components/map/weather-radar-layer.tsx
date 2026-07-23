import { queries } from "@/lib/queries";
import { useShipmentMapStore } from "@/stores/shipment-map-store";
import type { WeatherLayerId } from "@/types/shipment-map";
import { useQuery } from "@tanstack/react-query";
import { useMap } from "@vis.gl/react-google-maps";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { WeatherTimeline } from "./weather-timeline";

const REFRESH_INTERVAL = 10 * 60 * 1000;
const PLAYBACK_INTERVAL = 500;

function useWeatherRadar() {
  const currentIndex = useShipmentMapStore.use.currentIndex();
  const setCurrentIndex = useShipmentMapStore.use.setCurrentIndex();
  const isPlaying = useShipmentMapStore.use.isPlaying();
  const setIsPlaying = useShipmentMapStore.use.setIsPlaying();
  const hasInitialized = useRef(false);

  const { data } = useQuery({
    ...queries.weatherRadar.weatherMaps(),
    refetchInterval: REFRESH_INTERVAL,
    retry: false,
  });

  const frames = useMemo(() => data?.radar.past ?? [], [data?.radar.past]);
  const host = data?.host ?? "";

  useEffect(() => {
    if (!hasInitialized.current && frames.length > 0) {
      setCurrentIndex(frames.length - 1);
      hasInitialized.current = true;
    }
  }, [frames, setCurrentIndex]);

  useEffect(() => {
    if (!isPlaying || frames.length === 0) return;

    const interval = setInterval(() => {
      setCurrentIndex((prev) => {
        const next = prev + 1;
        return next >= frames.length ? 0 : next;
      });
    }, PLAYBACK_INTERVAL);

    return () => clearInterval(interval);
  }, [isPlaying, frames.length, setCurrentIndex]);

  const togglePlay = useCallback(() => {
    setIsPlaying((p) => !p);
  }, [setIsPlaying]);

  return {
    frames,
    host,
    currentIndex,
    setCurrentIndex,
    isPlaying,
    togglePlay,
  };
}

export function WeatherRadarLayer({
  weatherLayer,
  onWeatherLayerChange,
}: {
  weatherLayer: WeatherLayerId;
  onWeatherLayerChange: (layer: WeatherLayerId) => void;
}) {
  const map = useMap();
  const { frames, host, currentIndex, setCurrentIndex, isPlaying, togglePlay } = useWeatherRadar();
  const overlayRef = useRef<google.maps.ImageMapType | null>(null);

  useEffect(() => {
    if (!map) return;

    if (overlayRef.current) {
      const types = map.overlayMapTypes;
      for (let i = 0; i < types.getLength(); i++) {
        if (types.getAt(i) === overlayRef.current) {
          types.removeAt(i);
          break;
        }
      }
      overlayRef.current = null;
    }

    if (frames.length === 0 || !host || weatherLayer !== "precipitation") return;

    const frame = frames[currentIndex];
    const overlay = new google.maps.ImageMapType({
      getTileUrl: (coord, zoom) =>
        `${host}${frame.path}/256/${zoom}/${coord.x}/${coord.y}/2/1_1.png`,
      tileSize: new google.maps.Size(256, 256),
      opacity: 0.7,
    });

    map.overlayMapTypes.insertAt(0, overlay);
    overlayRef.current = overlay;
  }, [map, frames, host, currentIndex, weatherLayer]);

  useEffect(() => {
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
  }, [map]);

  if (frames.length === 0) return null;

  return (
    <WeatherTimeline
      frames={frames}
      currentIndex={currentIndex}
      onIndexChange={setCurrentIndex}
      isPlaying={isPlaying}
      onTogglePlay={togglePlay}
      weatherLayer={weatherLayer}
      onWeatherLayerChange={onWeatherLayerChange}
    />
  );
}
