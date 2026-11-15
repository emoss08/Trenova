import { AccountTypeTableDocument } from "@trenova/graphql/generated/documents/account-type/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const accountTypeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AccountTypeTableDocument,
  operationName: "AccountTypeTable",
  connectionKey: "accountTypes",
});

export type AccountTypeRow = DataTableConfigRow<typeof accountTypeTableGraphQLConfig>;
