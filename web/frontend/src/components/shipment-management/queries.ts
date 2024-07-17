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

import { getTractors } from "@/services/EquipmentRequestService";
import { getShipments } from "@/services/ShipmentRequestService";
import { EquipmentStatus } from "@/types/equipment";
import { useQuery } from "@tanstack/react-query";

export function useShipmentData(
  searchQuery: string,
  statusFilter: string,
  shipmentPage: number,
  itemsPerPage: number,
) {
  const {
    data: shipmentData,
    isLoading: isShipmentLoading,
    isError: isShipmentError,
  } = useQuery({
    queryKey: [
      "shipments",
      searchQuery,
      statusFilter,
      shipmentPage,
      itemsPerPage,
    ],
    queryFn: () =>
      getShipments(
        searchQuery,
        statusFilter,
        (shipmentPage - 1) * itemsPerPage,
        itemsPerPage,
      ),
    staleTime: Infinity,
  });

  return {
    shipmentData,
    isShipmentLoading,
    isShipmentError,
  };
}

export function useTractorsData(
  tractorSearchQuery: string,
  tractorStatusFilter: EquipmentStatus,
  tractorPage: number,
  itemsPerPage: number,
  fleetCodeId?: string,
) {
  const {
    data: tractorData,
    isLoading: isTractorLoading,
    isError: isTractorError,
  } = useQuery({
    queryKey: [
      "tractors",
      tractorSearchQuery,
      tractorStatusFilter,
      fleetCodeId,
      tractorPage,
      itemsPerPage,
    ],
    queryFn: () =>
      getTractors(
        tractorStatusFilter,
        (tractorPage - 1) * itemsPerPage,
        itemsPerPage,
        fleetCodeId,
      ),
    staleTime: Infinity,
  });

  return {
    tractorData,
    isTractorLoading,
    isTractorError,
  };
}
