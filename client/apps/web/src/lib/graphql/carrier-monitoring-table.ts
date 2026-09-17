import {
  CarrierIntelEventCarrierSummaryDocument,
  CarrierIntelEventInboxDocument,
  CarrierMonitoringEnrollmentTableDocument,
  type CarrierIntelEventCarrierSummaryQuery,
  type CarrierIntelEventFilterInput,
  type CarrierMonitoringEnrollmentFilterInput,
  type FieldFilterInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";
import type { CarrierIntelEvent } from "./carrier-intelligence";

export const CARRIER_INTEL_EVENT_LIST_KEY = "carrier-intel-event-list";
export const CARRIER_MONITORING_ENROLLMENT_LIST_KEY = "carrier-monitoring-enrollment-list";

export function carrierMonitoringEnrollmentTableGraphQLConfig(
  filter: CarrierMonitoringEnrollmentFilterInput | null,
) {
  return defineDataTableGraphQLConfig({
    document: CarrierMonitoringEnrollmentTableDocument,
    operationName: "CarrierMonitoringEnrollmentTable",
    connectionKey: "carrierMonitoringEnrollments",
    extraVariables: filter ? { filter } : undefined,
  });
}

export type CarrierMonitoringEnrollmentRow = DataTableConfigRow<
  ReturnType<typeof carrierMonitoringEnrollmentTableGraphQLConfig>
>;

export type CarrierIntelEventInboxVariables = {
  filter: CarrierIntelEventFilterInput;
  query: string | null;
  fieldFilters: FieldFilterInput[];
};

export type CarrierIntelEventInboxPage = {
  events: CarrierIntelEvent[];
  hasNextPage: boolean;
  endCursor: string | null;
};

export type FetchCarrierIntelEventInboxArgs = CarrierIntelEventInboxVariables & {
  first: number;
  after: string | null;
};

export async function fetchCarrierIntelEventInbox(
  { filter, query, fieldFilters, first, after }: FetchCarrierIntelEventInboxArgs,
  options?: { signal?: AbortSignal },
): Promise<CarrierIntelEventInboxPage> {
  const data = await requestGraphQL({
    document: CarrierIntelEventInboxDocument,
    operationName: "CarrierIntelEventInbox",
    variables: {
      input: {
        first,
        after,
        query,
        fieldFilters: fieldFilters.length > 0 ? fieldFilters : null,
      },
      filter,
    },
    signal: options?.signal,
  });
  const connection = data.carrierIntelEvents as unknown as UnmaskFragments<
    typeof data.carrierIntelEvents
  >;
  return {
    events: connection.edges.map((edge) => edge.node),
    hasNextPage: connection.pageInfo.hasNextPage,
    endCursor: connection.pageInfo.endCursor ?? null,
  };
}

export type CarrierIntelEventCarrierSummary = NonNullable<
  CarrierIntelEventCarrierSummaryQuery["carrier"]
>;

export async function fetchCarrierIntelEventCarrierSummary(
  carrierId: string,
  options?: { signal?: AbortSignal },
): Promise<CarrierIntelEventCarrierSummary | null> {
  const data = await requestGraphQL({
    document: CarrierIntelEventCarrierSummaryDocument,
    operationName: "CarrierIntelEventCarrierSummary",
    variables: { carrierId },
    signal: options?.signal,
  });
  return data.carrier;
}
