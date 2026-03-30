import { z } from "zod/v4";

export const userOrganizationSchema = z.object({
  id: z.string().min(1, { error: "Organization ID is required" }),
  name: z.string().min(1, { error: "Organization name is required" }),
  city: z.string().min(1, { error: "City is required" }),
  state: z.string().min(1, { error: "State is required" }),
  logoUrl: z.string().nullish(),
  isDefault: z.boolean(),
  isCurrent: z.boolean(),
});

export type UserOrganization = z.infer<typeof userOrganizationSchema>;

export const userOrganizationsResponseSchema = z.array(userOrganizationSchema);

export const switchOrganizationRequestSchema = z.object({
  organizationId: z.string(),
});

export type SwitchOrganizationRequest = z.infer<
  typeof switchOrganizationRequestSchema
>;

export const switchOrganizationResponseSchema = z.object({
  user: z.any(),
});

export type SwitchOrganizationResponse = z.infer<
  typeof switchOrganizationResponseSchema
>;

export const organizationSettingsSchema = z.object({
  id: z.string().min(1, { error: "Organization ID is required" }),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  bucketName: z.string().nullish(),
  businessUnitId: z.string().nullish(),
  loginSlug: z.string().nullish(),

  name: z.string().min(1, { error: "Name is required" }),
  scacCode: z.string().min(1, { error: "SCAC code is required" }),
  dotNumber: z.string().min(1, { error: "DOT number is required" }),
  logoUrl: z.string().nullish(),
  addressLine1: z.string().min(1, { error: "Address line 1 is required" }),
  addressLine2: z.string().nullish(),
  city: z.string().min(1, { error: "City is required" }),
  stateId: z.string().min(1, { error: "State is required" }),
  postalCode: z.string().min(1, { error: "Postal code is required" }),
  timezone: z.string().min(1, { error: "Timezone is required" }),
  taxId: z.string().nullish(),
  state: z
    .object({
      id: z.string(),
      name: z.string(),
      abbreviation: z.string().nullish(),
    })
    .nullish(),
});

export type OrganizationSettings = z.infer<typeof organizationSettingsSchema>;

export const organizationLogoUrlResponseSchema = z.object({
  url: z.string(),
});

export type OrganizationLogoUrlResponse = z.infer<
  typeof organizationLogoUrlResponseSchema
>;

export const microsoftSSOConfigSchema = z.object({
  organizationId: z.string().optional().default(""),
  enabled: z.boolean().default(false),
  enforceSso: z.boolean().default(false),
  tenantId: z.string().optional().default(""),
  clientId: z.string().optional().default(""),
  clientSecret: z.string().optional().default(""),
  redirectUrl: z.string().optional().default(""),
  allowedDomains: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  secretConfigured: z.boolean().default(false),
});

export type MicrosoftSSOConfig = z.infer<typeof microsoftSSOConfigSchema>;

export const tenantLoginMetadataSchema = z.object({
  organizationId: z.string(),
  organizationName: z.string(),
  organizationSlug: z.string(),
  microsoftEnabled: z.boolean(),
  passwordEnabled: z.boolean(),
  enforceSso: z.boolean(),
});

export type TenantLoginMetadata = z.infer<typeof tenantLoginMetadataSchema>;
