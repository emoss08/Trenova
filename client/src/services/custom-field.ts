import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  customFieldDefinitionSchema,
  definitionUsageStatsSchema,
  resourceTypesResponseSchema,
  type CustomFieldDefinition,
  type DefinitionUsageStats,
  type ResourceTypesResponse,
} from "@/types/custom-field";

export class CustomFieldService {
  public async getResourceTypes(): Promise<ResourceTypesResponse> {
    const response = await api.get<ResourceTypesResponse>(
      "/custom-fields/resource-types/",
    );
    return safeParse(resourceTypesResponseSchema, response, "Custom Field Resource Types");
  }

  public async patch(
    id: CustomFieldDefinition["id"],
    data: Partial<CustomFieldDefinition>,
  ) {
    const response = await api.patch<CustomFieldDefinition>(
      `/custom-fields/definitions/${id}/`,
      data,
    );
    return safeParse(customFieldDefinitionSchema, response, "Custom Field Definition");
  }

  public async getByResourceType(resourceType: string) {
    const response = await api.get<CustomFieldDefinition[]>(
      `/custom-fields/resources/${resourceType}/`,
    );
    return response;
  }

  public async getUsageStats(id: string): Promise<DefinitionUsageStats> {
    const response = await api.get<DefinitionUsageStats>(
      `/custom-fields/definitions/${id}/usage/`,
    );
    return safeParse(definitionUsageStatsSchema, response, "Custom Field Usage Stats");
  }

  public async delete(id: string): Promise<void> {
    await api.delete(`/custom-fields/definitions/${id}/`);
  }
}
