import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const organization = createQueryKeys("organization", {
  detail: (organizationId: string) => ({
    queryKey: ["detail", organizationId],
    queryFn: async () => apiService.organizationService.getByID(organizationId),
  }),
  logo: (organizationId: string) => ({
    queryKey: ["logo", organizationId],
    queryFn: async () =>
      apiService.organizationService.getLogoURL(organizationId),
  }),
  microsoftSSO: (organizationId: string) => ({
    queryKey: ["microsoft-sso", organizationId],
    queryFn: async () =>
      apiService.organizationService.getMicrosoftSSOConfig(organizationId),
  }),
  tenantLogin: (organizationSlug: string) => ({
    queryKey: ["tenant-login", organizationSlug],
    queryFn: async () =>
      apiService.organizationService.getTenantLoginMetadata(organizationSlug),
  }),
});
