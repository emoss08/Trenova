import { createGlobalStore } from "@/lib/useGlobalStore";
import { type Tractor } from "@/types/equipment";
import { type Shipment } from "@/types/shipment";
import { create, type SetState, type StateCreator } from "zustand";
import { persist } from "zustand/middleware";

type ShipmentStore = {
  currentShipment?: Shipment;
  sendMessageDialogOpen: boolean;
  reviewLogDialogOpen: boolean;
  currentTractor?: Tractor;
  addShipmentDialogOpen: boolean;
};

export const useShipmentStore = createGlobalStore<ShipmentStore>({
  currentShipment: undefined,
  sendMessageDialogOpen: false,
  reviewLogDialogOpen: false,
  currentTractor: undefined,
  addShipmentDialogOpen: false,
});

export type MapType = "roadmap" | "hybrid" | "terrain";

export type MapLayer = "TrafficLayer" | "WeatherLayer" | "TransitLayer";

type ShipmentListStore = {
  mapType: MapType;
  setMapType: (mapType: MapType) => void;
  mapLayers: MapLayer[];
  setMapLayers: (mapLayers: MapLayer[]) => void;
};

const createShipmentStore = (set: SetState<ShipmentListStore>) => ({
  mapType: "roadmap",
  setMapType: (mapType: MapType) => set({ mapType }),
  mapLayers: [] as MapLayer[],
  setMapLayers: (mapLayers: MapLayer[]) => set({ mapLayers }),
});

export const useShipmentListStore = create<ShipmentListStore>(
  persist(createShipmentStore, {
    name: "trenova-shipment-map-preferences",
  }) as StateCreator<ShipmentListStore>,
);
