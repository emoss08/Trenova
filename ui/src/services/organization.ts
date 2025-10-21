/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import { type OrganizationSchema } from "@/lib/schemas/organization-schema";
import { type LimitOffsetResponse } from "@/types/server";

/**
 * Get the organization for the current user
 */
type GetOrganizationByIdParams = {
  orgId: OrganizationSchema["id"];
  includeState?: boolean;
  includeBu?: boolean;
};

export class OrganizationAPI {
  async getById({
    orgId,
    includeState = false,
    includeBu = false,
  }: GetOrganizationByIdParams) {
    const response = await http.get<OrganizationSchema>(
      `/organizations/${orgId}/`,
      {
        params: {
          includeState: includeState.toString(),
          includeBu: includeBu.toString(),
        },
      },
    );
    return response.data;
  }

  async list() {
    const response =
      await http.get<LimitOffsetResponse<OrganizationSchema>>(
        "/organizations/me/",
      );
    return response.data;
  }

  async update(orgId: OrganizationSchema["id"], data: OrganizationSchema) {
    const response = await http.put<OrganizationSchema>(
      `/organizations/${orgId}/`,
      data,
    );
    return response.data;
  }
}
