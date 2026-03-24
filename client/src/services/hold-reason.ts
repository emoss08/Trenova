import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { holdReasonSchema, type HoldReason } from "@/types/hold-reason";
import type { GenericLimitOffsetResponse } from "@/types/server";

export class HoldReasonService {
  async getAll(params?: string) {
    const response = await api.get<GenericLimitOffsetResponse<HoldReason>>(
      `/hold-reasons/${params ? `?${params}` : ""}`,
    );
    return safeParse(holdReasonSchema.array(), response.results, "HoldReasons");
  }

  async getById(id: string) {
    const response = await api.get<HoldReason>(`/hold-reasons/${id}/`);
    return safeParse(holdReasonSchema, response, "HoldReason");
  }

  async create(data: HoldReason) {
    const response = await api.post<HoldReason>("/hold-reasons/", data);
    return safeParse(holdReasonSchema, response, "HoldReason");
  }

  async update(id: string, data: HoldReason) {
    const response = await api.put<HoldReason>(`/hold-reasons/${id}/`, data);
    return safeParse(holdReasonSchema, response, "HoldReason");
  }
}
