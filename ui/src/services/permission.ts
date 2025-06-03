import { http } from "@/lib/http-client";
import type { PermissionSchema } from "@/lib/schemas/user-schema";
import type { LimitOffsetResponse } from "@/types/server";

export class PermissionAPI {
  async list(limit: number, offset: number) {
    const { data } = await http.get<LimitOffsetResponse<PermissionSchema>>(
      "/permissions",
      {
        params: {
          limit: limit.toString(),
          offset: offset.toString(),
        },
      },
    );
    return data;
  }

  async getById(id: string) {
    const { data } = await http.get<PermissionSchema>(`/permissions/${id}`, {
      params: {
        includeChildren: "true",
        includePermissions: "true",
        includeParent: "true",
      },
    });
    return data;
  }
}
