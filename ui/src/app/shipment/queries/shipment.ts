import { getShipmentByID, getShipments } from "@/services/shipment";
import { type LimitOffsetResponse } from "@/types/server";
import type {
  ShipmentDetailsQueryParams,
  ShipmentQueryParams,
  Shipment as ShipmentResponse,
} from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";

export function useShipments(queryParams: ShipmentQueryParams) {
  return useQuery<LimitOffsetResponse<ShipmentResponse>>({
    queryKey: ["shipments", queryParams],
    queryFn: async () => {
      return await getShipments(queryParams);
    },
    enabled: queryParams.enabled,
  });
}

export function useShipmentDetails({
  shipmentId,
  enabled,
}: ShipmentDetailsQueryParams) {
  return useQuery<ShipmentResponse>({
    queryKey: ["shipment", shipmentId],
    queryFn: async () => {
      return await getShipmentByID(shipmentId, true);
    },
    enabled: enabled,
  });
}
