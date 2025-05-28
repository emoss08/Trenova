import { http } from "@/lib/http-client";
import type { Integration, IntegrationType } from "@/types/integration";
import type { LimitOffsetResponse } from "@/types/server";

export class IntegrationAPI {
  async get() {
    const response =
      await http.get<LimitOffsetResponse<Integration>>("/integrations/");
    return response.data;
  }

  async getByType(type: IntegrationType) {
    const response = await http.get<Integration>(`/integrations/${type}`);
    return response.data;
  }

  async update(
    integrationId: string,
    data: Record<string, any>,
    userId: string,
  ) {
    return http.put(`/integrations/${integrationId}`, {
      enabled: true, // * We want to set the integration to enabled if it's not already
      configuration: data,
      enabledById: userId,
    });
  }
}
