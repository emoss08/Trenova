import {
  AccessPoliciesDocument,
  AccessSettingsDocument,
  CreateAccessPolicyDocument,
  CreateIdentityProviderDocument,
  CreateScimDirectoryDocument,
  CreateScimGroupRoleMappingDocument,
  CreateScimTokenDocument,
  DeleteAccessPolicyDocument,
  DeleteIdentityProviderDocument,
  DeleteScimDirectoryDocument,
  DeleteScimGroupRoleMappingDocument,
  IdentityProvidersDocument,
  MicrosoftSsoConfigDocument,
  OktaSsoConfigDocument,
  ProvisioningAuditRecordsDocument,
  RecentAuthEventsDocument,
  RecentRiskDecisionsDocument,
  RevokeScimTokenDocument,
  ScimTokensDocument,
  SecurityBriefingDocument,
  UpdateAccessPolicyDocument,
  UpdateAccessSettingsDocument,
  UpdateIdentityProviderDocument,
  UpdateScimDirectoryDocument,
  UpdateScimGroupRoleMappingDocument,
  UpsertMicrosoftSsoConfigDocument,
  UpsertOktaSsoConfigDocument,
  type AccessPolicyInput,
  type IdentityProviderInput,
} from "@trenova/graphql/generated/graphql";
import { api } from "@trenova/shared/lib/api";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  accessPoliciesSchema,
  accessPolicySchema,
  accessSettingsSchema,
  authEventsSchema,
  identityProviderSchema,
  identityProvidersSchema,
  provisioningAuditRecordsSchema,
  riskDecisionsSchema,
  scimDirectorySchema,
  scimGroupRoleMappingSchema,
  scimTokenCreateResponseSchema,
  scimTokenSchema,
  scimTokensSchema,
  securityBriefingSchema,
  type AccessPolicy,
  type IdentityProvider,
  type SCIMDirectory,
  type SCIMGroupRoleMapping,
} from "@trenova/shared/types/iam";
import {
  microsoftSSOConfigSchema,
  oktaSSOConfigSchema,
  organizationLogoUrlResponseSchema,
  organizationSettingsSchema,
  tenantLoginMetadataSchema,
  type MicrosoftSSOConfig,
  type OktaSSOConfig,
  type OrganizationSettings,
  type TenantLoginMetadata,
} from "@trenova/shared/types/organization";

type RequestOptions = { signal?: AbortSignal };

/** How many of the latest sign-ins and risk decisions the briefing reads; it shows six. */
const RECENT_ACTIVITY_PAGE = 10;

function identityProviderInput(data: IdentityProvider): IdentityProviderInput {
  return {
    name: data.name,
    slug: data.slug,
    protocol: data.protocol,
    enabled: data.enabled,
    enforceSso: data.enforceSso,
    autoProvision: data.autoProvision,
    allowFederatedMfa: data.allowFederatedMfa,
    federatedMfaAcrValues: data.federatedMfaAcrValues,
    allowedDomains: data.allowedDomains,
    attributeMap: data.attributeMap,
    oidcIssuerUrl: data.oidcIssuerUrl,
    oidcClientId: data.oidcClientId,
    oidcClientSecret: data.oidcClientSecret,
    oidcRedirectUrl: data.oidcRedirectUrl,
    oidcScopes: data.oidcScopes,
    version: data.version,
  };
}

function accessPolicyInput(data: AccessPolicy): AccessPolicyInput {
  return {
    name: data.name,
    resource: data.resource,
    operation: data.operation,
    effect: data.effect,
    priority: data.priority,
    conditions: data.conditions,
    enabled: data.enabled,
  };
}

function scimGroupRoleMappingInput(data: SCIMGroupRoleMapping) {
  return {
    externalGroupId: data.externalGroupId,
    displayName: data.displayName,
    roleId: data.roleId,
    ...("version" in data && typeof data.version === "number" ? { version: data.version } : {}),
  };
}

