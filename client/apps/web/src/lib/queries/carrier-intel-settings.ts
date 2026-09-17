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
  costEstimate: (params: CarrierIntelCostEstimateParams) => ({
    queryKey: [
      "cost-estimate",
      params.policy,
      params.recentUsageDays ?? null,
      params.includeOpenTenders ?? null,
    ],
    queryFn: async ({ signal }) => fetchCarrierIntelCostEstimate(params, { signal }),
  }),
});
