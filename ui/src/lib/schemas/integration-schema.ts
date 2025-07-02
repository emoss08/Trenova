import { IntegrationCategory, IntegrationType } from "@/types/integration";
import * as z from "zod/v4";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { organizationSchema } from "./organization-schema";
import { userSchema } from "./user-schema";

export const googleMapsConfigurationSchema = z.object({
  apiKey: z.string().min(1, { error: "API key is required" }),
});

export const pcmilerConfigurationSchema = z.object({
  username: z.string().min(1, { error: "Username is required" }),
  password: z.string().min(1, { error: "Password is required" }),
  licenseKey: z.string().min(1, { error: "License key is required" }),
});

export const integrationSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  enabled: z.boolean(),
  name: z.string(),
  description: z.string(),
  builtBy: z.string(),
  category: z.enum(IntegrationCategory),
  type: z.enum(IntegrationType),
  configuration: z.record(z.string(), z.any()),
  enabledById: nullableStringSchema,
  // * Relationships
  organization: organizationSchema.nullish(),
  enabledBy: userSchema.nullish(),
});

export type GoogleMapsConfigurationSchema = z.infer<
  typeof googleMapsConfigurationSchema
>;

export type PCMilerConfigurationSchema = z.infer<
  typeof pcmilerConfigurationSchema
>;

export type IntegrationSchema = z.infer<typeof integrationSchema>;
