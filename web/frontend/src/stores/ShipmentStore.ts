/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



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
