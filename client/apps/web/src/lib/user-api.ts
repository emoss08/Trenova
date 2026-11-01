import {
  ResetUserMfaDocument,
  ResetUserPasswordDocument,
  UserRoleAssignmentsDocument,
  UserSummaryDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { userRoleAssignmentSchema, type UserRoleAssignment } from "@trenova/shared/types/role";
import { z } from "zod";

type RequestOptions = { signal?: AbortSignal };

export type UserSummary = {
  id: string;
  name: string;
  username: string;
  emailAddress: string | null;
};

export async function getUserSummary(id: string, options: RequestOptions = {}) {
  const data = await requestGraphQL({
    document: UserSummaryDocument,
    operationName: "UserSummary",
    variables: { id },
    signal: options.signal,
  });
  if (!data.user) {
    return null;
  }
  const summary: UserSummary = {
    id: data.user.id,
    name: data.user.name,
    username: data.user.username,
    emailAddress: data.user.emailAddress ?? null,
  };
  return summary;
}

export async function getUserRoleAssignments(
  userId: string,
  options: RequestOptions = {},
): Promise<UserRoleAssignment[]> {
  const data = await requestGraphQL<{ userRoleAssignments: unknown }>({
    document: UserRoleAssignmentsDocument,
    operationName: "UserRoleAssignments",
    variables: { userId },
    signal: options.signal,
  });
  return safeParse(z.array(userRoleAssignmentSchema), data.userRoleAssignments, "Role assignments");
}

export async function resetUserPassword(userId: string): Promise<void> {
  await requestGraphQL({
    document: ResetUserPasswordDocument,
    operationName: "ResetUserPassword",
    variables: { userId },
  });
}

export async function resetUserMFA(userId: string): Promise<void> {
  await requestGraphQL({
    document: ResetUserMfaDocument,
    operationName: "ResetUserMfa",
    variables: { userId },
  });
}
