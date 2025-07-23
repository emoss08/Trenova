/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { http } from "@/lib/http-client";
import type { ShipmentUncancelSchema } from "@/lib/schemas/shipment-cancellation-schema";
import type { ShipmentDuplicateSchema } from "@/lib/schemas/shipment-duplicate-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { LimitOffsetResponse, type ListResult } from "@/types/server";
import { type ShipmentQueryParams } from "@/types/shipment";

export type GetPreviousRatesRequest = {
  originLocationId: string;
  destinationLocationId: string;
  shipmentTypeId: string;
  serviceTypeId: string;
  customerId?: string | null;
};

export class ShipmentAPI {
  // Get shipments from the API
  async getShipments(queryParams: ShipmentQueryParams) {
    const response = await http.get<LimitOffsetResponse<ShipmentSchema>>(
      "/shipments/",
      {
        params: {
          id: queryParams.id,
          query: queryParams.query,
          limit: queryParams.limit?.toString(),
          offset: queryParams.offset?.toString(),
          expandShipmentDetails: queryParams.expandShipmentDetails?.toString(),
          ...(queryParams.tenantOpts && {
            buId: queryParams.tenantOpts.buId,
            orgId: queryParams.tenantOpts.orgId,
            userId: queryParams.tenantOpts.userId,
          }),
          ...(queryParams.sort && {
            sort: JSON.stringify(queryParams.sort),
          }),
          ...(queryParams.filters && {
            filters: JSON.stringify(queryParams.filters),
          }),
        },
      },
    );

    return response.data;
  }

  // Get a shipment by ID from the API
  async getShipmentByID(
    shipmentId: ShipmentSchema["id"],
    expandShipmentDetails = false,
  ) {
    const response = await http.get<ShipmentSchema>(
      `/shipments/${shipmentId}/`,
      {
        params: {
          expandShipmentDetails: expandShipmentDetails.toString(),
        },
      },
    );

    return response.data;
  }

  async create(shipment: ShipmentSchema) {
    return await http.post<ShipmentSchema>("/shipments/", shipment);
  }

  async uncancel(values: ShipmentUncancelSchema) {
    const response = await http.post<ShipmentSchema>(
      `/shipments/uncancel/`,
      values,
    );

    return response.data;
  }

  // Check for duplicate BOLs
  async checkForDuplicateBOLs(
    bol: ShipmentSchema["bol"],
    shipmentId?: ShipmentSchema["id"],
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
  async markReadyToBill(shipmentId: ShipmentSchema["id"]) {
    const response = await http.put<ShipmentSchema>(
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

  async getPreviousRates(values: GetPreviousRatesRequest) {
    const response = await http.post<ListResult<ShipmentSchema>>(
      `/shipments/previous-rates/`,
      values,
    );

    return response.data;
  }
}
