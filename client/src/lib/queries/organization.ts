import { apiService } from "@/services/api";
import { getOrganizationSettingsGraphQL } from "@/lib/graphql/organization";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const organization = createQueryKeys("organization", {
  detail: (organizationId: string) => ({
    queryKey: ["detail", organizationId],
    queryFn: async () => getOrganizationSettingsGraphQL(organizationId),
  }),
  logo: (organizationId: string) => ({
    queryKey: ["logo", organizationId],
    queryFn: async () => apiService.organizationService.getLogoURL(organizationId),
  }),
  microsoftSSO: (organizationId: string) => ({
    queryKey: ["microsoft-sso", organizationId],
    queryFn: async () => apiService.organizationService.getMicrosoftSSOConfig(organizationId),
  }),
  oktaSSO: (organizationId: string) => ({
    queryKey: ["okta-sso", organizationId],
    queryFn: async () => apiService.organizationService.getOktaSSOConfig(organizationId),
  }),
  tenantLogin: (organizationSlug: string) => ({
    queryKey: ["tenant-login", organizationSlug],
    queryFn: async () => apiService.organizationService.getTenantLoginMetadata(organizationSlug),
  }),
  identityProviders: (organizationId: string) => ({
    queryKey: ["identity-providers", organizationId],
    queryFn: async () => apiService.organizationService.listIdentityProviders(organizationId),
  }),
  scimDirectories: (organizationId: string) => ({
    queryKey: ["scim-directories", organizationId],
    queryFn: async () =>
      apiService.organizationService.listSCIMDirectories(organizationId, {
        limit: 100,
        offset: 0,
      }),
  }),
  provisioningAudit: (organizationId: string) => ({
    queryKey: ["provisioning-audit", organizationId],
    queryFn: async () => apiService.organizationService.listProvisioningAudit(organizationId),
  }),
  accessPolicies: (organizationId: string) => ({
    queryKey: ["access-policies", organizationId],
    queryFn: async () => apiService.organizationService.listAccessPolicies(organizationId),
  }),
  authEvents: (organizationId: string) => ({
    queryKey: ["auth-events", organizationId],
    queryFn: async () => apiService.organizationService.listAuthEvents(organizationId),
  }),
  riskDecisions: (organizationId: string) => ({
    queryKey: ["risk-decisions", organizationId],
    queryFn: async () => apiService.organizationService.listRiskDecisions(organizationId),
  }),
  externalIdentities: (organizationId: string) => ({
    queryKey: ["external-identities", organizationId],
    queryFn: async () => apiService.organizationService.listExternalIdentities(organizationId),
  }),
  mfaAuthenticators: (organizationId: string) => ({
    queryKey: ["mfa-authenticators", organizationId],
    queryFn: async () => apiService.organizationService.listMFAAuthenticators(organizationId),
  }),
});
