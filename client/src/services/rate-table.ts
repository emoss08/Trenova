import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { rateTableRowSchema, rateTableSchema, type RateTable } from "@/types/rate-table";
import type { GenericLimitOffsetResponse } from "@/types/server";

export class RateTableService {
  async list(params?: string) {
    const response = await api.get<GenericLimitOffsetResponse<RateTable>>(
      `/rate-tables/${params ? `?${params}` : ""}`,
    );
    return safeParse(rateTableRowSchema.array(), response.results, "RateTables");
  }

  async getById(id: string) {
    const response = await api.get<RateTable>(`/rate-tables/${id}/`);
    return safeParse(rateTableSchema, response, "RateTable");
  }

  async create(data: RateTable) {
    const response = await api.post<RateTable>("/rate-tables/", data);
    return safeParse(rateTableSchema, response, "RateTable");
  }

  async update(id: string, data: RateTable) {
    const response = await api.put<RateTable>(`/rate-tables/${id}/`, data);
    return safeParse(rateTableSchema, response, "RateTable");
  }

  async delete(id: string) {
    await api.delete(`/rate-tables/${id}/`);
  }
}
