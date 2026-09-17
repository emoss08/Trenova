import {
  CarrierIntelEventTableDocument,
  CarrierMonitoringEnrollmentTableDocument,
  type CarrierIntelEventFilterInput,
  type CarrierMonitoringEnrollmentFilterInput,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const CARRIER_INTEL_EVENT_LIST_KEY = "carrier-intel-event-list";
export const CARRIER_MONITORING_ENROLLMENT_LIST_KEY = "carrier-monitoring-enrollment-list";

export function carrierIntelEventTableGraphQLConfig(filter: CarrierIntelEventFilterInput | null) {
  return defineDataTableGraphQLConfig({
    document: CarrierIntelEventTableDocument,
    operationName: "CarrierIntelEventTable",
    connectionKey: "carrierIntelEvents",
    extraVariables: filter ? { filter } : undefined,
  });
}

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

export type CarrierIntelEventRow = DataTableConfigRow<
  ReturnType<typeof carrierIntelEventTableGraphQLConfig>
>;
export type CarrierMonitoringEnrollmentRow = DataTableConfigRow<
  ReturnType<typeof carrierMonitoringEnrollmentTableGraphQLConfig>
>;
