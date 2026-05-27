import { z } from "zod/v4";
import { createLimitOffsetResponse } from "./server";

const stringArraySchema = z
  .array(z.string())
  .nullish()
  .transform((value) => value ?? []);

const stringMapSchema = z
  .record(z.string(), z.string())
  .nullish()
  .transform((value) => value ?? {});

export const identityProviderSchema = z.object({
  id: z.string().optional().default(""),
  organizationId: z.string().optional().default(""),
  businessUnitId: z.string().optional().default(""),
  name: z.string().default(""),
  slug: z.string().default(""),
  protocol: z.literal("OIDC").default("OIDC"),
  enabled: z.boolean().default(true),
  enforceSso: z.boolean().default(false),
  autoProvision: z.boolean().default(false),
  allowFederatedMfa: z.boolean().default(true),
  allowedDomains: stringArraySchema,
  attributeMap: stringMapSchema,
  oidcIssuerUrl: z.string().optional().default(""),
  oidcClientId: z.string().optional().default(""),
  oidcClientSecret: z.string().optional().default(""),
  oidcRedirectUrl: z.string().optional().default(""),
  oidcScopes: stringArraySchema,
  version: z.number().default(0),
  createdAt: z.number().default(0),
  updatedAt: z.number().default(0),
});

export type IdentityProvider = z.infer<typeof identityProviderSchema>;

export const identityProvidersSchema = z.array(identityProviderSchema);

export const identityProviderFormSchema = identityProviderSchema.extend({
  name: z.string().trim().min(1, "Provider name is required"),
  slug: z.string().trim().min(1, "Provider slug is required"),
  allowedDomains: z.array(z.string().trim().min(1)).default([]),
  attributeMap: z.record(z.string(), z.string()).default({ email: "email" }),
  oidcIssuerUrl: z.string().trim().min(1, "Issuer URL is required"),
  oidcClientId: z.string().trim().min(1, "Client ID is required"),
  oidcClientSecret: z.string().default(""),
  oidcRedirectUrl: z.string().trim().min(1, "Redirect URI is required"),
  oidcScopes: z.array(z.string().trim().min(1)).min(1, "At least one OIDC scope is required"),
});

export const identityProviderCreateFormSchema = identityProviderFormSchema.extend({
  oidcClientSecret: z.string().trim().min(1, "Client secret is required"),
});

export type IdentityProviderFormValues = z.infer<typeof identityProviderFormSchema>;

export const scimDirectorySchema = z.object({
  id: z.string().optional().default(""),
  organizationId: z.string().optional().default(""),
  businessUnitId: z.string().optional().default(""),
  tenantSlug: z.string().default(""),
  enabled: z.boolean().default(true),
  createdAt: z.number().default(0),
  updatedAt: z.number().default(0),
});

export type SCIMDirectory = z.infer<typeof scimDirectorySchema>;

export const scimDirectoryListSchema = createLimitOffsetResponse(scimDirectorySchema);

export type SCIMDirectoryListResponse = z.infer<typeof scimDirectoryListSchema>;

export const scimDirectoriesSchema = z.array(scimDirectorySchema);

export const scimDirectoryFormSchema = scimDirectorySchema.extend({
  tenantSlug: z.string().trim().min(1, "Tenant slug is required"),
});

export type SCIMDirectoryFormValues = z.infer<typeof scimDirectoryFormSchema>;

export const scimTokenSchema = z.object({
  id: z.string().optional().default(""),
  organizationId: z.string().optional().default(""),
  directoryId: z.string().optional().default(""),
  name: z.string().default(""),
  prefix: z.string().default(""),
  status: z.enum(["active", "revoked"]).default("active"),
  lastUsedAt: z.number().default(0),
  expiresAt: z.number().default(0),
  createdAt: z.number().default(0),
  updatedAt: z.number().default(0),
});

export type SCIMToken = z.infer<typeof scimTokenSchema>;

export const scimTokenCreateResponseSchema = scimTokenSchema.extend({
  token: z.string(),
});

export type SCIMTokenCreateResponse = z.infer<typeof scimTokenCreateResponseSchema>;

export const scimTokensSchema = z.array(scimTokenSchema);

export const scimGroupRoleMappingSchema = z.object({
  id: z.string().optional().default(""),
  organizationId: z.string().optional().default(""),
  businessUnitId: z.string().optional().default(""),
  directoryId: z.string().optional().default(""),
  externalGroupId: z.string().default(""),
  displayName: z.string().default(""),
  roleId: z.string().default(""),
  createdAt: z.number().default(0),
  updatedAt: z.number().default(0),
});

export type SCIMGroupRoleMapping = z.infer<typeof scimGroupRoleMappingSchema>;

export const scimGroupRoleMappingsSchema = z.array(scimGroupRoleMappingSchema);

export const scimGroupRoleMappingFormSchema = scimGroupRoleMappingSchema.extend({
  externalGroupId: z.string().trim().min(1, "External group ID is required"),
  displayName: z.string().default(""),
  roleId: z.string().trim().min(1, "Role is required"),
});

export type SCIMGroupRoleMappingFormValues = z.infer<typeof scimGroupRoleMappingFormSchema>;

