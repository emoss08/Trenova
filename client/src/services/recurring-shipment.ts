import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  generateRecurringShipmentResultSchema,
  matchRecurringShipmentsResponseSchema,
  recurringShipmentRunSchema,
  recurringShipmentSchema,
  type GenerateRecurringShipmentResult,
  type MatchRecurringShipmentsResponse,
  type RecurringShipment,
  type RecurringShipmentRun,
  type RecurringShipmentStatus,
} from "@/types/recurring-shipment";
import type { GenericLimitOffsetResponse } from "@/types/server";

export type MatchRecurringShipmentsInput = {
  customerId: string;
  originLocationId: string;
  destinationLocationId: string;
};

export class RecurringShipmentService {
  async getById(id: string, expandDetails = false) {
    const response = await api.get<RecurringShipment>(
      `/recurring-shipments/${id}/${expandDetails ? "?expandDetails=true" : ""}`,
    );
    return safeParse(recurringShipmentSchema, response, "RecurringShipment");
  }

  async create(data: RecurringShipment) {
    const response = await api.post<RecurringShipment>("/recurring-shipments/", data);
    return safeParse(recurringShipmentSchema, response, "RecurringShipment");
  }

  async update(id: string, data: RecurringShipment) {
    const response = await api.put<RecurringShipment>(`/recurring-shipments/${id}/`, data);
    return safeParse(recurringShipmentSchema, response, "RecurringShipment");
  }

  async updateStatus(id: string, status: RecurringShipmentStatus, version: number) {
    const response = await api.put<RecurringShipment>(`/recurring-shipments/${id}/status/`, {
      status,
      version,
    });
    return safeParse(recurringShipmentSchema, response, "RecurringShipment");
  }

  async generate(id: string, occurrenceAt?: number | null) {
    const response = await api.post<GenerateRecurringShipmentResult>(
      `/recurring-shipments/${id}/generate/`,
      occurrenceAt ? { occurrenceAt } : {},
    );
    return safeParse(
      generateRecurringShipmentResultSchema,
      response,
      "GenerateRecurringShipmentResult",
    );
  }

  async match(input: MatchRecurringShipmentsInput) {
    const response = await api.post<MatchRecurringShipmentsResponse>(
      "/recurring-shipments/match/",
      input,
    );
    return safeParse(
      matchRecurringShipmentsResponseSchema,
      response,
      "MatchRecurringShipmentsResponse",
    );
  }

  async listRuns(id: string, params?: { limit?: number; offset?: number }) {
    const search = new URLSearchParams();
    if (params?.limit) search.set("limit", String(params.limit));
    if (params?.offset) search.set("offset", String(params.offset));
    const qs = search.toString();
    const response = await api.get<GenericLimitOffsetResponse<RecurringShipmentRun>>(
      `/recurring-shipments/${id}/runs/${qs ? `?${qs}` : ""}`,
    );
    return {
      ...response,
      results: safeParse(
        recurringShipmentRunSchema.array(),
        response.results,
        "RecurringShipmentRuns",
      ),
    };
  }
}
