import { UserTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const userTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: UserTableDocument,
  operationName: "UserTable",
  connectionKey: "users",
});

export type UserRow = DataTableConfigRow<typeof userTableGraphQLConfig>;
