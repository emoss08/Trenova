import {
  FactoringAgreementActivityDocument,
  FactoringCompanyCustomersDocument,
  FactoringCompanyOverviewDocument,
  FactoringCompanySchedulesDocument,
  FactoringFundingPreviewDocument,
  FactoringReserveHistoryDocument,
  type FactoringAgreementActivityQuery,
  type FactoringCompanyOverviewQuery,
  type FactoringCompanySchedulesQuery,
  type FactoringCustomerNoticeStage,
  type FactoringFundingPreviewInput,
  type FactoringFundingPreviewQuery,
  type FactoringReserveHistoryQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

type RequestOptions = { signal?: AbortSignal };

export type FactoringCompanyOverview = FactoringCompanyOverviewQuery["factoringCompanyOverview"];
export type FactoringAttention = FactoringCompanyOverview["attention"][number];
export type FactoringOverviewSegment = FactoringCompanyOverview["summary"][number];
export type FactoringPacePoint = FactoringCompanyOverview["pace"][number];
export type FactoringReserveDay = FactoringReserveHistoryQuery["factoringReserveHistory"][number];
export type FactoringAgreementActivity =
  FactoringAgreementActivityQuery["factoringAgreementActivity"];
export type FactoringFundingPreview = FactoringFundingPreviewQuery["factoringFundingPreview"];
export type FactoringCompanySchedule =
  FactoringCompanySchedulesQuery["factoringSubmissions"]["edges"][number]["node"];

/** How far back the reserve chart reads. */
export const FACTORING_RESERVE_HISTORY_DAYS = 90;
/** The schedules the delivery tab lists; the factoring workspace holds the rest. */
export const FACTORING_COMPANY_SCHEDULES = 5;
/** The customers table's query key, so a change to a notice can refresh it. */
export const FACTORING_COMPANY_CUSTOMER_KEY = "factoring-company-customer-list";

/** The customers assigned to one factor, as a data table, narrowed to a notice stage when given. */
export function createFactoringCompanyCustomersTableConfig(
  factoringCompanyId: string,
  stage: FactoringCustomerNoticeStage | null,
) {
  return defineDataTableGraphQLConfig({
    document: FactoringCompanyCustomersDocument,
    operationName: "FactoringCompanyCustomers",
    connectionKey: "factoringCompanyCustomers",
    extraVariables: { factoringCompanyId, stage },
  });
}

export type FactoringCompanyCustomer = DataTableConfigRow<
  ReturnType<typeof createFactoringCompanyCustomersTableConfig>
>;

export async function fetchFactoringCompanyOverview(
  factoringCompanyId: string,
  options?: RequestOptions,
): Promise<FactoringCompanyOverview> {
  const data = await requestGraphQL({
    document: FactoringCompanyOverviewDocument,
    operationName: "FactoringCompanyOverview",
    variables: { factoringCompanyId },
    signal: options?.signal,
  });
  return data.factoringCompanyOverview;
}

export async function fetchFactoringReserveHistory(
  factoringCompanyId: string,
  options?: RequestOptions,
): Promise<FactoringReserveDay[]> {
  const data = await requestGraphQL({
    document: FactoringReserveHistoryDocument,
    operationName: "FactoringReserveHistory",
    variables: { factoringCompanyId, days: FACTORING_RESERVE_HISTORY_DAYS },
    signal: options?.signal,
  });
  return data.factoringReserveHistory;
}

export async function listFactoringCompanySchedules(
  factoringCompanyId: string,
  options?: RequestOptions,
): Promise<FactoringCompanySchedule[]> {
  const data = await requestGraphQL({
    document: FactoringCompanySchedulesDocument,
    operationName: "FactoringCompanySchedules",
    variables: { factoringCompanyId, first: FACTORING_COMPANY_SCHEDULES },
    signal: options?.signal,
  });
  return data.factoringSubmissions.edges.map((edge) => edge.node);
}

/** What an invoice would pay under these terms, worked out by the server exactly as funding does. */
export async function fetchFactoringFundingPreview(
  input: FactoringFundingPreviewInput,
  options?: RequestOptions,
): Promise<FactoringFundingPreview> {
  const data = await requestGraphQL({
    document: FactoringFundingPreviewDocument,
    operationName: "FactoringFundingPreview",
    variables: { input },
    signal: options?.signal,
  });
  return data.factoringFundingPreview;
}

/** What the schedules sent under one agreement carried. */
export async function fetchFactoringAgreementActivity(
  agreementId: string,
  options?: RequestOptions,
): Promise<FactoringAgreementActivity> {
  const data = await requestGraphQL({
    document: FactoringAgreementActivityDocument,
    operationName: "FactoringAgreementActivity",
    variables: { agreementId },
    signal: options?.signal,
  });
  return data.factoringAgreementActivity;
}
