import { RoleTableDocument } from "@trenova/graphql/generated/documents/role/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const roleTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RoleTableDocument,
  operationName: "RoleTable",
  connectionKey: "roles",
});

export type RoleRow = DataTableConfigRow<typeof roleTableGraphQLConfig>;
