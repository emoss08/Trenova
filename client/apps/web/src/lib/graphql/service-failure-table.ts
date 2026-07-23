import {
  ServiceFailureTableDocument,
  type ServiceFailureTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { ServiceFailure } from "@/types/service-failure";

export function createServiceFailureTableGraphQLConfig(shipmentId?: string) {
  return defineDataTableGraphQLConfig<ServiceFailure, ServiceFailureTableQueryVariables>({
    document: ServiceFailureTableDocument,
    operationName: "ServiceFailureTable",
    connectionKey: "serviceFailures",
    extraVariables: shipmentId ? { shipmentId } : undefined,
  });
}
