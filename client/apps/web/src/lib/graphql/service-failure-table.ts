import { ServiceFailureTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export function createServiceFailureTableGraphQLConfig(shipmentId?: string) {
  return defineDataTableGraphQLConfig({
    document: ServiceFailureTableDocument,
    operationName: "ServiceFailureTable",
    connectionKey: "serviceFailures",
    extraVariables: shipmentId ? { shipmentId } : undefined,
  });
}

export type ServiceFailureRow = DataTableConfigRow<
  ReturnType<typeof createServiceFailureTableGraphQLConfig>
>;