export class OrganizationService {
  readonly base_url = "/organizations";

  public async uploadLogo(organizationId: string, file: File) {
    const formData = new FormData();
    formData.append("file", file);

    const response = await api.upload<OrganizationSettings>(
      `${this.base_url}/${organizationId}/logo`,
      formData,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }

  public async getLogoURL(organizationId: string) {
    const response = await api.get<{ url: string }>(`${this.base_url}/${organizationId}/logo`);

    return (await safeParse(organizationLogoUrlResponseSchema, response, "OrganizationLogoUrl"))
      .url;
  }

  public async deleteLogo(organizationId: string) {
    const response = await api.delete<OrganizationSettings>(
      `${this.base_url}/${organizationId}/logo`,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }

  public async getMicrosoftSSOConfig(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ microsoftSsoConfig: unknown }>({
      document: MicrosoftSsoConfigDocument,
      operationName: "MicrosoftSSOConfig",
      signal: options.signal,
    });
    return safeParse(microsoftSSOConfigSchema, data.microsoftSsoConfig, "MicrosoftSSOConfig");
  }

  public async upsertMicrosoftSSOConfig(config: MicrosoftSSOConfig) {
    const data = await requestGraphQL<{ upsertMicrosoftSsoConfig: unknown }>({
      document: UpsertMicrosoftSsoConfigDocument,
      operationName: "UpsertMicrosoftSSOConfig",
      variables: {
        input: {
          enabled: config.enabled,
          enforceSso: config.enforceSso,
          tenantId: config.tenantId,
          clientId: config.clientId,
          clientSecret: config.clientSecret,
          redirectUrl: config.redirectUrl,
          allowedDomains: config.allowedDomains,
        },
      },
    });
    return safeParse(microsoftSSOConfigSchema, data.upsertMicrosoftSsoConfig, "MicrosoftSSOConfig");
  }

  public async getOktaSSOConfig(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ oktaSsoConfig: unknown }>({
      document: OktaSsoConfigDocument,
      operationName: "OktaSSOConfig",
      signal: options.signal,
    });
    return safeParse(oktaSSOConfigSchema, data.oktaSsoConfig, "OktaSSOConfig");
  }

  public async upsertOktaSSOConfig(config: OktaSSOConfig) {
    const data = await requestGraphQL<{ upsertOktaSsoConfig: unknown }>({
      document: UpsertOktaSsoConfigDocument,
      operationName: "UpsertOktaSSOConfig",
      variables: {
        input: {
          enabled: config.enabled,
          enforceSso: config.enforceSso,
          issuerUrl: config.issuerUrl,
          clientId: config.clientId,
          clientSecret: config.clientSecret,
          redirectUrl: config.redirectUrl,
          scopes: config.scopes,
          allowedDomains: config.allowedDomains,
        },
      },
    });
    return safeParse(oktaSSOConfigSchema, data.upsertOktaSsoConfig, "OktaSSOConfig");
  }

  public async getTenantLoginMetadata(organizationSlug: string) {
    const response = await api.get<TenantLoginMetadata>(`/auth/tenant/${organizationSlug}`);
    return safeParse(tenantLoginMetadataSchema, response, "TenantLoginMetadata");
  }

