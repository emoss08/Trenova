import { fetchCustomerRateAgreements, fetchDeskCustomer } from "@/lib/graphql/customer-card";
import { getCustomerTrackingProfile } from "@/lib/graphql/customer-tracking";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const customer = createQueryKeys("customer", {
  getBillingProfile: (customerId: string) => ({
    queryKey: ["getBillingProfile", customerId],
    queryFn: async () => apiService.customerService.getBillingProfile(customerId),
  }),
  card: (customerId: string) => ({
    queryKey: ["card", customerId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskCustomer(customerId, { signal }),
  }),
  // The customer's tracking link and status email settings, or the defaults.
  trackingProfile: (customerId: string) => ({
    queryKey: ["trackingProfile", customerId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      getCustomerTrackingProfile(customerId, { signal }),
  }),
  rateAgreements: (customerId: string, limit: number) => ({
    queryKey: ["rateAgreements", customerId, limit],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchCustomerRateAgreements(customerId, limit, { signal }),
  }),
});
