import {
  FleetCodeTableDocument,
  type FleetCodeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { FleetCode } from "@trenova/shared/types/fleet-code";

export const fleetCodeTableGraphQLConfig = defineDataTableGraphQLConfig<
  FleetCode,
  FleetCodeTableQueryVariables
>({
  document: FleetCodeTableDocument,
  operationName: "FleetCodeTable",
  connectionKey: "fleetCodes",
});
