import { http } from "@/lib/http-client";
import type {
  ConsolidationGroupSchema,
  CreateConsolidationSchema,
} from "@/lib/schemas/consolidation-schema";

export type AddShipmentsRequest = {
  consolidationId: string;
  shipmentIds: string[];
};

export type RemoveShipmentsRequest = {
  consolidationId: string;
  shipmentIds: string[];
};

export class ConsolidationAPI {
  async getConsolidationGroupByID(
    consolidationId: ConsolidationGroupSchema["id"],
    expandDetails = true,
  ) {
    const response = await http.get<ConsolidationGroupSchema>(
      `/consolidations/${consolidationId}/`,
      {
        params: {
          expandDetails: expandDetails.toString(),
        },
      },
    );

    return response.data;
  }

  // Create a new consolidation group
  async create(request: CreateConsolidationSchema) {
    const response = await http.post<ConsolidationGroupSchema>(
      "/consolidations/",
      request,
    );

    return response.data;
  }

  // Update a consolidation group
  async update(
    consolidationId: string,
    data: Partial<ConsolidationGroupSchema>,
  ) {
    const response = await http.put<ConsolidationGroupSchema>(
      `/consolidations/${consolidationId}/`,
      data,
    );

    return response.data;
  }

  // Delete a consolidation group
  async delete(consolidationId: string) {
    await http.delete(`/consolidations/${consolidationId}/`);
  }

  // Add shipments to a consolidation group
  async addShipments(request: AddShipmentsRequest) {
    const response = await http.post<ConsolidationGroupSchema>(
      `/consolidations/${request.consolidationId}/add-shipments/`,
      { shipmentIds: request.shipmentIds },
    );

    return response.data;
  }

  // Remove shipments from a consolidation group
  async removeShipments(request: RemoveShipmentsRequest) {
    const response = await http.post<ConsolidationGroupSchema>(
      `/consolidations/${request.consolidationId}/remove-shipments/`,
      { shipmentIds: request.shipmentIds },
    );

    return response.data;
  }
}
