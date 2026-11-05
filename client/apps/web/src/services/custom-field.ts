import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  customFieldDefinitionSchema,
  definitionUsageStatsSchema,
  type CustomFieldDefinition,
  type DefinitionUsageStats,
} from "@/types/custom-field";

export class CustomFieldService {
  public async patch(id: CustomFieldDefinition["id"], data: Partial<CustomFieldDefinition>) {
    const response = await api.patch<CustomFieldDefinition>(
      `/custom-fields/definitions/${id}/`,
      data,
    );
    return safeParse(customFieldDefinitionSchema, response, "Custom Field Definition");
  }

  public async getUsageStats(id: string): Promise<DefinitionUsageStats> {
    const response = await api.get<DefinitionUsageStats>(`/custom-fields/definitions/${id}/usage/`);
    return safeParse(definitionUsageStatsSchema, response, "Custom Field Usage Stats");
  }

  public async delete(id: string): Promise<void> {
    await api.delete(`/custom-fields/definitions/${id}/`);
  }
}

export const customFieldService = new CustomFieldService();
