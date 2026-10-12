import {
  fetchCustomerRateAgreements,
  fetchCustomerRecentShipments,
  fetchDeskCustomer,
} from "@/lib/graphql/customer-card";
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
  recentShipments: (customerId: string, limit: number) => ({
    queryKey: ["recentShipments", customerId, limit],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchCustomerRecentShipments(customerId, limit, { signal }),
  }),
  rateAgreements: (customerId: string, limit: number) => ({
    queryKey: ["rateAgreements", customerId, limit],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchCustomerRateAgreements(customerId, limit, { signal }),
  }),
});
