import { apiService } from "@/services/api";
import { getOrganizationSettingsGraphQL } from "@/lib/graphql/organization";
import { listSCIMDirectoriesGraphQL } from "@/lib/graphql/scim-directory";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/** The most directories one read of the list asks for; the server serves at most 100 a page. */
const SCIM_DIRECTORY_FIRST_PAGE = 100;

export const organization = createQueryKeys("organization", {
  detail: (organizationId: string) => ({
    queryKey: ["detail", organizationId],
    queryFn: async ({ signal }) => getOrganizationSettingsGraphQL(organizationId, { signal }),
  }),
  logo: (organizationId: string) => ({
    queryKey: ["logo", organizationId],
    queryFn: async () => apiService.organizationService.getLogoURL(organizationId),
  }),
  microsoftSSO: (organizationId: string) => ({
    queryKey: ["microsoft-sso", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.getMicrosoftSSOConfig({ signal }),
  }),
  oktaSSO: (organizationId: string) => ({
    queryKey: ["okta-sso", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.getOktaSSOConfig({ signal }),
  }),
  tenantLogin: (organizationSlug: string) => ({
    queryKey: ["tenant-login", organizationSlug],
    queryFn: async () => apiService.organizationService.getTenantLoginMetadata(organizationSlug),
  }),
  accessSettings: (organizationId: string) => ({
    queryKey: ["access-settings", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.getAccessSettings({ signal }),
  }),
  identityProviders: (organizationId: string) => ({
    queryKey: ["identity-providers", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.listIdentityProviders({ signal }),
  }),
  scimDirectories: (organizationId: string) => ({
    queryKey: ["scim-directories", organizationId],
    queryFn: async ({ signal }) =>
      listSCIMDirectoriesGraphQL({ first: SCIM_DIRECTORY_FIRST_PAGE }, { signal }),
  }),
  provisioningAudit: (organizationId: string) => ({
    queryKey: ["provisioning-audit", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.listProvisioningAudit({ signal }),
  }),
  accessPolicies: (organizationId: string) => ({
    queryKey: ["access-policies", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.listAccessPolicies({ signal }),
  }),
  authEvents: (organizationId: string) => ({
    queryKey: ["auth-events", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.listAuthEvents({ signal }),
  }),
  riskDecisions: (organizationId: string) => ({
    queryKey: ["risk-decisions", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.listRiskDecisions({ signal }),
  }),
  securityBriefing: (organizationId: string) => ({
    queryKey: ["security-briefing", organizationId],
    queryFn: async ({ signal }) => apiService.organizationService.getSecurityBriefing({ signal }),
  }),
});
