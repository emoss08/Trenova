import { DEFAULT_LOCALE, LOCALES } from "@trenova/shared/i18n/generated/locales";
import { z } from "zod";
import {
  auditTimestampSchema,
  capabilityFlagSchema,
  nullableTimestampSchema,
  optionalStringSchema,
  statusSchema,
  stringArraySchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { roleSchema, roleSummaryArraySchema } from "./role";
import { createLimitOffsetResponse } from "./server";
import { translate } from "@trenova/shared/i18n/runtime";
export { apiErrorResponseSchema, type ApiErrorResponse } from "./errors";

export const TimeFormat = z.enum(["12-hour", "24-hour"]);

export const availabilityStatusSchema = z.enum(["available", "focus", "away"]);
export type AvailabilityStatus = z.infer<typeof availabilityStatusSchema>;
export type TimeFormatType = z.infer<typeof TimeFormat>;

export const userRoleAssignmentSchema = z.object({
  id: optionalStringSchema,
  userId: z.string(),
  organizationId: optionalStringSchema,
  roleId: z.string(),
  expiresAt: z.number().nullable().optional(),
  assignedBy: optionalStringSchema,
  assignedAt: timestampSchema,
  role: roleSchema.optional(),
});

export type UserRoleAssignment = z.infer<typeof userRoleAssignmentSchema>;

export const userOrganizationMembershipSchema = z.object({
  id: optionalStringSchema,
  userId: z.string(),
  businessUnitId: optionalStringSchema,
  organizationId: z.string(),
  isDefault: z.boolean().default(false),
  joinedAt: timestampSchema.optional(),
  organization: z
    .object({
      id: z.string(),
      name: z.string(),
      city: z.string().nullish(),
      state: z
        .object({
          id: z.string().optional(),
          name: z.string().nullish(),
        })
        .nullish(),
      brokerageEnabled: capabilityFlagSchema,
      assetOperationsEnabled: capabilityFlagSchema,
    })
    .nullish(),
});

export type UserOrganizationMembership = z.infer<typeof userOrganizationMembershipSchema>;

export const userOrganizationMembershipsResponseSchema = z.array(userOrganizationMembershipSchema);

export const replaceOrganizationMembershipsRequestSchema = z.object({
  organizationIds: z.array(z.string()),
});

export type ReplaceOrganizationMembershipsRequest = z.infer<
  typeof replaceOrganizationMembershipsRequestSchema
>;

export const userSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: auditTimestampSchema,
  updatedAt: auditTimestampSchema,
  businessUnitId: optionalStringSchema,
  currentOrganizationId: optionalStringSchema,

  status: statusSchema,
  name: z.string().min(1, { error: () => translate("Name is required") }),
  username: z.string().min(1, { error: () => translate("Username is required") }),
  emailAddress: z.email().min(1, { error: () => translate("Email address is required") }),
  profilePicUrl: optionalStringSchema,
  thumbnailUrl: optionalStringSchema,
  timezone: z.string().min(1, { error: () => translate("Timezone is required") }),
  timeFormat: TimeFormat.default("12-hour"),
  locale: z.enum(LOCALES).default(DEFAULT_LOCALE),
  isLocked: z.boolean().default(false),
  mustChangePassword: z.boolean().default(true),
  lastLoginAt: nullableTimestampSchema,
  jobTitle: z.string().catch("").default(""),
  phoneNumber: z.string().catch("").default(""),
  avatarStyle: z.string().catch("auto").default("auto"),
  availabilityStatus: availabilityStatusSchema.catch("available").default("available"),
  focusUntil: z.number().nullish(),
  awayBackOn: z.number().nullish(),
  awayDelegateUserId: z.string().nullish(),
  awayNote: z.string().catch("").default(""),
  passwordChangedAt: nullableTimestampSchema,

  assignments: z.array(userRoleAssignmentSchema).nullish(),
  memberships: z.array(userOrganizationMembershipSchema).nullish(),
});

export type User = z.infer<typeof userSchema>;

// The fields an administrator sets on any account. Both forms derive from userSchema,
// so a rule changed there reaches create and edit alike.
const userAccountShape = {
  status: true,
  name: true,
  username: true,
  emailAddress: true,
  timezone: true,
  timeFormat: true,
  locale: true,
} as const;

export const userCreateSchema = userSchema.pick(userAccountShape).extend({
  roleIds: z.array(z.string()),
});

export type UserCreateFormValues = z.infer<typeof userCreateSchema>;

export const userEditSchema = userSchema.pick({
  ...userAccountShape,
  id: true,
  version: true,
  isLocked: true,
  mustChangePassword: true,
  jobTitle: true,
  phoneNumber: true,
  lastLoginAt: true,
  passwordChangedAt: true,
  createdAt: true,
  updatedAt: true,
});

