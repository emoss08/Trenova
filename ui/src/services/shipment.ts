import { http } from "@/lib/http-client";
import type { ShipmentDuplicateSchema } from "@/lib/schemas/shipment-duplicate-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { LimitOffsetResponse } from "@/types/server";
import { type Shipment, type ShipmentQueryParams } from "@/types/shipment";

export class ShipmentAPI {
  // Get shipments from the API
  async getShipments(queryParams: ShipmentQueryParams) {
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

  // Get a shipment by ID from the API
  async getShipmentByID(
    shipmentId: Shipment["id"],
    expandShipmentDetails = false,
  ) {
    const response = await http.get<Shipment>(`/shipments/${shipmentId}/`, {
      params: {
        expandShipmentDetails: expandShipmentDetails.toString(),
      },
    });

    return response.data;
  }

  async create(shipment: ShipmentSchema) {
    return await http.post<ShipmentSchema>("/shipments/", shipment);
  }

  // Check for duplicate BOLs
  async checkForDuplicateBOLs(
    bol: Shipment["bol"],
    shipmentId?: Shipment["id"],
  ) {
    const response = await http.post<{ valid: boolean }>(
      `/shipments/check-for-duplicate-bols/`,
      {
        bol,
        shipmentId,
      },
    );

    return response.data;
  }

  // Mark a shipment as ready to bill
  async markReadyToBill(shipmentId: Shipment["id"]) {
    const response = await http.put<Shipment>(
      `/shipments/${shipmentId}/mark-ready-to-bill/`,
      {},
    );

    return response.data;
  }

  // Calculate shipment totals (preview)
  async calculateTotals(values: Partial<ShipmentSchema>) {
    const response = await http.post<{
      baseCharge: string | number;
      otherChargeAmount: string | number;
      totalChargeAmount: string | number;
    }>(`/shipments/calculate-totals/`, values);

    return response.data;
  }

  async duplicate(values: ShipmentDuplicateSchema) {
    const response = await http.post<{ message: string }>(
      `/shipments/duplicate/`,
      values,
    );

    return response.data;
  }
}
