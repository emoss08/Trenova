import {
  FleetCodeTableDocument,
  type FleetCodeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { FleetCode } from "@/types/fleet-code";

export const fleetCodeTableGraphQLConfig = defineDataTableGraphQLConfig<
  FleetCode,
  FleetCodeTableQueryVariables
>({
  document: FleetCodeTableDocument,
  operationName: "FleetCodeTable",
  connectionKey: "fleetCodes",
});