export type UserEditFormValues = z.infer<typeof userEditSchema>;

export const userResponseSchema = createLimitOffsetResponse(userSchema);

export type UserResponse = z.infer<typeof userResponseSchema>;

export const loginRequestSchema = z.object({
  emailAddress: z.email({
    error: () => translate("Please enter a valid email address"),
  }),
  password: z
    .string({ error: () => translate("Password is required") })
    .min(1, { error: () => translate("Password is required") }),
  organizationSlug: z.string().optional(),
});

export type LoginRequest = z.infer<typeof loginRequestSchema>;

export const loginResponseSchema = z.object({
  user: userSchema,
  sessionId: z.string(),
  expiresAt: z.number(),
  csrfToken: z.string(),
  activeRoleIds: stringArraySchema,
  authorizedRoleIds: stringArraySchema,
  activeRoles: roleSummaryArraySchema,
  authorizedRoles: roleSummaryArraySchema,
  requiresRoleActivation: z.boolean().optional().default(false),
});

export type LoginResponse = z.infer<typeof loginResponseSchema>;

export const updateMySettingsSchema = z.object({
  timezone: z.string().min(1, { error: () => translate("Timezone is required") }),
  timeFormat: TimeFormat,
  locale: z.enum(LOCALES),
});

export type UpdateMySettings = z.infer<typeof updateMySettingsSchema>;

export const changeMyPasswordSchema = z.object({
  currentPassword: z.string().min(1, { error: () => translate("Current password is required") }),
  newPassword: z.string().min(1, { error: () => translate("New password is required") }),
  confirmPassword: z.string().min(1, { error: () => translate("Confirm password is required") }),
});

export type ChangeMyPassword = z.infer<typeof changeMyPasswordSchema>;

export const profilePictureUrlResponseSchema = z.object({
  url: z.string(),
});

export type ProfilePictureUrlResponse = z.infer<typeof profilePictureUrlResponseSchema>;

export const bulkUpdateUserStatusRequestSchema = z.object({
  userIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateUserStatusRequest = z.infer<typeof bulkUpdateUserStatusRequestSchema>;

export const bulkUpdateUserStatusResponseSchema = z.array(userSchema);

export type BulkUpdateUserStatusResponse = z.infer<typeof bulkUpdateUserStatusResponseSchema>;

export const updateMyProfileSchema = z.object({
  jobTitle: z.string().max(100).optional(),
  phoneNumber: z.string().max(30).optional(),
  avatarStyle: z.string().optional(),
});

export type UpdateMyProfile = z.infer<typeof updateMyProfileSchema>;

export const updateMyAvailabilitySchema = z.object({
  status: availabilityStatusSchema,
  focusUntil: z.number().nullable().optional(),
  backOn: z.number().nullable().optional(),
  delegateUserId: z.string().nullable().optional(),
  awayNote: z.string().max(280).optional(),
});

export type UpdateMyAvailability = z.infer<typeof updateMyAvailabilitySchema>;

export const deviceKindSchema = z.enum(["desktop", "mobile", "tablet", "unknown"]).catch("unknown");
export type DeviceKind = z.infer<typeof deviceKindSchema>;

export const sessionSummarySchema = z.object({
  id: z.string(),
  label: z.string().catch(""),
  browser: z.string().catch(""),
  os: z.string().catch(""),
  device: deviceKindSchema,
  ipAddress: z.string().catch(""),
  authProvider: z.string().catch(""),
  createdAt: z.number(),
  lastAccessedAt: z.number(),
  expiresAt: z.number(),
  current: z.boolean(),
});

export type SessionSummary = z.infer<typeof sessionSummarySchema>;

export const sessionSummariesSchema = z.array(sessionSummarySchema);

export const revokeSessionsResponseSchema = z.object({ revoked: z.number() });

export const signInSummarySchema = z.object({
  id: z.string(),
  provider: z.string(),
  outcome: z.enum(["success", "challenge", "denied", "failed"]).catch("failed"),
  mfaState: z.string().catch(""),
  errorCode: z.string().catch(""),
  ipAddress: z.string().catch(""),
  label: z.string().catch(""),
  device: deviceKindSchema,
  occurredAt: z.number(),
});

export type SignInSummary = z.infer<typeof signInSummarySchema>;

export const signInSummariesSchema = z.array(signInSummarySchema);

export const avatarSummarySchema = z.object({
  id: z.string(),
  name: z.string(),
  username: z.string(),
  avatarStyle: z.string().catch("auto"),
  availability: availabilityStatusSchema.catch("available"),
  awayBackOn: z.number().nullish(),
  awayNote: z.string().catch(""),
});

export type AvatarSummary = z.infer<typeof avatarSummarySchema>;

export const avatarSummariesSchema = z.array(avatarSummarySchema);
