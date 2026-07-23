import {
  UserTableDocument,
  type UserTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { User } from "@/types/user";

export const userTableGraphQLConfig = defineDataTableGraphQLConfig<
  User,
  UserTableQueryVariables
>({
  document: UserTableDocument,
  operationName: "UserTable",
  connectionKey: "users",
});
