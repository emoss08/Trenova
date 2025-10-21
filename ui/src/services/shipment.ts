import { http } from "@/lib/http-client";
import type { ShipmentUncancelSchema } from "@/lib/schemas/shipment-cancellation-schema";
import { ShipmentCommentSchema } from "@/lib/schemas/shipment-comment-schema";
import type { ShipmentDuplicateSchema } from "@/lib/schemas/shipment-duplicate-schema";
import {
  HoldShipmentRequestSchema,
  ReleaseShipmentHoldRequestSchema,
  ShipmentHoldSchema,
} from "@/lib/schemas/shipment-hold-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { LimitOffsetResponse, type ListResult } from "@/types/server";
import { type ShipmentQueryParams } from "@/types/shipment";

export type GetPreviousRatesRequest = {
  originLocationId: string;
  destinationLocationId: string;
  shipmentTypeId: string;
  serviceTypeId: string;
  customerId?: string | null;
  excludeShipmentId?: string | null;
};

export class ShipmentAPI {
  // Get shipments from the API
  async getShipments(queryParams: ShipmentQueryParams) {
    const response = await http.get<LimitOffsetResponse<ShipmentSchema>>(
      "/shipments/",
      {
        params: {
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

  async update(shipmentId: ShipmentSchema["id"], shipment: ShipmentSchema) {
    return await http.put<ShipmentSchema>(
      `/shipments/${shipmentId}/`,
      shipment,
    );
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

  async addComment(
    shipmentId: ShipmentSchema["id"],
    values: ShipmentCommentSchema,
  ) {
    const response = await http.post<ShipmentCommentSchema>(
      `/shipments/${shipmentId}/comments/`,
      values,
    );

    return response.data;
  }

  async listComments(shipmentId: ShipmentSchema["id"]) {
    const response = await http.get<LimitOffsetResponse<ShipmentCommentSchema>>(
      `/shipments/${shipmentId}/comments/`,
    );

    return response.data;
  }

  async updateComment(
    commentId: ShipmentCommentSchema["id"],
    values: ShipmentCommentSchema,
  ) {
    const response = await http.put<ShipmentCommentSchema>(
      `/shipments/${values.shipmentId}/comments/${commentId}/`,
      values,
    );

    return response.data;
  }

  async deleteComment(
    shipmentId: ShipmentSchema["id"],
    commentId: ShipmentCommentSchema["id"],
  ) {
    await http.delete(`/shipments/${shipmentId}/comments/${commentId}/`);
  }

  async getCommentCount(shipmentId: ShipmentSchema["id"]) {
    const response = await http.get<{ count: number }>(
      `/shipments/${shipmentId}/comments/count/`,
    );

    return response.data;
  }

  async getHolds(shipmentId: ShipmentSchema["id"]) {
    const response = await http.get<LimitOffsetResponse<ShipmentHoldSchema>>(
      `/shipment-holds/${shipmentId}/`,
    );

    return response.data;
  }

  async applyHold(values: HoldShipmentRequestSchema) {
    const response = await http.post<HoldShipmentRequestSchema>(
      "/shipment-holds/hold/",
      values,
    );

    return response.data;
  }

  async releaseHold(values: ReleaseShipmentHoldRequestSchema) {
    const response = await http.post<ReleaseShipmentHoldRequestSchema>(
      "/shipment-holds/release/",
      values,
    );

    return response.data;
  }
}
