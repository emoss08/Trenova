import { fetchCustomerRateAgreements, fetchDeskCustomer } from "@/lib/graphql/customer-card";
import { getCustomerTrackingProfile } from "@/lib/graphql/customer-tracking";
import { ShipmentCustomerBillingRulesDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const customer = createQueryKeys("customer", {
  /**
   * What a shipment form shows about the customers it bills (BOL requirement, credit
   * standing), every customer in one read. Keyed by the sorted ids, so the same payers
   * share one answer whatever order the form names them in.
   */
  shipmentBillingRules: (customerIds: readonly string[]) => {
    const ids = [...customerIds].sort();
    return {
      queryKey: ["shipmentBillingRules", ids],
      queryFn: async ({ signal }: { signal?: AbortSignal }) =>
        (
          await requestGraphQL({
            document: ShipmentCustomerBillingRulesDocument,
            operationName: "ShipmentCustomerBillingRules",
            variables: { customerIds: ids },
            signal,
          })
        ).shipmentCustomerBillingRules,
    };
  },
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
