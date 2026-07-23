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
    queryFn: async () => fetchArAgingSummary(asOfDate),
  }),
  openItems: (options?: { customerId?: string; asOfDate?: number }) => ({
    queryKey: ["openItems", options],
    queryFn: async () => fetchArOpenItems(options),
  }),
  customerLedger: (customerId: string) => ({
    queryKey: ["customerLedger", customerId],
    queryFn: async () => fetchArCustomerLedger(customerId),
  }),
  customerStatement: (
    customerId: string,
    options?: { startDate?: number; asOfDate?: number },
  ) => ({
    queryKey: ["customerStatement", customerId, options],
    queryFn: async () => fetchArCustomerStatement(customerId, options),
  }),
  dashboardKpis: () => ({
    queryKey: ["dashboardKpis"],
    queryFn: fetchArDashboardKpis,
  }),
  dsoTrend: (weeks?: number) => ({
    queryKey: ["dsoTrend", weeks ?? 0],
    queryFn: async () => fetchArDsoTrend(weeks),
  }),
  agingTrend: (weeks?: number) => ({
    queryKey: ["agingTrend", weeks ?? 0],
    queryFn: async () => fetchArAgingTrend(weeks),
  }),
  cashFlowForecast: (options?: { pastWeeks?: number; futureWeeks?: number }) => ({
    queryKey: ["cashFlowForecast", options],
    queryFn: async () => fetchArCashFlowForecast(options),
  }),
  collectionPerformance: (periodDays?: number) => ({
    queryKey: ["collectionPerformance", periodDays ?? 0],
    queryFn: async () => fetchArCollectionPerformance(periodDays),
  }),
  topOverdueCustomers: (limit?: number) => ({
    queryKey: ["topOverdueCustomers", limit ?? 0],
    queryFn: async () => fetchArTopOverdueCustomers(limit),
  }),
  collectionsWorklist: (limit?: number) => ({
    queryKey: ["collectionsWorklist", limit ?? 0],
    queryFn: async () => fetchArCollectionsWorklist(limit),
  }),
  customerProfile: (customerId: string) => ({
    queryKey: ["customerProfile", customerId],
    queryFn: async () => fetchArCustomerProfile(customerId),
  }),
  paymentStats: () => ({
    queryKey: ["paymentStats"],
    queryFn: fetchArPaymentStats,
  }),
});
