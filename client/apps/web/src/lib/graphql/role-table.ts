import {
  RoleTableDocument,
  type RoleTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { Role } from "@/types/role";

export const roleTableGraphQLConfig = defineDataTableGraphQLConfig<
  Role,
  RoleTableQueryVariables
>({
  document: RoleTableDocument,
  operationName: "RoleTable",
  connectionKey: "roles",
});
