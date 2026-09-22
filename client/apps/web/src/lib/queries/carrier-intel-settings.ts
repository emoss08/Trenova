import { fetchCarrierIntelSettings } from "@/lib/graphql/carrier-intelligence";
import {
  fetchCarrierIntelCostEstimate,
  fetchCarrierIntelMonitoringStatus,
  fetchCarrierIntelUsage,
  type CarrierIntelCostEstimateParams,
} from "@/lib/graphql/carrier-intel-settings";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const carrierIntelSettings = createQueryKeys("carrierIntelSettings", {
  settings: () => ({
    queryKey: ["settings"],
    queryFn: async ({ signal }) => fetchCarrierIntelSettings({ signal }),
  }),
  monitoringStatus: () => ({
    queryKey: ["monitoring-status"],
    queryFn: async ({ signal }) => fetchCarrierIntelMonitoringStatus({ signal }),
  }),
  usage: (month?: number) => ({
    queryKey: ["usage", month ?? "current"],
    queryFn: async ({ signal }) => fetchCarrierIntelUsage(month, { signal }),
  }),
  /*
   * The two optional parameters fall back to a word rather than to null.
   *
   * query-key-factory's key type does not admit null, and a key that does not
   * typecheck takes the whole factory down with it — every read off any of
   * these queries then infers {} and the carrier-intelligence screens fail to
   * compile. The fallbacks are strings so they cannot be confused with a real
   * value: 0 days and false are both answers somebody might actually give.
   */
  costEstimate: (params: CarrierIntelCostEstimateParams) => ({
    queryKey: [
      "cost-estimate",
      params.policy,
      params.recentUsageDays ?? "any",
      params.includeOpenTenders ?? "unset",
    ],
    queryFn: async ({ signal }) => fetchCarrierIntelCostEstimate(params, { signal }),
  }),
});
