import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { api } from "@/services/api";
import { type LimitOffsetResponse } from "@/types/server";
import type {
  ShipmentDetailsQueryParams,
  ShipmentQueryParams,
} from "@/types/shipment";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useShipments(queryParams: ShipmentQueryParams) {
  return useQuery<LimitOffsetResponse<ShipmentSchema>>({
    queryKey: ["shipments", queryParams],
    queryFn: async () => {
      return await api.shipments.getShipments(queryParams);
    },
    enabled: queryParams.enabled,
  });
}

export function useShipmentDetails({
  shipmentId,
  enabled,
}: ShipmentDetailsQueryParams) {
  return useQuery<ShipmentSchema>({
    queryKey: ["shipment", shipmentId],
    queryFn: async () => {
      return await api.shipments.getShipmentByID(shipmentId, true);
    },
    enabled: enabled,
    placeholderData: keepPreviousData,
  });
}
