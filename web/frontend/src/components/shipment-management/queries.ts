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