  public async listIdentityProviders(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ identityProviders: unknown }>({
      document: IdentityProvidersDocument,
      operationName: "IdentityProviders",
      signal: options.signal,
    });
    return safeParse(identityProvidersSchema, data.identityProviders, "IdentityProviders");
  }

  public async createIdentityProvider(provider: IdentityProvider) {
    const data = await requestGraphQL<{ createIdentityProvider: unknown }>({
      document: CreateIdentityProviderDocument,
      operationName: "CreateIdentityProvider",
      variables: { input: identityProviderInput(provider) },
    });
    return safeParse(identityProviderSchema, data.createIdentityProvider, "IdentityProvider");
  }

  public async updateIdentityProvider(provider: IdentityProvider) {
    const data = await requestGraphQL<{ updateIdentityProvider: unknown }>({
      document: UpdateIdentityProviderDocument,
      operationName: "UpdateIdentityProvider",
      variables: { id: provider.id, input: identityProviderInput(provider) },
    });
    return safeParse(identityProviderSchema, data.updateIdentityProvider, "IdentityProvider");
  }

  public async deleteIdentityProvider(providerId: string) {
    await requestGraphQL({
      document: DeleteIdentityProviderDocument,
      operationName: "DeleteIdentityProvider",
      variables: { id: providerId },
    });
  }

  public async createSCIMDirectory(directory: SCIMDirectory) {
    const data = await requestGraphQL<{ createSCIMDirectory: unknown }>({
      document: CreateScimDirectoryDocument,
      operationName: "CreateSCIMDirectory",
      variables: { input: { tenantSlug: directory.tenantSlug, enabled: directory.enabled } },
    });
    return safeParse(scimDirectorySchema, data.createSCIMDirectory, "SCIMDirectory");
  }

  public async updateSCIMDirectory(directory: SCIMDirectory) {
    const data = await requestGraphQL<{ updateSCIMDirectory: unknown }>({
      document: UpdateScimDirectoryDocument,
      operationName: "UpdateSCIMDirectory",
      variables: {
        id: directory.id,
        input: { tenantSlug: directory.tenantSlug, enabled: directory.enabled },
      },
    });
    return safeParse(scimDirectorySchema, data.updateSCIMDirectory, "SCIMDirectory");
  }

  public async deleteSCIMDirectory(directoryId: string) {
    await requestGraphQL({
      document: DeleteScimDirectoryDocument,
      operationName: "DeleteSCIMDirectory",
      variables: { id: directoryId },
    });
  }

  public async listSCIMTokens(directoryId: string, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ scimTokens: unknown }>({
      document: ScimTokensDocument,
      operationName: "SCIMTokens",
      variables: { directoryId },
      signal: options.signal,
    });
    return safeParse(scimTokensSchema, data.scimTokens, "SCIMTokens");
  }

  public async createSCIMToken(directoryId: string, name: string) {
    const data = await requestGraphQL<{
      createSCIMToken: { token: string; scimToken: Record<string, unknown> };
    }>({
      document: CreateScimTokenDocument,
      operationName: "CreateSCIMToken",
      variables: { directoryId, input: { name } },
    });
    return safeParse(
      scimTokenCreateResponseSchema,
      { ...data.createSCIMToken.scimToken, token: data.createSCIMToken.token },
      "SCIMTokenCreateResponse",
    );
  }

  public async revokeSCIMToken(tokenId: string) {
    const data = await requestGraphQL<{ revokeSCIMToken: unknown }>({
      document: RevokeScimTokenDocument,
      operationName: "RevokeSCIMToken",
      variables: { id: tokenId },
    });
    return safeParse(scimTokenSchema, data.revokeSCIMToken, "SCIMToken");
  }

  public async createSCIMGroupRoleMapping(directoryId: string, mapping: SCIMGroupRoleMapping) {
    const data = await requestGraphQL<{ createSCIMGroupRoleMapping: unknown }>({
      document: CreateScimGroupRoleMappingDocument,
      operationName: "CreateSCIMGroupRoleMapping",
      variables: { directoryId, input: scimGroupRoleMappingInput(mapping) },
    });
    return safeParse(
      scimGroupRoleMappingSchema,
      data.createSCIMGroupRoleMapping,
      "SCIMGroupRoleMapping",
    );
  }

  public async updateSCIMGroupRoleMapping(mapping: SCIMGroupRoleMapping) {
    const data = await requestGraphQL<{ updateSCIMGroupRoleMapping: unknown }>({
      document: UpdateScimGroupRoleMappingDocument,
      operationName: "UpdateSCIMGroupRoleMapping",
      variables: { id: mapping.id, input: scimGroupRoleMappingInput(mapping) },
    });
    return safeParse(
      scimGroupRoleMappingSchema,
      data.updateSCIMGroupRoleMapping,
      "SCIMGroupRoleMapping",
    );
  }

  public async deleteSCIMGroupRoleMapping(mappingId: string) {
    await requestGraphQL({
      document: DeleteScimGroupRoleMappingDocument,
      operationName: "DeleteSCIMGroupRoleMapping",
      variables: { id: mappingId },
    });
  }

  public async listProvisioningAudit(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ provisioningAuditRecords: unknown }>({
      document: ProvisioningAuditRecordsDocument,
      operationName: "ProvisioningAuditRecords",
      variables: {},
      signal: options.signal,
    });
    return safeParse(
      provisioningAuditRecordsSchema,
      data.provisioningAuditRecords,
      "ProvisioningAuditRecords",
    );
  }

  public async listAccessPolicies(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ accessPolicies: unknown }>({
      document: AccessPoliciesDocument,
      operationName: "AccessPolicies",
      signal: options.signal,
    });
    return safeParse(accessPoliciesSchema, data.accessPolicies, "AccessPolicies");
  }

  public async createAccessPolicy(policy: AccessPolicy) {
    const data = await requestGraphQL<{ createAccessPolicy: unknown }>({
      document: CreateAccessPolicyDocument,
      operationName: "CreateAccessPolicy",
      variables: { input: accessPolicyInput(policy) },
    });
    return safeParse(accessPolicySchema, data.createAccessPolicy, "AccessPolicy");
  }

  public async updateAccessPolicy(policy: AccessPolicy) {
    const data = await requestGraphQL<{ updateAccessPolicy: unknown }>({
      document: UpdateAccessPolicyDocument,
      operationName: "UpdateAccessPolicy",
      variables: { id: policy.id, input: accessPolicyInput(policy) },
    });
    return safeParse(accessPolicySchema, data.updateAccessPolicy, "AccessPolicy");
  }

  public async deleteAccessPolicy(policyId: string) {
    await requestGraphQL({
      document: DeleteAccessPolicyDocument,
      operationName: "DeleteAccessPolicy",
      variables: { id: policyId },
    });
  }

  public async getAccessSettings(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ accessSettings: unknown }>({
      document: AccessSettingsDocument,
      operationName: "AccessSettings",
      signal: options.signal,
    });
    return safeParse(accessSettingsSchema, data.accessSettings, "AccessSettings");
  }

  public async updateAccessSettings(settings: { requireRoleActivation: boolean; version: number }) {
    const data = await requestGraphQL<{ updateAccessSettings: unknown }>({
      document: UpdateAccessSettingsDocument,
      operationName: "UpdateAccessSettings",
      variables: { input: settings },
    });
    return safeParse(accessSettingsSchema, data.updateAccessSettings, "AccessSettings");
  }

  public async listAuthEvents(options: RequestOptions = {}) {
    const data = await requestGraphQL({
      document: RecentAuthEventsDocument,
      operationName: "RecentAuthEvents",
      variables: { first: RECENT_ACTIVITY_PAGE },
      signal: options.signal,
    });
    return safeParse(
      authEventsSchema,
      data.authEvents.edges.map((edge) => edge.node),
      "AuthEvents",
    );
  }

  public async listRiskDecisions(options: RequestOptions = {}) {
    const data = await requestGraphQL({
      document: RecentRiskDecisionsDocument,
      operationName: "RecentRiskDecisions",
      variables: { first: RECENT_ACTIVITY_PAGE },
      signal: options.signal,
    });
    return safeParse(
      riskDecisionsSchema,
      data.riskDecisions.edges.map((edge) => edge.node),
      "RiskDecisions",
    );
  }

  public async getSecurityBriefing(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ securityBriefing: unknown }>({
      document: SecurityBriefingDocument,
      operationName: "SecurityBriefing",
      signal: options.signal,
    });
    return safeParse(securityBriefingSchema, data.securityBriefing, "SecurityBriefing");
  }
}
