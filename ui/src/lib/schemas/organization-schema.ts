import { OrganizationType } from "@/types/organization";
import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { usStateSchema } from "./us-state-schema";

const organizationMetadataSchema = z.object({
  objectID: z.string().optional(),
});

export const organizationSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  bucketName: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  name: z.string().min(1, {
    error: "Name is required",
  }),
  scacCode: z.string().min(1, {
    error: "SCAC code is required",
  }),
  dotNumber: z.string().min(1, {
    error: "DOT number is required",
  }),
  logoUrl: z.string().optional(),
  orgType: z.enum(OrganizationType, {
    error: "Organization type is required",
  }),
  addressLine1: z.string().min(1, {
    error: "Address line 1 is required",
  }),
  addressLine2: z.string().optional(),
  city: z.string().min(1, {
    error: "City is required",
  }),
  stateId: z.string().min(1, {
    error: "State is required",
  }),
  postalCode: z.string().min(1, {
    error: "Postal code is required",
  }),
  timezone: z.string().min(1, {
    error: "Timezone is required",
  }),
  taxId: optionalStringSchema,
  state: usStateSchema.optional(),
  metadata: organizationMetadataSchema.optional(),
});

export const organizationMembershipSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  businessUnitId: optionalStringSchema,
  userId: optionalStringSchema,
  organizationId: optionalStringSchema,

  roleIds: optionalStringSchema.array(),
  directPolicies: optionalStringSchema.array(),
  joinedAt: timestampSchema,
  grantedById: optionalStringSchema,
  expiresAt: timestampSchema.optional(),
  isDefault: z.boolean(),
});

export type OrganizationSchema = z.infer<typeof organizationSchema>;

export type OrganizationMembershipSchema = z.infer<
  typeof organizationMembershipSchema
>;
