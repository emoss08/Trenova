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
