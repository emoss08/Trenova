import * as z from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

/**
 * Schema for the preferences data structure stored in JSONB
 */
export const preferenceDataSchema = z.object({
  dismissedNotices: z.array(z.string()).default([]),
  dismissedDialogs: z.array(z.string()).default([]),
  uiSettings: z.record(z.string(), z.any()).default({}),
});

export type PreferenceDataSchema = z.infer<typeof preferenceDataSchema>;

/**
 * Schema for the complete user preference object
 */
export const userPreferenceSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,

  userId: z.string().min(1, {
    message: "User ID is required",
  }),
  organizationId: z.string().min(1, {
    message: "Organization ID is required",
  }),
  businessUnitId: z.string().min(1, {
    message: "Business Unit ID is required",
  }),
  preferences: preferenceDataSchema,
});

export type UserPreferenceSchema = z.infer<typeof userPreferenceSchema>;

/**
 * Schema for updating preferences (partial updates via PATCH)
 */
export const updatePreferenceDataSchema = z.object({
  dismissedNotices: z.array(z.string()).optional(),
  dismissedDialogs: z.array(z.string()).optional(),
  uiSettings: z.record(z.string(), z.any()).optional(),
});

export type UpdatePreferenceDataSchema = z.infer<
  typeof updatePreferenceDataSchema
>;
