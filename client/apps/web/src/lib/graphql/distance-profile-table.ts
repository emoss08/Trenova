import {
  DistanceProfileTableDocument,
} from "@trenova/graphql/generated/documents/distance-profile/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const distanceProfileTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DistanceProfileTableDocument,
  operationName: "DistanceProfileTable",
  connectionKey: "distanceProfiles",
});

export type DistanceProfileRow = DataTableConfigRow<typeof distanceProfileTableGraphQLConfig>;
