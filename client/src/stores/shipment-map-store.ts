import { createSelectors } from "@/lib/utils";
import type { RainViewerData } from "@/types/shipment-map";
import type { SetStateAction } from "react";
import { create } from "zustand";
import { createJSONStorage, devtools, persist } from "zustand/middleware";

interface ShipmentMapState {
  data: RainViewerData | null;
  setData: (data: RainViewerData) => void;
  currentIndex: number;
  setCurrentIndex: (index: SetStateAction<number>) => void;
  weatherLayerOpen: boolean;
  setWeatherLayerOpen: (open: boolean) => void;
  isPlaying: boolean;
  setIsPlaying: (playing: SetStateAction<boolean>) => void;
  isFullscreen: boolean;
  setIsFullscreen: (fullscreen: SetStateAction<boolean>) => void;
}

const baseStore = create<ShipmentMapState>()(
  devtools(
    persist(
      (set) => ({
        data: null,
        currentIndex: 0,
        weatherLayerOpen: false,
        isPlaying: false,
        isFullscreen: false,

        setData: (data: RainViewerData) => {
          set({ data });
        },
        setCurrentIndex: (index: SetStateAction<number>) =>
          set((state) => ({
            currentIndex: typeof index === "function" ? index(state.currentIndex) : index,
          })),
        setWeatherLayerOpen: (open: boolean) => set({ weatherLayerOpen: open }),
        setIsPlaying: (playing: SetStateAction<boolean>) =>
          set((state) => ({
            isPlaying: typeof playing === "function" ? playing(state.isPlaying) : playing,
          })),
        setIsFullscreen: (fullscreen: SetStateAction<boolean>) =>
          set((state) => ({
            isFullscreen:
              typeof fullscreen === "function" ? fullscreen(state.isFullscreen) : fullscreen,
          })),
      }),
      { name: "shipment-map-storage", storage: createJSONStorage(() => localStorage) },
    ),
  ),
);

export const useShipmentMapStore = createSelectors(baseStore);
