import { LocationTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const locationTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: LocationTableDocument,
  operationName: "LocationTable",
  connectionKey: "locations",
});

export type LocationRow = DataTableConfigRow<typeof locationTableGraphQLConfig>;
