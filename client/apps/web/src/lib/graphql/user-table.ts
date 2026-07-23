import {
  UserTableDocument,
  type UserTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { User } from "@trenova/shared/types/user";

export const userTableGraphQLConfig = defineDataTableGraphQLConfig<
  User,
  UserTableQueryVariables
>({
  document: UserTableDocument,
  operationName: "UserTable",
  connectionKey: "users",
});
