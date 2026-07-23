import {
  RoleTableDocument,
  type RoleTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { Role } from "@trenova/shared/types/role";

export const roleTableGraphQLConfig = defineDataTableGraphQLConfig<
  Role,
  RoleTableQueryVariables
>({
  document: RoleTableDocument,
  operationName: "RoleTable",
  connectionKey: "roles",
});