export const provisioningAuditRecordSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  directoryId: z.string(),
  action: z.enum(["create", "update", "deactivate", "delete"]),
  resourceType: z.string(),
  externalId: z.string().optional().default(""),
  resourceId: z.string().optional().default(""),
  status: z.string(),
  errorMessage: z.string().optional().default(""),
  createdAt: z.number(),
});

export type ProvisioningAuditRecord = z.infer<typeof provisioningAuditRecordSchema>;

export const provisioningAuditRecordsSchema = z.array(provisioningAuditRecordSchema);

export const accessPolicySchema = z.object({
  id: z.string().optional().default(""),
  organizationId: z.string().optional().default(""),
  businessUnitId: z.string().optional().default(""),
  name: z.string().default(""),
  resource: z.string().default(""),
  operation: z.string().default("read"),
  effect: z.enum(["allow", "deny"]).default("deny"),
  priority: z.number().default(0),
  conditions: stringMapSchema,
  enabled: z.boolean().default(true),
  createdAt: z.number().default(0),
  updatedAt: z.number().default(0),
});

export type AccessPolicy = z.infer<typeof accessPolicySchema>;

export const accessPoliciesSchema = z.array(accessPolicySchema);

export const accessPolicyConditionRowSchema = z.object({
  id: z.string(),
  key: z.string().default(""),
  value: z.string().default(""),
});

export type AccessPolicyConditionRow = z.infer<typeof accessPolicyConditionRowSchema>;

export const accessPolicyFormSchema = accessPolicySchema
  .extend({
    name: z.string().trim().min(1, "Policy name is required"),
    resource: z.string().trim().min(1, "Resource is required"),
    operation: z.string().trim().min(1, "Operation is required"),
    priority: z.number().int().min(0, "Priority cannot be negative"),
    conditionRows: z.array(accessPolicyConditionRowSchema).default([]),
  })
  .superRefine((value, ctx) => {
    value.conditionRows.forEach((row, index) => {
      const hasKey = row.key.trim() !== "";
      const hasValue = row.value.trim() !== "";

      if (hasKey && !hasValue) {
        ctx.addIssue({
          code: "custom",
          message: "Condition value is required when a key is provided",
          path: ["conditionRows", index, "value"],
        });
      }

      if (hasValue && !hasKey) {
        ctx.addIssue({
          code: "custom",
          message: "Condition key is required when a value is provided",
          path: ["conditionRows", index, "key"],
        });
      }
    });
  });

export type AccessPolicyFormValues = z.infer<typeof accessPolicyFormSchema>;

export const authEventSchema = z.object({
  id: z.string(),
  userId: z.string().optional().default(""),
  organizationId: z.string().optional().default(""),
  businessUnitId: z.string().optional().default(""),
  identityProviderId: z.string().optional().default(""),
  provider: z.string(),
  outcome: z.enum(["success", "challenge", "denied", "failed"]),
  ipAddress: z.string().optional().default(""),
  userAgent: z.string().optional().default(""),
  authenticatorAal: z.number().default(1),
  federationFal: z.number().default(1),
  mfaState: z.string().optional().default(""),
  riskOutcome: z.enum(["allow", "challenge", "deny"]).default("allow"),
  riskSignals: stringArraySchema,
  errorCode: z.string().optional().default(""),
  occurredAt: z.number(),
  createdAt: z.number(),
});

export type AuthEvent = z.infer<typeof authEventSchema>;

export const authEventsSchema = z.array(authEventSchema);

export const riskDecisionSchema = z.object({
  id: z.string(),
  userId: z.string().optional().default(""),
  organizationId: z.string().optional().default(""),
  businessUnitId: z.string().optional().default(""),
  outcome: z.enum(["allow", "challenge", "deny"]),
  signals: stringArraySchema,
  reason: z.string().optional().default(""),
  createdAt: z.number(),
});

export type RiskDecision = z.infer<typeof riskDecisionSchema>;

export const riskDecisionsSchema = z.array(riskDecisionSchema);

export const externalIdentitySchema = z.object({
  id: z.string(),
  userId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  identityProviderId: z.string(),
  externalSubject: z.string(),
  externalUsername: z.string().optional().default(""),
  externalEmail: z.string().optional().default(""),
  rawClaims: stringMapSchema,
  lastLoginAt: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type ExternalIdentity = z.infer<typeof externalIdentitySchema>;

export const externalIdentitiesSchema = z.array(externalIdentitySchema);

export const mfaAuthenticatorSchema = z.object({
  id: z.string(),
  userId: z.string(),
  organizationId: z.string(),
  type: z.enum(["webauthn", "totp"]),
  name: z.string(),
  credentialId: z.string().optional().default(""),
  enabled: z.boolean(),
  verifiedAt: z.number().default(0),
  lastUsedAt: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type MFAAuthenticator = z.infer<typeof mfaAuthenticatorSchema>;

export const mfaAuthenticatorsSchema = z.array(mfaAuthenticatorSchema);

export const authProviderSummarySchema = z.object({
  id: z.string(),
  name: z.string(),
  provider: z.string(),
  protocol: z.string(),
  enabled: z.boolean(),
});

export type AuthProviderSummary = z.infer<typeof authProviderSummarySchema>;

export const authProviderSummariesSchema = z.array(authProviderSummarySchema);
