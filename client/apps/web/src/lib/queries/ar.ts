import {
  fetchArAgingSummary,
  fetchArAgingTrend,
  fetchArCashFlowForecast,
  fetchArCollectionPerformance,
  fetchArCollectionsWorklist,
  fetchArCustomerLedger,
  fetchArCustomerProfile,
  fetchArCustomerStatement,
  fetchArDashboardKpis,
  fetchArDsoTrend,
  fetchArOpenItems,
  fetchArPaymentStats,
  fetchArTopOverdueCustomers,
} from "@/lib/graphql/accounts-receivable";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const ar = createQueryKeys("ar", {
  agingSummary: (asOfDate?: number) => ({
    queryKey: ["agingSummary", asOfDate ?? 0],
    queryFn: async ({ signal }) => fetchArAgingSummary(asOfDate, { signal }),
  }),
  openItems: (options?: { customerId?: string; asOfDate?: number }) => ({
    queryKey: ["openItems", options],
    queryFn: async ({ signal }) => fetchArOpenItems(options, { signal }),
  }),
  customerLedger: (customerId: string) => ({
    queryKey: ["customerLedger", customerId],
    queryFn: async ({ signal }) => fetchArCustomerLedger(customerId, { signal }),
  }),
  customerStatement: (customerId: string, options?: { startDate?: number; asOfDate?: number }) => ({
    queryKey: ["customerStatement", customerId, options],
    queryFn: async ({ signal }) => fetchArCustomerStatement(customerId, options, { signal }),
  }),
  dashboardKpis: () => ({
    queryKey: ["dashboardKpis"],
    queryFn: ({ signal }) => fetchArDashboardKpis({ signal }),
  }),
  dsoTrend: (weeks?: number) => ({
    queryKey: ["dsoTrend", weeks ?? 0],
    queryFn: async ({ signal }) => fetchArDsoTrend(weeks, { signal }),
  }),
  agingTrend: (weeks?: number) => ({
    queryKey: ["agingTrend", weeks ?? 0],
    queryFn: async ({ signal }) => fetchArAgingTrend(weeks, { signal }),
  }),
  cashFlowForecast: (options?: { pastWeeks?: number; futureWeeks?: number }) => ({
    queryKey: ["cashFlowForecast", options],
    queryFn: async ({ signal }) => fetchArCashFlowForecast(options, { signal }),
  }),
  collectionPerformance: (periodDays?: number) => ({
    queryKey: ["collectionPerformance", periodDays ?? 0],
    queryFn: async ({ signal }) => fetchArCollectionPerformance(periodDays, { signal }),
  }),
  topOverdueCustomers: (limit?: number) => ({
    queryKey: ["topOverdueCustomers", limit ?? 0],
    queryFn: async ({ signal }) => fetchArTopOverdueCustomers(limit, { signal }),
  }),
  collectionsWorklist: (limit?: number) => ({
    queryKey: ["collectionsWorklist", limit ?? 0],
    queryFn: async ({ signal }) => fetchArCollectionsWorklist(limit, { signal }),
  }),
  customerProfile: (customerId: string) => ({
    queryKey: ["customerProfile", customerId],
    queryFn: async ({ signal }) => fetchArCustomerProfile(customerId, { signal }),
  }),
  paymentStats: () => ({
    queryKey: ["paymentStats"],
    queryFn: ({ signal }) => fetchArPaymentStats({ signal }),
  }),
});
