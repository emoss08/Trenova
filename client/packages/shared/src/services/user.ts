import {
  BulkUpdateUserStatusDocument,
  MySessionsDocument,
  MySignInsDocument,
  ReplaceUserOrganizationMembershipsDocument,
  RevokeMyOtherSessionsDocument,
  RevokeMySessionDocument,
  SetMyAssistantHiddenDocument,
  UpdateMyAvailabilityDocument,
  UpdateMyProfileDocument,
  UpdateMySettingsDocument,
  UserAvatarsDocument,
  UserOrganizationMembershipsDocument,
  type AvatarStyle,
  type TimeFormat as GraphQLTimeFormat,
} from "@trenova/graphql/generated/graphql";
import { ApiRequestError, api } from "@trenova/shared/lib/api";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  switchOrganizationResponseSchema,
  userOrganizationsResponseSchema,
  type SwitchOrganizationRequest,
  type SwitchOrganizationResponse,
  type UserOrganization,
} from "@trenova/shared/types/organization";
import {
  avatarSummariesSchema,
  sessionSummariesSchema,
  signInSummariesSchema,
  type AvatarSummary,
  type SessionSummary,
  type SignInSummary,
  type UpdateMyAssistantVisibility,
  type UpdateMyAvailability,
  type UpdateMyProfile,
  userOrganizationMembershipsResponseSchema,
  profilePictureUrlResponseSchema,
  userSchema,
  type BulkUpdateUserStatusRequest,
  type ChangeMyPassword,
  type ProfilePictureUrlResponse,
  type ReplaceOrganizationMembershipsRequest,
  type TimeFormatType,
  type UpdateMySettings,
  type User,
  type UserOrganizationMembership,
} from "@trenova/shared/types/user";

const TIME_FORMAT_TO_GRAPHQL: Record<TimeFormatType, GraphQLTimeFormat> = {
  "12-hour": "TwelveHour",
  "24-hour": "TwentyFourHour",
};

const TIME_FORMAT_FROM_GRAPHQL: Record<GraphQLTimeFormat, TimeFormatType> = {
  TwelveHour: "12-hour",
  TwentyFourHour: "24-hour",
};

function isGraphQLTimeFormat(value: unknown): value is GraphQLTimeFormat {
  return typeof value === "string" && Object.hasOwn(TIME_FORMAT_FROM_GRAPHQL, value);
}

/**
 * A user read through GraphQL carries its time format as the schema's enum; the rest of the
 * client, and the session cache, keep the spelling the auth endpoints send.
 */
