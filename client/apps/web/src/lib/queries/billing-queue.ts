import {
  getBillingQueueFormulaTemplateGraphQL,
  getBillingQueueRateAgreementGraphQL,
  getBillingQueueShipmentGraphQL,
} from "@/lib/graphql/billing-queue";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const billingQueue = createQueryKeys("billingQueue", {
  stats: (dayStart?: number) => ({
    queryKey: ["stats", dayStart ?? 0],
    queryFn: async ({ signal }) => apiService.billingQueueService.getStats(dayStart, { signal }),
  }),
  recommendations: () => ({
    queryKey: ["recommendations"],
    queryFn: async ({ signal }) => apiService.billingQueueService.getRecommendations({ signal }),
  }),
  assist: () => ({
    queryKey: ["assist"],
    queryFn: async ({ signal }) => apiService.billingQueueService.getAssist({ signal }),
  }),
  brief: () => ({
    queryKey: ["brief"],
    queryFn: async ({ signal }) => apiService.billingQueueService.getBrief({ signal }),
  }),
  get: (itemId: string) => ({
    queryKey: ["get", itemId],
    queryFn: async ({ signal }) =>
      apiService.billingQueueService.getById(itemId, { expandShipmentDetails: "true" }, { signal }),
  }),
  /** The item's shipment as the Shipment tab reads it, fetched when the tab opens. */
  shipment: (shipmentId: string) => ({
    queryKey: ["shipment", shipmentId],
    queryFn: async ({ signal }) => getBillingQueueShipmentGraphQL(shipmentId, { signal }),
  }),
  /** The agreement behind "Rated by", fetched when its terms are opened. */
  rateAgreement: (agreementId: string) => ({
    queryKey: ["rateAgreement", agreementId],
    queryFn: async ({ signal }) => getBillingQueueRateAgreementGraphQL(agreementId, { signal }),
  }),
  /** The template behind "Rated by", fetched when its terms are opened. */
  formulaTemplate: (templateId: string) => ({
    queryKey: ["formulaTemplate", templateId],
    queryFn: async ({ signal }) => getBillingQueueFormulaTemplateGraphQL(templateId, { signal }),
  }),
  neighbors: (itemId: string, params: Record<string, string> = {}) => ({
    queryKey: ["neighbors", itemId, params],
    queryFn: async () => apiService.billingQueueService.getNeighbors(itemId, params),
  }),
  activity: (itemId: string) => ({
    queryKey: ["activity", itemId],
    queryFn: async () => apiService.billingQueueService.getActivity(itemId),
  }),
  summaries: (itemIds: string[]) => ({
    queryKey: ["summaries", itemIds],
    queryFn: async () => apiService.billingQueueService.getSummaries(itemIds),
  }),
  bulkRun: (runId: string) => ({
    queryKey: ["bulkRun", runId],
    queryFn: async ({ signal }) => apiService.billingQueueService.getBulk(runId, { signal }),
  }),
});
