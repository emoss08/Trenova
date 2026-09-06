import { FleetCodeTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const fleetCodeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FleetCodeTableDocument,
  operationName: "FleetCodeTable",
  connectionKey: "fleetCodes",
});

export type FleetCodeRow = DataTableConfigRow<typeof fleetCodeTableGraphQLConfig>;
