import {
  ArAgingSummaryDocument,
  ArAgingTrendDocument,
  ArCashFlowForecastDocument,
  ArCollectionPerformanceDocument,
  ArCollectionsWorklistDocument,
  ArCustomerLedgerDocument,
  ArCustomerProfileDocument,
  ArCustomerStatementDocument,
  ArDashboardKpisDocument,
  ArDsoTrendDocument,
  ArOpenItemsDocument,
  ArPaymentStatsDocument,
  ArTopOverdueCustomersDocument,
  type ArAgingSummaryQuery,
  type ArAgingTrendQuery,
  type ArCashFlowForecastQuery,
  type ArCollectionPerformanceQuery,
  type ArCollectionsWorklistQuery,
  type ArCustomerLedgerQuery,
  type ArCustomerProfileQuery,
  type ArCustomerStatementQuery,
  type ArDashboardKpisQuery,
  type ArDsoTrendQuery,
  type ArOpenItemsQuery,
  type ArPaymentStatsQuery,
  type ArTopOverdueCustomersQuery,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";

export type ARAgingSummary = ArAgingSummaryQuery["arAgingSummary"];
export type ARAgingRow = ARAgingSummary["rows"][number];
export type ARAgingBuckets = ARAgingSummary["totals"];
export type AROpenItem = ArOpenItemsQuery["arOpenItems"][number];
export type ARLedgerEntry = ArCustomerLedgerQuery["arCustomerLedger"][number];
export type ARCustomerStatement = ArCustomerStatementQuery["arCustomerStatement"];
export type ARStatementTransaction = ARCustomerStatement["transactions"][number];
export type ARDashboardKpis = ArDashboardKpisQuery["arDashboardKpis"];
export type ARDsoTrendPoint = ArDsoTrendQuery["arDsoTrend"][number];
export type ARAgingTrendPoint = ArAgingTrendQuery["arAgingTrend"][number];
export type ARCashFlowPoint = ArCashFlowForecastQuery["arCashFlowForecast"][number];
export type ARCollectionPerformance =
  ArCollectionPerformanceQuery["arCollectionPerformance"];
export type ARTopOverdueCustomer =
  ArTopOverdueCustomersQuery["arTopOverdueCustomers"][number];
export type ARWorklistItem = ArCollectionsWorklistQuery["arCollectionsWorklist"][number];
export type ARCustomerProfile = ArCustomerProfileQuery["arCustomerProfile"];
export type ARPaymentStats = ArPaymentStatsQuery["arPaymentStats"];
export type ARCustomerSnapshot = ARCustomerProfile["snapshot"];

export async function fetchArAgingSummary(asOfDate?: number) {
  const data = await requestGraphQL({
    document: ArAgingSummaryDocument,
    operationName: "ArAgingSummary",
    variables: { asOfDate },
  });
  return data.arAgingSummary;
}

export async function fetchArOpenItems(options?: {
  customerId?: string;
  asOfDate?: number;
}) {
  const data = await requestGraphQL({
    document: ArOpenItemsDocument,
    operationName: "ArOpenItems",
    variables: {
      customerId: options?.customerId,
      asOfDate: options?.asOfDate,
    },
  });
  return data.arOpenItems;
}

export async function fetchArCustomerLedger(customerId: string) {
  const data = await requestGraphQL({
    document: ArCustomerLedgerDocument,
    operationName: "ArCustomerLedger",
    variables: { customerId },
  });
  return data.arCustomerLedger;
}

export async function fetchArCustomerStatement(
  customerId: string,
  options?: { startDate?: number; asOfDate?: number },
) {
  const data = await requestGraphQL({
    document: ArCustomerStatementDocument,
    operationName: "ArCustomerStatement",
    variables: {
      customerId,
      startDate: options?.startDate,
      asOfDate: options?.asOfDate,
    },
  });
  return data.arCustomerStatement;
}

export async function fetchArDashboardKpis() {
  const data = await requestGraphQL({
    document: ArDashboardKpisDocument,
    operationName: "ArDashboardKpis",
    variables: {},
  });
  return data.arDashboardKpis;
}

export async function fetchArDsoTrend(weeks?: number) {
  const data = await requestGraphQL({
    document: ArDsoTrendDocument,
    operationName: "ArDsoTrend",
    variables: { weeks },
  });
  return data.arDsoTrend;
}

export async function fetchArAgingTrend(weeks?: number) {
  const data = await requestGraphQL({
    document: ArAgingTrendDocument,
    operationName: "ArAgingTrend",
    variables: { weeks },
  });
  return data.arAgingTrend;
}

export async function fetchArCashFlowForecast(options?: {
  pastWeeks?: number;
  futureWeeks?: number;
}) {
  const data = await requestGraphQL({
    document: ArCashFlowForecastDocument,
    operationName: "ArCashFlowForecast",
    variables: {
      pastWeeks: options?.pastWeeks,
      futureWeeks: options?.futureWeeks,
    },
  });
  return data.arCashFlowForecast;
}

export async function fetchArCollectionPerformance(periodDays?: number) {
  const data = await requestGraphQL({
    document: ArCollectionPerformanceDocument,
    operationName: "ArCollectionPerformance",
    variables: { periodDays },
  });
  return data.arCollectionPerformance;
}

export async function fetchArTopOverdueCustomers(limit?: number) {
  const data = await requestGraphQL({
    document: ArTopOverdueCustomersDocument,
    operationName: "ArTopOverdueCustomers",
    variables: { limit },
  });
  return data.arTopOverdueCustomers;
}

export async function fetchArCollectionsWorklist(limit?: number) {
  const data = await requestGraphQL({
    document: ArCollectionsWorklistDocument,
    operationName: "ArCollectionsWorklist",
    variables: { limit },
  });
  return data.arCollectionsWorklist;
}

export async function fetchArPaymentStats() {
  const data = await requestGraphQL({
    document: ArPaymentStatsDocument,
    operationName: "ArPaymentStats",
    variables: {},
  });
  return data.arPaymentStats;
}

export async function fetchArCustomerProfile(customerId: string) {
  const data = await requestGraphQL({
    document: ArCustomerProfileDocument,
    operationName: "ArCustomerProfile",
    variables: { customerId },
  });
  return data.arCustomerProfile;
}
