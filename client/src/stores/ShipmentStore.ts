/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import { Shipment } from "@/types/order";
import { Worker } from "@/types/worker";
import { SetState, StateCreator, create } from "zustand";
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
    name: "monta-shipment-map-preferences",
  }) as StateCreator<ShipmentMapStore>,
);
