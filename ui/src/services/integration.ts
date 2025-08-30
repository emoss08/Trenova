/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import type { IntegrationSchema } from "@/lib/schemas/integration-schema";
import type { LimitOffsetResponse } from "@/types/server";

export class IntegrationAPI {
  async get() {
    const response =
      await http.get<LimitOffsetResponse<IntegrationSchema>>("/integrations/");
    return response.data;
  }

  async getByType(type: IntegrationSchema["type"]) {
    const response = await http.get<IntegrationSchema>(`/integrations/${type}`);
    return response.data;
  }

  // Fetch a dynamic configuration form spec for an integration type
  async getFormSpec(type: IntegrationSchema["type"]) {
    const response = await http.get(`/integrations/${type}/form-spec`);
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
