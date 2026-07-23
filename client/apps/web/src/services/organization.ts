import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  accessPoliciesSchema,
  accessPolicySchema,
  authEventsSchema,
  externalIdentitiesSchema,
  identityProviderSchema,
  identityProvidersSchema,
  mfaAuthenticatorsSchema,
  provisioningAuditRecordsSchema,
  riskDecisionsSchema,
  scimDirectoryListSchema,
  scimDirectorySchema,
  scimGroupRoleMappingListSchema,
  scimGroupRoleMappingSchema,
  scimTokenCreateResponseSchema,
  scimTokensSchema,
  type AccessPolicy,
  type IdentityProvider,
  type SCIMDirectory,
  type SCIMDirectoryListResponse,
  type SCIMGroupRoleMapping,
  type SCIMGroupRoleMappingListResponse,
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

export class OrganizationService {
  readonly base_url = "/organizations";

  public async getByID(
    organizationId: string,
    opts: { includeState?: boolean; includeBu?: boolean } = {},
  ) {
    const query = new URLSearchParams();
    query.set("includeState", String(opts.includeState ?? true));
    query.set("includeBu", String(opts.includeBu ?? false));

    const response = await api.get<OrganizationSettings>(
      `${this.base_url}/${organizationId}?${query.toString()}`,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }

  public async update(organizationId: string, data: OrganizationSettings) {
    const response = await api.put<OrganizationSettings>(
      `${this.base_url}/${organizationId}`,
      data,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }

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

  public async getMicrosoftSSOConfig(organizationId: string) {
    const response = await api.get<MicrosoftSSOConfig>(
      `${this.base_url}/${organizationId}/microsoft-sso`,
    );

    return safeParse(microsoftSSOConfigSchema, response, "MicrosoftSSOConfig");
  }

  public async upsertMicrosoftSSOConfig(organizationId: string, data: MicrosoftSSOConfig) {
    const response = await api.put<MicrosoftSSOConfig>(
      `${this.base_url}/${organizationId}/microsoft-sso`,
      data,
    );

    return safeParse(microsoftSSOConfigSchema, response, "MicrosoftSSOConfig");
  }

  public async getOktaSSOConfig(organizationId: string) {
    const response = await api.get<OktaSSOConfig>(`${this.base_url}/${organizationId}/okta-sso`);

    return safeParse(oktaSSOConfigSchema, response, "OktaSSOConfig");
  }

  public async upsertOktaSSOConfig(organizationId: string, data: OktaSSOConfig) {
    const response = await api.put<OktaSSOConfig>(
      `${this.base_url}/${organizationId}/okta-sso`,
      data,
    );

    return safeParse(oktaSSOConfigSchema, response, "OktaSSOConfig");
  }

  public async getTenantLoginMetadata(organizationSlug: string) {
    const response = await api.get<TenantLoginMetadata>(`/auth/tenant/${organizationSlug}`);
    return safeParse(tenantLoginMetadataSchema, response, "TenantLoginMetadata");
  }

  public async listIdentityProviders(organizationId: string) {
    const response = await api.get(`${this.base_url}/${organizationId}/iam/identity-providers`);
    return safeParse(identityProvidersSchema, response, "IdentityProviders");
  }

  public async createIdentityProvider(organizationId: string, data: IdentityProvider) {
    const response = await api.post(
      `${this.base_url}/${organizationId}/iam/identity-providers`,
      data,
    );
    return safeParse(identityProviderSchema, response, "IdentityProvider");
  }

  public async updateIdentityProvider(organizationId: string, data: IdentityProvider) {
    const response = await api.put(
      `${this.base_url}/${organizationId}/iam/identity-providers/${data.id}`,
      data,
    );
    return safeParse(identityProviderSchema, response, "IdentityProvider");
  }

  public async deleteIdentityProvider(organizationId: string, providerId: string) {
    await api.delete(`${this.base_url}/${organizationId}/iam/identity-providers/${providerId}`);
  }

  public async listSCIMDirectories(
    organizationId: string,
    params?: { limit?: number; offset?: number },
  ) {
    const searchParams = new URLSearchParams();
    if (params?.limit != null) searchParams.set("limit", String(params.limit));
    if (params?.offset != null) searchParams.set("offset", String(params.offset));
    const qs = searchParams.toString();
    const response = await api.get<SCIMDirectoryListResponse>(
      `${this.base_url}/${organizationId}/iam/scim/directories${qs ? `?${qs}` : ""}`,
    );
    return safeParse(scimDirectoryListSchema, response, "SCIMDirectories");
  }

  public async createSCIMDirectory(organizationId: string, data: SCIMDirectory) {
    const response = await api.post(
      `${this.base_url}/${organizationId}/iam/scim/directories`,
      data,
    );
    return safeParse(scimDirectorySchema, response, "SCIMDirectory");
  }

  public async updateSCIMDirectory(organizationId: string, data: SCIMDirectory) {
    const response = await api.put(
      `${this.base_url}/${organizationId}/iam/scim/directories/${data.id}`,
      data,
    );
    return safeParse(scimDirectorySchema, response, "SCIMDirectory");
  }

  public async deleteSCIMDirectory(organizationId: string, directoryId: string) {
    await api.delete(`${this.base_url}/${organizationId}/iam/scim/directories/${directoryId}`);
  }

  public async listSCIMTokens(organizationId: string, directoryId: string) {
    const response = await api.get(
      `${this.base_url}/${organizationId}/iam/scim/directories/${directoryId}/tokens`,
    );
    return safeParse(scimTokensSchema, response, "SCIMTokens");
  }

  public async createSCIMToken(organizationId: string, directoryId: string, name: string) {
    const response = await api.post(
      `${this.base_url}/${organizationId}/iam/scim/directories/${directoryId}/tokens`,
      { name },
    );
    return safeParse(scimTokenCreateResponseSchema, response, "SCIMTokenCreateResponse");
  }

  public async revokeSCIMToken(organizationId: string, tokenId: string) {
    const response = await api.post(
      `${this.base_url}/${organizationId}/iam/scim/tokens/${tokenId}/revoke`,
    );
    return safeParse(scimTokenCreateResponseSchema.omit({ token: true }), response, "SCIMToken");
  }

  public async listSCIMGroupRoleMappings(
    organizationId: string,
    directoryId: string,
    params?: { limit?: number; offset?: number },
  ) {
    const searchParams = new URLSearchParams();
    if (params?.limit != null) searchParams.set("limit", String(params.limit));
    if (params?.offset != null) searchParams.set("offset", String(params.offset));
    const qs = searchParams.toString();
    const response = await api.get<SCIMGroupRoleMappingListResponse>(
      `${this.base_url}/${organizationId}/iam/scim/directories/${directoryId}/group-role-mappings${qs ? `?${qs}` : ""}`,
    );
    return safeParse(scimGroupRoleMappingListSchema, response, "SCIMGroupRoleMappings");
  }

  public async createSCIMGroupRoleMapping(
    organizationId: string,
    directoryId: string,
    data: SCIMGroupRoleMapping,
  ) {
    const response = await api.post(
      `${this.base_url}/${organizationId}/iam/scim/directories/${directoryId}/group-role-mappings`,
      data,
    );
    return safeParse(scimGroupRoleMappingSchema, response, "SCIMGroupRoleMapping");
  }

  public async updateSCIMGroupRoleMapping(
    organizationId: string,
    directoryId: string,
    data: SCIMGroupRoleMapping,
  ) {
    const response = await api.put(
      `${this.base_url}/${organizationId}/iam/scim/directories/${directoryId}/group-role-mappings/${data.id}`,
      data,
    );
    return safeParse(scimGroupRoleMappingSchema, response, "SCIMGroupRoleMapping");
  }

  public async deleteSCIMGroupRoleMapping(
    organizationId: string,
    directoryId: string,
    mappingId: string,
  ) {
    await api.delete(
      `${this.base_url}/${organizationId}/iam/scim/directories/${directoryId}/group-role-mappings/${mappingId}`,
    );
  }

  public async listProvisioningAudit(organizationId: string) {
    const response = await api.get(`${this.base_url}/${organizationId}/iam/provisioning-audit`);
    return safeParse(provisioningAuditRecordsSchema, response, "ProvisioningAuditRecords");
  }

  public async listAccessPolicies(organizationId: string) {
    const response = await api.get(`${this.base_url}/${organizationId}/iam/access-policies`);
    return safeParse(accessPoliciesSchema, response, "AccessPolicies");
  }

  public async createAccessPolicy(organizationId: string, data: AccessPolicy) {
    const response = await api.post(`${this.base_url}/${organizationId}/iam/access-policies`, data);
    return safeParse(accessPolicySchema, response, "AccessPolicy");
  }

  public async updateAccessPolicy(organizationId: string, data: AccessPolicy) {
    const response = await api.put(
      `${this.base_url}/${organizationId}/iam/access-policies/${data.id}`,
      data,
    );
    return safeParse(accessPolicySchema, response, "AccessPolicy");
  }

  public async deleteAccessPolicy(organizationId: string, policyId: string) {
    await api.delete(`${this.base_url}/${organizationId}/iam/access-policies/${policyId}`);
  }

  public async listAuthEvents(organizationId: string) {
    const response = await api.get(`${this.base_url}/${organizationId}/iam/auth-events`);
    return safeParse(authEventsSchema, response, "AuthEvents");
  }

  public async listRiskDecisions(organizationId: string) {
    const response = await api.get(`${this.base_url}/${organizationId}/iam/risk-decisions`);
    return safeParse(riskDecisionsSchema, response, "RiskDecisions");
  }

  public async listExternalIdentities(organizationId: string) {
    const response = await api.get(`${this.base_url}/${organizationId}/iam/external-identities`);
    return safeParse(externalIdentitiesSchema, response, "ExternalIdentities");
  }

  public async listMFAAuthenticators(organizationId: string) {
    const response = await api.get(`${this.base_url}/${organizationId}/iam/mfa-authenticators`);
    return safeParse(mfaAuthenticatorsSchema, response, "MFAAuthenticators");
  }
}
