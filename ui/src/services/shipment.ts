import { http } from "@/lib/http-client";
import { LimitOffsetResponse } from "@/types/server";
import { Shipment, type ShipmentQueryParams } from "@/types/shipment";

export async function getShipments(queryParams: ShipmentQueryParams) {
  const response = await http.get<LimitOffsetResponse<Shipment>>(
    "/shipments/",
    {
      params: {
        limit: queryParams.pageSize.toString(),
        offset: (queryParams.pageIndex * queryParams.pageSize).toString(),
        expandShipmentDetails: queryParams.expandShipmentDetails.toString(),
        query: queryParams.query ?? "",
      },
    },
  );

  return response.data;
}

export async function getShipmentByID(
  shipmentId: string,
  expandShipmentDetails = false,
) {
  const response = await http.get<Shipment>(`/shipments/${shipmentId}`, {
    params: {
      expandShipmentDetails: expandShipmentDetails.toString(),
    },
  });
  return response.data;
}

export async function checkForDuplicateBOLs(bol: string, shipmentId?: string) {
  const response = await http.post<{ valid: boolean }>(
    "/shipments/check-for-duplicate-bols/",
    {
      bol,
      shipmentId,
    },
  );
  return response.data;
}
