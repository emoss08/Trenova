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
