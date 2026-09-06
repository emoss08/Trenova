import { CustomerTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const customerTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: CustomerTableDocument,
  operationName: "CustomerTable",
  connectionKey: "customers",
});

export type CustomerRow = DataTableConfigRow<typeof customerTableGraphQLConfig>;
