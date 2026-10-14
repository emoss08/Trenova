import {
  CreateUserDocument,
  PatchUserDocument,
  UserDetailDocument,
  type CreateUserInput,
  type CreateUserMutation,
  type CreateUserMutationVariables,
  type PatchUserMutation,
  type PatchUserMutationVariables,
  type TimeFormat as GraphQLTimeFormat,
  type UserDetailQuery,
  type UserDetailQueryVariables,
  type UserPatchInput,
} from "@trenova/graphql/generated/graphql";
import { DEFAULT_LOCALE, LOCALES, type Locale } from "@trenova/shared/i18n/generated/locales";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  TimeFormatType,
  UserCreateFormValues,
  UserEditFormValues,
} from "@trenova/shared/types/user";

export type UserDetail = NonNullable<UserDetailQuery["user"]>;
export type CreatedUser = CreateUserMutation["createUser"];
export type PatchedUser = PatchUserMutation["patchUser"];

const TIME_FORMAT_TO_GRAPHQL: Record<TimeFormatType, GraphQLTimeFormat> = {
  "12-hour": "TwelveHour",
  "24-hour": "TwentyFourHour",
};

const TIME_FORMAT_FROM_GRAPHQL: Record<GraphQLTimeFormat, TimeFormatType> = {
  TwelveHour: "12-hour",
  TwentyFourHour: "24-hour",
};

export function timeFormatToGraphQL(value: TimeFormatType): GraphQLTimeFormat {
  return TIME_FORMAT_TO_GRAPHQL[value];
}

export function timeFormatFromGraphQL(value: GraphQLTimeFormat): TimeFormatType {
  return TIME_FORMAT_FROM_GRAPHQL[value];
}

function toLocale(value: string): Locale {
  return (LOCALES as readonly string[]).includes(value) ? (value as Locale) : DEFAULT_LOCALE;
}

export function userDetailToFormValues(detail: UserDetail | PatchedUser): UserEditFormValues {
  return {
    id: detail.id,
    version: detail.version,
    status: detail.status,
    name: detail.name,
    username: detail.username,
    emailAddress: detail.emailAddress,
    timezone: detail.timezone,
    timeFormat: timeFormatFromGraphQL(detail.timeFormat),
    locale: toLocale(detail.locale),
    isLocked: detail.isLocked,
    mustChangePassword: detail.mustChangePassword,
    jobTitle: detail.jobTitle,
    phoneNumber: detail.phoneNumber,
    lastLoginAt: detail.lastLoginAt,
    passwordChangedAt: detail.passwordChangedAt,
    createdAt: detail.createdAt,
    updatedAt: detail.updatedAt,
  };
}

export function userCreateInput(values: UserCreateFormValues): CreateUserInput {
  return {
    name: values.name.trim(),
    username: values.username.trim(),
    emailAddress: values.emailAddress.trim(),
    timezone: values.timezone,
    timeFormat: timeFormatToGraphQL(values.timeFormat),
    locale: values.locale,
    status: values.status,
    roleIds: values.roleIds,
  };
}

type PatchableField = Exclude<keyof UserPatchInput, "timeFormat">;

const PATCHABLE_FIELDS = [
  "status",
  "name",
  "emailAddress",
  "timezone",
  "locale",
  "isLocked",
  "mustChangePassword",
  "jobTitle",
  "phoneNumber",
] as const satisfies readonly PatchableField[];

// Only what changed is sent, so a field the form shows but the person did not touch can
// never overwrite a value somebody else saved in the meantime.
export function buildUserPatch(
  original: UserEditFormValues,
  values: UserEditFormValues,
): UserPatchInput {
  const patch: UserPatchInput = {};

  for (const field of PATCHABLE_FIELDS) {
    if (values[field] !== original[field]) {
      Object.assign(patch, { [field]: values[field] });
    }
  }

  if (values.timeFormat !== original.timeFormat) {
    patch.timeFormat = timeFormatToGraphQL(values.timeFormat);
  }

  return patch;
}

export async function fetchUserDetail(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<UserDetail> {
  const data = await requestGraphQL<UserDetailQuery, UserDetailQueryVariables>({
    document: UserDetailDocument,
    operationName: "UserDetail",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.user) {
    throw new Error(`User ${id} was not found`);
  }
  return data.user;
}

export async function createUser(input: CreateUserInput): Promise<CreatedUser> {
  const data = await requestGraphQL<CreateUserMutation, CreateUserMutationVariables>({
    document: CreateUserDocument,
    operationName: "CreateUser",
    variables: { input },
  });
  return data.createUser;
}

export async function patchUser(id: string, input: UserPatchInput): Promise<PatchedUser> {
  const data = await requestGraphQL<PatchUserMutation, PatchUserMutationVariables>({
    document: PatchUserDocument,
    operationName: "PatchUser",
    variables: { id, input },
  });
  return data.patchUser;
}
