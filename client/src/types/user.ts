import { z } from "zod";
import {
  optionalStringSchema,
  statusSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { roleSchema } from "./role";
import { createLimitOffsetResponse } from "./server";
export { apiErrorResponseSchema, type ApiErrorResponse } from "./errors";

export const TimeFormat = z.enum(["12-hour", "24-hour"]);
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
    })
    .nullish(),
});

export type UserOrganizationMembership = z.infer<
  typeof userOrganizationMembershipSchema
>;

export const userOrganizationMembershipsResponseSchema = z.array(
  userOrganizationMembershipSchema,
);

export const replaceOrganizationMembershipsRequestSchema = z.object({
  organizationIds: z.array(z.string()),
});

export type ReplaceOrganizationMembershipsRequest = z.infer<
  typeof replaceOrganizationMembershipsRequestSchema
>;

export const userSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  businessUnitId: optionalStringSchema,
  currentOrganizationId: optionalStringSchema,

  status: statusSchema,
  name: z.string().min(1, { error: "Name is required" }),
  username: z.string().min(1, { error: "Username is required" }),
  emailAddress: z.email().min(1, { error: "Email address is required" }),
  profilePicUrl: optionalStringSchema,
  thumbnailUrl: optionalStringSchema,
  timezone: z.string().min(1, { error: "Timezone is required" }),
  timeFormat: TimeFormat.default("12-hour"),
  isLocked: z.boolean().default(false),
  mustChangePassword: z.boolean().default(true),
  lastLoginAt: timestampSchema.optional(),

  assignments: z.array(userRoleAssignmentSchema).nullish(),
  memberships: z.array(userOrganizationMembershipSchema).nullish(),
});

export type User = z.infer<typeof userSchema>;

export const userResponseSchema = createLimitOffsetResponse(userSchema);

export type UserResponse = z.infer<typeof userResponseSchema>;

export const loginRequestSchema = z.object({
  emailAddress: z.email({
    error: "Please enter a valid email address",
  }),
  password: z
    .string({ error: "Password is required" })
    .min(1, "Password is required"),
  organizationSlug: z.string().optional(),
});

export type LoginRequest = z.infer<typeof loginRequestSchema>;

export const loginResponseSchema = z.object({
  user: userSchema,
  sessionId: z.string(),
  expiresAt: z.number(),
});

export type LoginResponse = z.infer<typeof loginResponseSchema>;

export const updateMySettingsSchema = z.object({
  timezone: z.string().min(1, { error: "Timezone is required" }),
  timeFormat: TimeFormat,
  profilePicUrl: z.string().optional(),
  thumbnailUrl: z.string().optional(),
});

export type UpdateMySettings = z.infer<typeof updateMySettingsSchema>;

export const changeMyPasswordSchema = z.object({
  currentPassword: z.string().min(1, { error: "Current password is required" }),
  newPassword: z.string().min(1, { error: "New password is required" }),
  confirmPassword: z.string().min(1, { error: "Confirm password is required" }),
});

export type ChangeMyPassword = z.infer<typeof changeMyPasswordSchema>;

export const bulkUpdateUserStatusRequestSchema = z.object({
  userIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateUserStatusRequest = z.infer<
  typeof bulkUpdateUserStatusRequestSchema
>;

export const bulkUpdateUserStatusResponseSchema = z.array(userSchema);

export type BulkUpdateUserStatusResponse = z.infer<
  typeof bulkUpdateUserStatusResponseSchema
>;
