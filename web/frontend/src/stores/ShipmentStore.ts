/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
