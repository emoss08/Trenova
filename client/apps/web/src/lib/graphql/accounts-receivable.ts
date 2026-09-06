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
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

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
export type ARCollectionPerformance = ArCollectionPerformanceQuery["arCollectionPerformance"];
export type ARTopOverdueCustomer = ArTopOverdueCustomersQuery["arTopOverdueCustomers"][number];
export type ARWorklistItem = ArCollectionsWorklistQuery["arCollectionsWorklist"][number];
export type ARCustomerProfile = ArCustomerProfileQuery["arCustomerProfile"];
export type ARPaymentStats = ArPaymentStatsQuery["arPaymentStats"];
export type ARCustomerSnapshot = ARCustomerProfile["snapshot"];

export async function fetchArAgingSummary(asOfDate?: number, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: ArAgingSummaryDocument,
    operationName: "ArAgingSummary",
    variables: { asOfDate },
    signal: options?.signal,
  });
  return data.arAgingSummary;
}

export async function fetchArOpenItems(
  options?: { customerId?: string; asOfDate?: number },
  requestOptions?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArOpenItemsDocument,
    operationName: "ArOpenItems",
    variables: {
      customerId: options?.customerId,
      asOfDate: options?.asOfDate,
    },
    signal: requestOptions?.signal,
  });
  return data.arOpenItems;
}

export async function fetchArCustomerLedger(
  customerId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArCustomerLedgerDocument,
    operationName: "ArCustomerLedger",
    variables: { customerId },
    signal: options?.signal,
  });
  return data.arCustomerLedger;
}

export async function fetchArCustomerStatement(
  customerId: string,
  options?: { startDate?: number; asOfDate?: number },
  requestOptions?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArCustomerStatementDocument,
    operationName: "ArCustomerStatement",
    variables: {
      customerId,
      startDate: options?.startDate,
      asOfDate: options?.asOfDate,
    },
    signal: requestOptions?.signal,
  });
  return data.arCustomerStatement;
}

export async function fetchArDashboardKpis(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: ArDashboardKpisDocument,
    operationName: "ArDashboardKpis",
    variables: {},
    signal: options?.signal,
  });
  return data.arDashboardKpis;
}

export async function fetchArDsoTrend(weeks?: number, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: ArDsoTrendDocument,
    operationName: "ArDsoTrend",
    variables: { weeks },
    signal: options?.signal,
  });
  return data.arDsoTrend;
}

export async function fetchArAgingTrend(weeks?: number, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: ArAgingTrendDocument,
    operationName: "ArAgingTrend",
    variables: { weeks },
    signal: options?.signal,
  });
  return data.arAgingTrend;
}

export async function fetchArCashFlowForecast(
  options?: {
    pastWeeks?: number;
    futureWeeks?: number;
  },
  requestOptions?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArCashFlowForecastDocument,
    operationName: "ArCashFlowForecast",
    variables: {
      pastWeeks: options?.pastWeeks,
      futureWeeks: options?.futureWeeks,
    },
    signal: requestOptions?.signal,
  });
  return data.arCashFlowForecast;
}

export async function fetchArCollectionPerformance(
  periodDays?: number,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArCollectionPerformanceDocument,
    operationName: "ArCollectionPerformance",
    variables: { periodDays },
    signal: options?.signal,
  });
  return data.arCollectionPerformance;
}

export async function fetchArTopOverdueCustomers(
  limit?: number,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArTopOverdueCustomersDocument,
    operationName: "ArTopOverdueCustomers",
    variables: { limit },
    signal: options?.signal,
  });
  return data.arTopOverdueCustomers;
}

export async function fetchArCollectionsWorklist(
  limit?: number,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArCollectionsWorklistDocument,
    operationName: "ArCollectionsWorklist",
    variables: { limit },
    signal: options?.signal,
  });
  return data.arCollectionsWorklist;
}

export async function fetchArPaymentStats(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: ArPaymentStatsDocument,
    operationName: "ArPaymentStats",
    variables: {},
    signal: options?.signal,
  });
  return data.arPaymentStats;
}

export async function fetchArCustomerProfile(
  customerId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: ArCustomerProfileDocument,
    operationName: "ArCustomerProfile",
    variables: { customerId },
    signal: options?.signal,
  });
  return data.arCustomerProfile;
}
