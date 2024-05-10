import { createGlobalStore } from "@/lib/useGlobalStore";
import { type Shipment } from "@/types/shipment";
import { type Worker } from "@/types/worker";
import { create, type SetState, type StateCreator } from "zustand";
import { persist } from "zustand/middleware";

export type ShipmentView = "list" | "calendar" | "map";

type ShipmentStore = {
  currentShipment?: Shipment;
  currentView: ShipmentView;
  sendMessageDialogOpen: boolean;
  reviewLogDialogOpen: boolean;
  currentWorker?: Worker;
};

export const useShipmentStore = createGlobalStore<ShipmentStore>({
  currentShipment: undefined,
  currentView: "map",
  sendMessageDialogOpen: false,
  reviewLogDialogOpen: false,
  currentWorker: undefined,
});

export type MapType = "roadmap" | "hybrid" | "terrain";

export type MapLayer = "TrafficLayer" | "WeatherLayer" | "TransitLayer";

type ShipmentMapStore = {
  mapType: MapType;
  setMapType: (mapType: MapType) => void;
  mapLayers: MapLayer[];
  setMapLayers: (mapLayers: MapLayer[]) => void;
};

const createShipmentStore = (set: SetState<ShipmentMapStore>) => ({
  mapType: "roadmap",
  setMapType: (mapType: MapType) => set({ mapType }),
  mapLayers: [] as MapLayer[],
  setMapLayers: (mapLayers: MapLayer[]) => set({ mapLayers }),
});

export const useShipmentMapStore = create<ShipmentMapStore>(
  persist(createShipmentStore, {
    name: "trenova-shipment-map-preferences",
  }) as StateCreator<ShipmentMapStore>,
);
