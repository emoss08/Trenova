import { http } from "@/lib/http-client";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { LimitOffsetResponse } from "@/types/server";
import { type Shipment, type ShipmentQueryParams } from "@/types/shipment";

export async function getShipments(queryParams: ShipmentQueryParams) {
  const response = await http.get<LimitOffsetResponse<Shipment>>(
    "/shipments/",
    {
      params: {
        limit: queryParams.pageSize?.toString(),
        offset: (
          (queryParams?.pageIndex ?? 0) * (queryParams?.pageSize ?? 10)
        ).toString(),
        expandShipmentDetails: queryParams.expandShipmentDetails?.toString(),
        query: queryParams.query ?? "",
        status: queryParams.status,
      },
    },
  );

  return response.data;
}

export async function getShipmentByID(
  shipmentId: ShipmentSchema["id"],
  expandShipmentDetails = false,
) {
  const response = await http.get<Shipment>(`/shipments/${shipmentId}`, {
    params: {
      expandShipmentDetails: expandShipmentDetails.toString(),
    },
  });
  return response.data;
}

export async function checkForDuplicateBOLs(
  bol: ShipmentSchema["bol"],
  shipmentId?: ShipmentSchema["id"],
) {
  const response = await http.post<{ valid: boolean }>(
    "/shipments/check-for-duplicate-bols/",
    {
      bol,
      shipmentId,
    },
  );
  return response.data;
}

export async function markReadyToBill(shipmentId: ShipmentSchema["id"]) {
  if (!shipmentId) {
    throw new Error(
      "Shipment ID is required to mark a shipment as ready to bill",
    );
  }
  const response = await http.put<ShipmentSchema>(
    `/shipments/${shipmentId}/mark-ready-to-bill/`,
    {},
  );
  return response.data;
}
