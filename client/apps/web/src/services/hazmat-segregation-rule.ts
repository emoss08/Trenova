import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  hazmatSegregationRuleSchema,
  type HazmatSegregationRule,
} from "@/types/hazmat-segregation-rule";
import type { GenericLimitOffsetResponse } from "@/types/server";

export class HazmatSegregationRuleService {
  async getAll(params?: string) {
    const response = await api.get<GenericLimitOffsetResponse<HazmatSegregationRule>>(
      `/hazmat-segregation-rules/${params ? `?${params}` : ""}`,
    );

    return safeParse(hazmatSegregationRuleSchema.array(), response.results, "HazmatSegregationRules");
  }

  async getById(id: string) {
    const response = await api.get<HazmatSegregationRule>(
      `/hazmat-segregation-rules/${id}/`,
    );

    return safeParse(hazmatSegregationRuleSchema, response, "HazmatSegregationRule");
  }

  async create(data: HazmatSegregationRule) {
    const response = await api.post<HazmatSegregationRule>(
      "/hazmat-segregation-rules/",
      data,
    );

    return safeParse(hazmatSegregationRuleSchema, response, "HazmatSegregationRule");
  }

  async update(id: string, data: HazmatSegregationRule) {
    const response = await api.put<HazmatSegregationRule>(
      `/hazmat-segregation-rules/${id}/`,
      data,
    );

    return safeParse(hazmatSegregationRuleSchema, response, "HazmatSegregationRule");
  }
}
