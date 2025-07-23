/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { http } from "@/lib/http-client";
import type { ChangePasswordSchema } from "@/lib/schemas/auth-schema";
import type { UserSchema } from "@/lib/schemas/user-schema";

export class UserAPI {
  async getUserById(userId: UserSchema["id"]) {
    const response = await http.get<UserSchema>(`/users/${userId}/`);

    return response.data;
  }

  async switchOrganization(userId: UserSchema["id"], organizationId: string) {
    const response = await http.put<UserSchema>(
      `/users/${userId}/switch-organization/`,
      { organizationId },
    );

    return response.data;
  }

  async changePassword(request: ChangePasswordSchema) {
    return http.post<UserSchema>("/users/change-password/", request);
  }
}
