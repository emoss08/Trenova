/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import type { RoleSchema } from "@/lib/schemas/user-schema";
import type { LimitOffsetResponse } from "@/types/server";

export class RoleAPI {
  async list(limit: number, offset: number) {
    const { data } = await http.get<LimitOffsetResponse<RoleSchema>>("/roles", {
      params: {
        limit: limit.toString(),
        offset: offset.toString(),
      },
    });
    return data;
  }

  async getById(id: string) {
    const { data } = await http.get<RoleSchema>(`/roles/${id}`, {
      params: {
        includeChildren: "true",
        includePermissions: "true",
        includeParent: "true",
      },
    });
    return data;
  }
}
