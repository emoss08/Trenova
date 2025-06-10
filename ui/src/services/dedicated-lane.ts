import { http } from "@/lib/http-client";
import {
  DedicatedLaneSchema,
  DedicatedLaneSuggestionSchema,
} from "@/lib/schemas/dedicated-lane-schema";
import type { LimitOffsetResponse } from "@/types/server";

export type GetDedicatedLaneByShipmentRequest = {
  customerId: string;
  serviceTypeId: string;
  shipmentTypeId: string;
  originLocationId?: string | null;
  destinationLocationId?: string | null;
  trailerTypeId?: string | null;
  tractorTypeId?: string | null;
};

export class DedicatedLaneAPI {
  async getByShipment(req: GetDedicatedLaneByShipmentRequest) {
    const response = await http.post<DedicatedLaneSchema | null>(
      "/dedicated-lanes/find-by-shipment",
      req,
    );

    return response.data;
  }
}

export class DedicatedLaneSuggestionAPI {
  async getSuggestions(limit: number, offset: number) {
    const response = await http.get<
      LimitOffsetResponse<DedicatedLaneSuggestionSchema>
    >("/dedicated-lane-suggestions/", {
      params: {
        limit: limit.toString(),
        offset: offset.toString(),
      },
    });

    return response.data;
  }

  async getSuggestionByID(id: string) {
    const response = await http.get<DedicatedLaneSuggestionSchema | null>(
      `/dedicated-lane-suggestions/${id}`,
    );

    return response.data;
  }

  async acceptSuggestion(id: string) {
    const response = await http.post<DedicatedLaneSuggestionSchema | null>(
      `/dedicated-lane-suggestions/${id}/accept`,
    );

    return response.data;
  }

  async rejectSuggestion(id: string) {
    const response = await http.post<DedicatedLaneSuggestionSchema | null>(
      `/dedicated-lane-suggestions/${id}/reject`,
    );

    return response.data;
  }

  async analyzePatterns() {
    const response = await http.post<DedicatedLaneSuggestionSchema | null>(
      `/dedicated-lane-suggestions/analyze-patterns`,
    );

    return response.data;
  }

  async expireOldSuggestions() {
    const response = await http.post<DedicatedLaneSuggestionSchema | null>(
      `/dedicated-lane-suggestions/expire-old`,
    );

    return response.data;
  }
}
