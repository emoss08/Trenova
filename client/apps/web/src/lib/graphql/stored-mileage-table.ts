import { StoredMileageTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const storedMileageTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: StoredMileageTableDocument,
  operationName: "StoredMileageTable",
  connectionKey: "storedMileages",
});

export type StoredMileageRow = DataTableConfigRow<typeof storedMileageTableGraphQLConfig>;