export function userFromGraphQL(value: unknown, label: string): Promise<User> {
  if (value && typeof value === "object" && "timeFormat" in value) {
    const { timeFormat } = value as { timeFormat: unknown };
    if (isGraphQLTimeFormat(timeFormat)) {
      return safeParse(
        userSchema,
        { ...value, timeFormat: TIME_FORMAT_FROM_GRAPHQL[timeFormat] },
        label,
      );
    }
  }
  return safeParse(userSchema, value, label);
}

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

  public async listOrganizationMemberships(
    userId: string,
    options?: { signal?: AbortSignal },
  ): Promise<UserOrganizationMembership[]> {
    const data = await requestGraphQL<{ userOrganizationMemberships: unknown }>({
      document: UserOrganizationMembershipsDocument,
      operationName: "UserOrganizationMemberships",
      variables: { userId },
      signal: options?.signal,
    });

    return safeParse(
      userOrganizationMembershipsResponseSchema,
      data.userOrganizationMemberships,
      "User Organization Memberships",
    );
  }

  public async replaceOrganizationMemberships(
    userId: string,
    request: ReplaceOrganizationMembershipsRequest,
  ): Promise<UserOrganizationMembership[]> {
    const data = await requestGraphQL<{ replaceUserOrganizationMemberships: unknown }>({
      document: ReplaceUserOrganizationMembershipsDocument,
      operationName: "ReplaceUserOrganizationMemberships",
      variables: { userId, organizationIds: request.organizationIds },
    });

    return safeParse(
      userOrganizationMembershipsResponseSchema,
      data.replaceUserOrganizationMemberships,
      "User Organization Memberships",
    );
  }

  public async bulkUpdateStatus(request: BulkUpdateUserStatusRequest): Promise<void> {
    await requestGraphQL({
      document: BulkUpdateUserStatusDocument,
      operationName: "BulkUpdateUserStatus",
      variables: { input: { userIds: request.userIds, status: request.status } },
    });
  }

  public async currentUser() {
    const response = await api.get<User>(`${this.base_url}/me/`);
    return safeParse(userSchema, response, "User");
  }

  public async updateMySettings(data: UpdateMySettings) {
    const result = await requestGraphQL<{ updateMySettings: unknown }>({
      document: UpdateMySettingsDocument,
      operationName: "UpdateMySettings",
      variables: {
        input: {
          timezone: data.timezone,
          timeFormat: TIME_FORMAT_TO_GRAPHQL[data.timeFormat],
          locale: data.locale,
        },
      },
    });
    return userFromGraphQL(result.updateMySettings, "User Settings");
  }

  public async changeMyPassword(data: ChangeMyPassword) {
    const response = await api.post<User>(`${this.base_url}/me/change-password/`, data);
    return safeParse(userSchema, response, "Change Password");
  }

  public async updateMyProfile(data: UpdateMyProfile) {
    const result = await requestGraphQL<{ updateMyProfile: unknown }>({
      document: UpdateMyProfileDocument,
      operationName: "UpdateMyProfile",
      variables: {
        input: {
          ...(data.jobTitle !== undefined ? { jobTitle: data.jobTitle } : {}),
          ...(data.phoneNumber !== undefined ? { phoneNumber: data.phoneNumber } : {}),
          ...(data.avatarStyle !== undefined
            ? { avatarStyle: data.avatarStyle as AvatarStyle }
            : {}),
        },
      },
    });
    return userFromGraphQL(result.updateMyProfile, "User Profile");
  }

  public async updateMyAvailability(data: UpdateMyAvailability) {
    const delegateUserId = data.delegateUserId?.trim();
    const result = await requestGraphQL<{ updateMyAvailability: unknown }>({
      document: UpdateMyAvailabilityDocument,
      operationName: "UpdateMyAvailability",
      variables: {
        input: {
          status: data.status,
          focusUntil: data.focusUntil ?? null,
          backOn: data.backOn ?? null,
          delegateUserId: delegateUserId ? delegateUserId : null,
          awayNote: data.awayNote ?? "",
        },
      },
    });
    return userFromGraphQL(result.updateMyAvailability, "User Availability");
  }

  public async updateMyAssistantVisibility(data: UpdateMyAssistantVisibility) {
    const result = await requestGraphQL<{ setMyAssistantHidden: unknown }>({
      document: SetMyAssistantHiddenDocument,
      operationName: "SetMyAssistantHidden",
      variables: { hidden: data.assistantHidden },
    });
    return userFromGraphQL(result.setMyAssistantHidden, "Assistant Visibility");
  }

  public async listMySessions(options?: { signal?: AbortSignal }): Promise<SessionSummary[]> {
    const data = await requestGraphQL<{ mySessions: unknown }>({
      document: MySessionsDocument,
      operationName: "MySessions",
      signal: options?.signal,
    });
    return safeParse(sessionSummariesSchema, data.mySessions, "Sessions");
  }

  public async revokeMySession(sessionId: string): Promise<void> {
    await requestGraphQL({
      document: RevokeMySessionDocument,
      operationName: "RevokeMySession",
      variables: { sessionId },
    });
  }

  public async revokeMyOtherSessions(): Promise<{ revoked: number }> {
    const data = await requestGraphQL({
      document: RevokeMyOtherSessionsDocument,
      operationName: "RevokeMyOtherSessions",
    });
    return { revoked: data.revokeMyOtherSessions };
  }

  public async listMySignIns(
    limit = 5,
    options?: { signal?: AbortSignal },
  ): Promise<SignInSummary[]> {
    const data = await requestGraphQL<{ mySignIns: unknown }>({
      document: MySignInsDocument,
      operationName: "MySignIns",
      variables: { limit },
      signal: options?.signal,
    });
    return safeParse(signInSummariesSchema, data.mySignIns, "Sign-ins");
  }

  public async listAvatars(
    ids: readonly string[],
    options?: { signal?: AbortSignal },
  ): Promise<AvatarSummary[]> {
    const data = await requestGraphQL<{ userAvatars: unknown }>({
      document: UserAvatarsDocument,
      operationName: "UserAvatars",
      variables: { ids: [...ids] },
      signal: options?.signal,
    });
    return safeParse(avatarSummariesSchema, data.userAvatars, "Avatars");
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

      return (
        await safeParse(profilePictureUrlResponseSchema, response, "User Profile Picture URL")
      ).url;
    } catch (error) {
      if (error instanceof ApiRequestError && error.isNotFoundError()) {
        return null;
      }

      throw error;
    }
  }
}

export const userService = new UserService();
