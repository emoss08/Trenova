import { ApiRequestError, api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  switchOrganizationResponseSchema,
  userOrganizationsResponseSchema,
  type SwitchOrganizationRequest,
  type SwitchOrganizationResponse,
  type UserOrganization,
} from "@/types/organization";
import {
  bulkUpdateUserStatusResponseSchema,
  userOrganizationMembershipsResponseSchema,
  profilePictureUrlResponseSchema,
  userSchema,
  type BulkUpdateUserStatusRequest,
  type BulkUpdateUserStatusResponse,
  type ChangeMyPassword,
  type ProfilePictureUrlResponse,
  type ReplaceOrganizationMembershipsRequest,
  type UpdateMySettings,
  type User,
  type UserOrganizationMembership,
} from "@/types/user";

export class UserService {
  private base_url = "/users";

  public async switchOrganization(request: SwitchOrganizationRequest) {
    const response = await api.post<SwitchOrganizationResponse>(
      `${this.base_url}/me/switch-organization/`,
      request,
    );

    return safeParse(switchOrganizationResponseSchema, response, "Switch Organization");
  }

  public async getUserOrganizations() {
    const response = await api.get<UserOrganization[]>(`${this.base_url}/me/organizations/`);

    return safeParse(userOrganizationsResponseSchema, response, "User Organizations");
  }

  public async listOrganizationMemberships(userId: User["id"]) {
    const response = await api.get<UserOrganizationMembership[]>(
      `${this.base_url}/${userId}/organization-memberships/`,
    );

    return safeParse(userOrganizationMembershipsResponseSchema, response, "User Organization Memberships");
  }

  public async replaceOrganizationMemberships(
    userId: User["id"],
    request: ReplaceOrganizationMembershipsRequest,
  ) {
    const response = await api.put<UserOrganizationMembership[]>(
      `${this.base_url}/${userId}/organization-memberships/`,
      request,
    );

    return safeParse(userOrganizationMembershipsResponseSchema, response, "User Organization Memberships");
  }

  public async bulkUpdateStatus(request: BulkUpdateUserStatusRequest) {
    const response = await api.post<BulkUpdateUserStatusResponse>(
      `${this.base_url}/bulk-update-status/`,
      request,
    );

    return safeParse(bulkUpdateUserStatusResponseSchema, response, "Bulk Update User Status");
  }

  public async get(id: User["id"]) {
    const response = await api.get<User>(`${this.base_url}/${id}/`);
    return safeParse(userSchema, response, "User");
  }

  public async create(user: User) {
    const response = await api.post<User>(`${this.base_url}/`, user);
    return safeParse(userSchema, response, "User");
  }

  public async update(id: User["id"], user: User) {
    const response = await api.put<User>(`${this.base_url}/${id}/`, user);
    return safeParse(userSchema, response, "User");
  }

  public async patch(id: User["id"], data: Partial<User>) {
    const response = await api.patch<User>(`${this.base_url}/${id}/`, data);
    return safeParse(userSchema, response, "User");
  }

  public async currentUser() {
    const response = await api.get<User>(`${this.base_url}/me/`);
    return safeParse(userSchema, response, "User");
  }

  public async updateMySettings(data: UpdateMySettings) {
    const response = await api.patch<User>(`${this.base_url}/me/settings/`, data);
    return safeParse(userSchema, response, "User Settings");
  }

  public async changeMyPassword(data: ChangeMyPassword) {
    const response = await api.post<User>(`${this.base_url}/me/change-password/`, data);
    return safeParse(userSchema, response, "Change Password");
  }

  public async uploadMyProfilePicture(file: File) {
    const formData = new FormData();
    formData.append("file", file);

    const response = await api.upload<User>(`${this.base_url}/me/profile-picture/`, formData);
    return safeParse(userSchema, response, "User Profile Picture");
  }

  public async deleteMyProfilePicture() {
    const response = await api.delete<User>(`${this.base_url}/me/profile-picture/`);
    return safeParse(userSchema, response, "User Profile Picture");
  }

  public async getProfilePictureURL(
    userId: User["id"],
    variant: "thumbnail" | "full" = "thumbnail",
  ) {
    try {
      const response = await api.get<ProfilePictureUrlResponse>(
        `${this.base_url}/${userId}/profile-picture/?variant=${variant}`,
      );

      return (await safeParse(
        profilePictureUrlResponseSchema,
        response,
        "User Profile Picture URL",
      )).url;
    } catch (error) {
      if (error instanceof ApiRequestError && error.isNotFoundError()) {
        return null;
      }

      throw error;
    }
  }
}
