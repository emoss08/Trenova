import { DistanceOverrideTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const distanceOverrideTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DistanceOverrideTableDocument,
  operationName: "DistanceOverrideTable",
  connectionKey: "distanceOverrides",
});

export type DistanceOverrideRow = DataTableConfigRow<typeof distanceOverrideTableGraphQLConfig>;
